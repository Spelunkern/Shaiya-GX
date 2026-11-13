package db

import "fmt"

// ApplyGMItemGrant atomically persists a planned grant. Existing items may
// only change count; new items receive unique IDs before becoming visible.
// The caller installs its inventory copy only after this transaction succeeds.
func ApplyGMItemGrant(charID uint32, changed, added []Item) error {
	pool, err := Pool(GameData)
	if err != nil {
		return err
	}
	c, cancel := ctx()
	defer cancel()
	tx, err := pool.BeginTx(c, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, it := range changed {
		result, e := tx.ExecContext(c, `UPDATE PS_GameData_CharItems SET [Count]=? WHERE CharID=? AND ItemUID=? AND Bag=? AND Slot=?`, it.Count, charID, it.UID, it.Bag, it.Slot)
		if e != nil {
			return e
		}
		n, e := result.RowsAffected()
		if e != nil {
			return e
		}
		if n != 1 {
			return fmt.Errorf("GM grant: inventory item changed")
		}
	}
	for _, it := range added {
		if it.UID == 0 || it.Bag == 0 || it.Count == 0 {
			return fmt.Errorf("invalid GM grant item")
		}
		_, err = tx.ExecContext(c, `INSERT INTO PS_GameData_CharItems
(CharID,Bag,Slot,ItemID,Type,TypeID,ItemUID,Quality,Gem1,Gem2,Gem3,Gem4,Gem5,Gem6,Craftname,[Count],MakeTime,MakeType)
VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, charID, it.Bag, it.Slot, it.ItemID(), it.Type, it.TypeID, it.UID, it.Quality,
			it.Gems[0], it.Gems[1], it.Gems[2], it.Gems[3], it.Gems[4], it.Gems[5], it.CraftName, it.Count, makeTime(&it), makeType(&it))
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}
