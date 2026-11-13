package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// Account points, gifts and the game log share one SQLite file. Guarded balance
// updates and gift claims are transactional across the former logical databases.

const GiftBoxSlots = 240

// Points reads the account's item-mall balance.
func Points(userUID uint32) (uint32, error) {
	pool, err := Pool(UserData)
	if err != nil {
		return 0, err
	}
	c, cancel := ctx()
	defer cancel()
	var point sql.NullInt32
	err = pool.QueryRowContext(c, `SELECT Point FROM PS_UserData_Users_Master WHERE UserUID = ?1`, int32(userUID)).Scan(&point)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrNotFound
	}
	return uint32(max(point.Int32, 0)), err
}

// ErrNotEnoughPoints means the balance does not cover the cost.
var ErrNotEnoughPoints = errors.New("not enough points")

// SpendPoints guards the balance and records the original NCash PointLog in
// the same transaction. useType: 1 buy, 2 gift.
func SpendPoints(userUID, charID, cost uint32, productCode, targetName string, useType uint8) (uint32, error) {
	p, e := Pool(UserData)
	if e != nil {
		return 0, e
	}
	c, cancel := ctx()
	defer cancel()
	var remaining uint32
	e = withTx(c, p, func(tx *sql.Tx) error {
		res, e := tx.ExecContext(c, `UPDATE PS_UserData_Users_Master SET Point=Point-?2 WHERE UserUID=?1 AND Point>=?2`, int32(userUID), int32(cost))
		if e != nil {
			return e
		}
		if n, _ := res.RowsAffected(); n != 1 {
			return ErrNotEnoughPoints
		}
		if e = tx.QueryRowContext(c, `SELECT Point FROM PS_UserData_Users_Master WHERE UserUID=?1`, int32(userUID)).Scan(&remaining); e != nil {
			return e
		}
		_, e = tx.ExecContext(c, `INSERT INTO PS_GameData_PointLog(UseType,UserUID,CharID,UsePoint,ProductCode,TargetName,UseDate,RemainPoint) VALUES(?1,?2,?3,?4,?5,?6,CURRENT_TIMESTAMP,?7)`, useType, int32(userUID), int32(charID), int32(cost), productCode, targetName, int32(remaining))
		return e
	})
	return remaining, e
}

// GiftEntry is one gift-box slot.
type GiftEntry struct {
	Slot                uint8
	Type, TypeID, Count uint8
}

// GiftBox runs usp_Read_User_StoredPointItems.
func GiftBox(userUID uint32) ([]GiftEntry, error) {
	pool, err := Pool(GameData)
	if err != nil {
		return nil, err
	}
	c, cancel := ctx()
	defer cancel()
	rows, err := pool.QueryContext(c, `SELECT Slot,ItemID,ItemCount FROM PS_GameData_UserStoredPointItems WHERE UserUID=@useruid`, sql.Named("useruid", int32(userUID)))
	if err != nil {
		return nil, fmt.Errorf("usp_Read_User_StoredPointItems: %w", err)
	}
	defer rows.Close()
	var gifts []GiftEntry
	for rows.Next() {
		var slot, count uint8
		var itemID int32
		if err := rows.Scan(&slot, &itemID, &count); err != nil {
			return nil, err
		}
		if slot < GiftBoxSlots && itemID > 0 && count > 0 {
			gifts = append(gifts, GiftEntry{Slot: slot, Type: uint8(min(itemID/1000, 255)),
				TypeID: uint8(min(itemID%1000, 255)), Count: count})
		}
	}
	return gifts, rows.Err()
}

// GiftResult codes match the original usp_Save_User_GiftPointItems_NCash
// and the client's gift result byte.
const (
	GiftOK             = 0
	GiftNoSuchChar     = 1
	GiftSameAccount    = 2
	GiftBoxFull        = 3
	GiftNotEnoughSlots = 4
	GiftInsertError    = 5
)

// StoreGift puts every item of product into the gift box of the account
// owning targetName, in free slots, atomically.
func StoreGift(targetName string, senderUID uint32, product *Product) (result uint8, targetUID uint32, err error) {
	pool, err := Pool(GameData)
	if err != nil {
		return GiftInsertError, 0, err
	}
	c, cancel := ctx()
	defer cancel()
	var uid sql.NullInt32
	if err := pool.QueryRowContext(c, `SELECT UserUID FROM PS_GameData_Chars WHERE CharName = ?1 AND Del = 0 LIMIT 1`,
		targetName).Scan(&uid); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return GiftInsertError, 0, err
	}
	if !uid.Valid || uid.Int32 == 0 {
		return GiftNoSuchChar, 0, nil
	}
	targetUID = uint32(uid.Int32)
	if targetUID == senderUID {
		return GiftSameAccount, targetUID, nil
	}
	err = withTx(c, pool, func(tx *sql.Tx) error {
		used := map[uint8]bool{}
		rows, err := tx.QueryContext(c, `SELECT Slot FROM PS_GameData_UserStoredPointItems  WHERE UserUID = ?1`, int32(targetUID))
		if err != nil {
			return err
		}
		for rows.Next() {
			var slot uint8
			rows.Scan(&slot)
			used[slot] = true
		}
		rows.Close()
		if len(used) >= GiftBoxSlots {
			result = GiftBoxFull
			return errRollback
		}
		next := uint8(0)
		for _, item := range product.Items {
			for int(next) < GiftBoxSlots && used[next] {
				next++
			}
			if int(next) >= GiftBoxSlots {
				result = GiftNotEnoughSlots
				return errRollback
			}
			if _, err := tx.ExecContext(c, `INSERT INTO PS_GameData_UserStoredPointItems (UserUID, Slot, ItemID, ItemCount, BuyDate)
				VALUES (?1, ?2, ?3, ?4, CURRENT_TIMESTAMP)`, int32(targetUID), next,
				int32(item.Type)*1000+int32(item.TypeID), item.Count); err != nil {
				result = GiftInsertError
				return err
			}
			used[next] = true
		}
		return nil
	})
	if errors.Is(err, errRollback) {
		return result, targetUID, nil
	}
	if err != nil {
		return GiftInsertError, targetUID, err
	}
	return GiftOK, targetUID, nil
}

// ClaimGift moves gift-box slot into the character's inventory as it, in
// one transaction (the effect of usp_Save_User_StoredPointItems). It
// returns false if the slot was already claimed.
func ClaimGift(userUID, charID uint32, giftSlot uint8, it *Item) (bool, error) {
	pool, err := Pool(GameData)
	if err != nil {
		return false, err
	}
	c, cancel := ctx()
	defer cancel()
	if it.UID == 0 {
		it.UID = NewItemUID()
	}
	if it.MakeTime.IsZero() {
		it.MakeTime = time.Now()
	}
	claimed := false
	err = withTx(c, pool, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(c, `DELETE FROM PS_GameData_UserStoredPointItems WHERE UserUID = ?1 AND Slot = ?2`,
			int32(userUID), giftSlot)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n != 1 {
			return errRollback
		}
		if _, err := tx.ExecContext(c, `INSERT INTO PS_GameData_CharItems (CharID, Bag, Slot, ItemID, Type, TypeID, ItemUID,
			Quality, Gem1, Gem2, Gem3, Gem4, Gem5, Gem6, Craftname, Count, Maketime, Maketype)
			VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8, 0, 0, 0, 0, 0, 0, '', ?9, ?10, 'B')`,
			int32(charID), it.Bag, it.Slot, it.ItemID(), it.Type, it.TypeID, it.UID,
			int16(min(it.Quality, 4999)), it.Count, makeTime(it)); err != nil {
			return err
		}
		claimed = true
		return nil
	})
	if errors.Is(err, errRollback) {
		return false, nil
	}
	return claimed, err
}

var errRollback = errors.New("rollback")

func withTx(c context.Context, pool *sql.DB, fn func(*sql.Tx) error) error {
	tx, err := pool.BeginTx(c, nil)
	if err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}
