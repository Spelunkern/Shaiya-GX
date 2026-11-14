package db

import (
	"crypto/rand"
	"database/sql"
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Spelunkern/Shaiya-GX/server/internal/config"
)

// Item is one CharItems or UserStoredItems row; UID (ItemUID) is the key
// the original procedures use.
type Item struct {
	UID          int64
	Bag, Slot    uint8
	Type, TypeID uint8
	Quality      uint16
	Gems         [6]uint8
	Count        uint16
	CraftName    string
	MakeTime     time.Time
	MakeType     string
}

// ItemID is Type*1000+TypeID, the original encoding.
func (it *Item) ItemID() int32 { return int32(it.Type)*1000 + int32(it.TypeID) }

// NewItemUID returns a random positive 63-bit ItemUID.
func NewItemUID() int64 {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	return int64(binary.LittleEndian.Uint64(b[:]) & 0x7FFF_FFFF_FFFF_FFFF)
}

func scanItems(rows *sql.Rows, stored bool) ([]Item, error) {
	defer rows.Close()
	var items []Item
	for rows.Next() {
		var it Item
		var itemID int32
		var quality int64
		var slot int64
		var craft, makeType sql.NullString
		var mk [6]sql.NullInt32
		dest := []any{&itemID, &it.Type, &it.TypeID, &it.UID}
		if !stored {
			dest = append(dest, &it.Bag)
		}
		dest = append(dest, &slot, &quality, &it.Gems[0], &it.Gems[1], &it.Gems[2], &it.Gems[3],
			&it.Gems[4], &it.Gems[5], &craft, &it.Count, &makeType, &mk[0], &mk[1], &mk[2], &mk[3], &mk[4], &mk[5])
		if err := rows.Scan(dest...); err != nil {
			return nil, err
		}
		it.Slot, it.Quality = uint8(min(max(slot, 0), 255)), clampU16(quality)
		it.CraftName, it.MakeType = strings.TrimRight(craft.String, " "), strings.TrimSpace(makeType.String)
		if mk[0].Valid {
			it.MakeTime = time.Date(int(mk[0].Int32), time.Month(mk[1].Int32), int(mk[2].Int32),
				int(mk[3].Int32), int(mk[4].Int32), int(mk[5].Int32), 0, time.UTC)
		}
		if it.Count > 0 {
			items = append(items, it)
		}
	}
	return items, rows.Err()
}

// CharacterItems runs usp_Read_Char_Items_Detail_R.
func CharacterItems(charID uint32) ([]Item, error) {
	pool, err := Pool(GameData)
	if err != nil {
		return nil, err
	}
	c, cancel := ctx()
	defer cancel()
	rows, err := pool.QueryContext(c, `SELECT ItemID, Type, TypeID, ItemUID, Bag, Slot, Quality,
Gem1, Gem2, Gem3, Gem4, Gem5, Gem6, Craftname, [Count],  Maketype ,
CAST(strftime('%Y',MakeTime) AS INTEGER) AS MakeTime_YYYY,
CAST(strftime('%m',MakeTime) AS INTEGER) AS MakeTime_MM,
CAST(strftime('%d',MakeTime) AS INTEGER) AS MakeTime_DD,
CAST(strftime('%H',MakeTime) AS INTEGER) AS MakeTime_HH,
CAST(strftime('%M',MakeTime) AS INTEGER) AS MakeTime_MI,
CAST(strftime('%S',MakeTime) AS INTEGER) AS MakeTime_SS

FROM PS_GameData_CharItems

WHERE CharID = @charid AND Del = 0`, sql.Named("charid", int32(charID)))
	if err != nil {
		return nil, fmt.Errorf("usp_Read_Char_Items_Detail_R: %w", err)
	}
	return scanItems(rows, false)
}

func makeTime(it *Item) string {
	t := it.MakeTime
	if t.IsZero() {
		t = time.Now()
	}
	return t.UTC().Format("2006-01-02 15:04:05")
}

func makeType(it *Item) string {
	if it.MakeType == "" {
		return "S"
	}
	return it.MakeType[:1]
}

// AddCharacterItem runs usp_Save_Char_Item_Add_E; a zero UID gets a new one.
func AddCharacterItem(charID uint32, it *Item) error {
	if it.UID == 0 {
		it.UID = NewItemUID()
	}
	if it.MakeTime.IsZero() {
		it.MakeTime = time.Now()
	}
	return execSQL(GameData, `INSERT INTO PS_GameData_CharItems
(CharID, bag, slot, ItemID, Type, TypeID, ItemUID, quality, gem1, gem2, gem3, gem4,
gem5, gem6, craftname, [count], maketime, maketype)
VALUES(@charid, @bag, @slot, @itemid, @type, @typeid, @itemuid, @quality, @gem1, @gem2, @gem3, @gem4,
@gem5, @gem6, @craftname, @count, @maketime, @maketype)`, sql.Named("charid", int32(charID)),
		sql.Named("itemuid", it.UID), sql.Named("bag", it.Bag), sql.Named("slot", it.Slot),
		sql.Named("itemid", it.ItemID()), sql.Named("type", it.Type), sql.Named("typeid", it.TypeID),
		sql.Named("quality", int32(it.Quality)), sql.Named("gem1", it.Gems[0]), sql.Named("gem2", it.Gems[1]),
		sql.Named("gem3", it.Gems[2]), sql.Named("gem4", it.Gems[3]), sql.Named("gem5", it.Gems[4]),
		sql.Named("gem6", it.Gems[5]), sql.Named("craftname", it.CraftName), sql.Named("count", it.Count),
		sql.Named("maketime", makeTime(it)), sql.Named("maketype", makeType(it)))
}

// SaveCharacterItem writes bag, slot, quality, gems, count and craft name
// with usp_Save_Char_Item_Mod_E. A zero count deletes the item.
func SaveCharacterItem(charID uint32, it *Item) error {
	if it.Count == 0 {
		return DeleteCharacterItems(charID, it.UID)
	}
	return execSQL(GameData, `UPDATE PS_GameData_CharItems
SET Bag=@bag, Slot=@slot, Quality=@quality, Gem1=@gem1, Gem2=@gem2, Gem3=@gem3, Gem4=@gem4, Gem5=@gem5, Gem6=@gem6,
[Count]=@count, Craftname=@craftname
WHERE CharID=@charid AND ItemUID = @itemuid`, sql.Named("charid", int32(charID)),
		sql.Named("itemuid", it.UID), sql.Named("bag", it.Bag), sql.Named("slot", it.Slot),
		sql.Named("quality", int16(min(it.Quality, 32767))), sql.Named("gem1", it.Gems[0]),
		sql.Named("gem2", it.Gems[1]), sql.Named("gem3", it.Gems[2]), sql.Named("gem4", it.Gems[3]),
		sql.Named("gem5", it.Gems[4]), sql.Named("gem6", it.Gems[5]), sql.Named("craftname", it.CraftName),
		sql.Named("count", it.Count))
}

func idList(uids []int64) string {
	parts := make([]string, len(uids))
	for i, uid := range uids {
		parts[i] = strconv.FormatInt(uid, 10)
	}
	return strings.Join(parts, ",")
}

// DeleteCharacterItems runs usp_Save_Char_Item_Del_E for the given UIDs.
func DeleteCharacterItems(charID uint32, uids ...int64) error {
	if len(uids) == 0 {
		return nil
	}
	return execSQL(GameData, `DELETE FROM PS_GameData_CharItems WHERE CharID=@charid AND ItemUID IN (SELECT value FROM json_each('['||@idlist||']'))`, sql.Named("charid", int32(charID)),
		sql.Named("idlist", idList(uids)))
}

// StoredItems runs usp_Read_User_StoredItems_Detail_R (account warehouse).
func StoredItems(userUID uint32) ([]Item, error) {
	pool, err := Pool(GameData)
	if err != nil {
		return nil, err
	}
	c, cancel := ctx()
	defer cancel()
	rows, err := pool.QueryContext(c, `SELECT ItemID, Type, TypeID, ItemUID, Slot, Quality,
Gem1, Gem2, Gem3, Gem4, Gem5, Gem6, Craftname, [Count], Maketype,
CAST(strftime('%Y',MakeTime) AS INTEGER) AS MakeTime_YYYY,
CAST(strftime('%m',MakeTime) AS INTEGER) AS MakeTime_MM,
CAST(strftime('%d',MakeTime) AS INTEGER) AS MakeTime_DD,
CAST(strftime('%H',MakeTime) AS INTEGER) AS MakeTime_HH,
CAST(strftime('%M',MakeTime) AS INTEGER) AS MakeTime_MI,
CAST(strftime('%S',MakeTime) AS INTEGER) AS MakeTime_SS

FROM PS_GameData_UserStoredItems

WHERE ServerID=@serverid AND UserUID=@useruid

ORDER BY MakeTime DESC`,
		sql.Named("serverid", config.ServerID), sql.Named("useruid", int32(userUID)))
	if err != nil {
		return nil, fmt.Errorf("usp_Read_User_StoredItems_Detail_R: %w", err)
	}
	return scanItems(rows, true)
}

// AddStoredItem runs usp_Save_User_StoredItem_Add_E.
func AddStoredItem(userUID uint32, it *Item) error {
	if it.UID == 0 {
		it.UID = NewItemUID()
	}
	return execSQL(GameData, `INSERT INTO PS_GameData_UserStoredItems
(ServerID, UserUID, Slot, ItemID, Type, TypeID, ItemUID, quality, gem1, gem2, gem3, gem4,
gem5, gem6, craftname, [count], maketime, maketype)
VALUES(@serverid, @useruid, @slot, @itemid, @type, @typeid, @itemuid, @quality, @gem1, @gem2, @gem3, @gem4,
@gem5, @gem6, @craftname, @count, @maketime, @maketype)`, sql.Named("serverid", config.ServerID),
		sql.Named("useruid", int32(userUID)), sql.Named("itemuid", it.UID), sql.Named("slot", it.Slot),
		sql.Named("itemid", it.ItemID()), sql.Named("type", it.Type), sql.Named("typeid", it.TypeID),
		sql.Named("quality", int16(min(it.Quality, 32767))), sql.Named("gem1", it.Gems[0]),
		sql.Named("gem2", it.Gems[1]), sql.Named("gem3", it.Gems[2]), sql.Named("gem4", it.Gems[3]),
		sql.Named("gem5", it.Gems[4]), sql.Named("gem6", it.Gems[5]), sql.Named("craftname", it.CraftName),
		sql.Named("count", it.Count), sql.Named("maketime", makeTime(it)), sql.Named("maketype", makeType(it)))
}

// SaveStoredItem runs usp_Save_User_StoredItem_Mod_E; zero count deletes.
func SaveStoredItem(userUID uint32, it *Item) error {
	if it.Count == 0 {
		return DeleteStoredItems(userUID, it.UID)
	}
	return execSQL(GameData, `UPDATE PS_GameData_UserStoredItems
SET Slot=@slot,Quality=@quality,Gem1=@gem1,Gem2=@gem2,Gem3=@gem3,Gem4=@gem4,Gem5=@gem5,Gem6=@gem6,
Craftname=@craftname,[Count]=@count
WHERE ServerID=@serverid AND UserUID=@useruid AND ItemUID=@itemuid`, sql.Named("serverid", config.ServerID),
		sql.Named("useruid", int32(userUID)), sql.Named("itemuid", it.UID), sql.Named("slot", it.Slot),
		sql.Named("quality", int16(min(it.Quality, 32767))), sql.Named("gem1", it.Gems[0]),
		sql.Named("gem2", it.Gems[1]), sql.Named("gem3", it.Gems[2]), sql.Named("gem4", it.Gems[3]),
		sql.Named("gem5", it.Gems[4]), sql.Named("gem6", it.Gems[5]), sql.Named("count", it.Count),
		sql.Named("craftname", it.CraftName))
}

// DeleteStoredItems runs usp_Save_User_StoredItem_Del_E.
func DeleteStoredItems(userUID uint32, uids ...int64) error {
	if len(uids) == 0 {
		return nil
	}
	return execSQL(GameData, `DELETE FROM PS_GameData_UserStoredItems WHERE ServerID=@serverid AND UserUID=@useruid AND ItemUID IN (SELECT value FROM json_each('['||@idlist||']'))`, sql.Named("serverid", config.ServerID),
		sql.Named("useruid", int32(userUID)), sql.Named("idlist", idList(uids)))
}

// GuildStoredItems runs usp_Read_Guild_StoredItems_Detail_R.
func GuildStoredItems(guildID uint32) ([]Item, error) {
	pool, err := Pool(GameData)
	if err != nil {
		return nil, err
	}
	c, cancel := ctx()
	defer cancel()
	rows, err := pool.QueryContext(c, `SELECT ItemID, Type, TypeID, ItemUID, Slot, Quality,
Gem1, Gem2, Gem3, Gem4, Gem5, Gem6, Craftname, [Count], Maketype,
CAST(strftime('%Y',MakeTime) AS INTEGER) AS MakeTime_YYYY,
CAST(strftime('%m',MakeTime) AS INTEGER) AS MakeTime_MM,
CAST(strftime('%d',MakeTime) AS INTEGER) AS MakeTime_DD,
CAST(strftime('%H',MakeTime) AS INTEGER) AS MakeTime_HH,
CAST(strftime('%M',MakeTime) AS INTEGER) AS MakeTime_MI,
CAST(strftime('%S',MakeTime) AS INTEGER) AS MakeTime_SS

FROM PS_GameData_GuildStoredItems

WHERE GuildID=@guildid AND Del = 0

ORDER BY MakeTime DESC`, sql.Named("guildid", int32(guildID)))
	if err != nil {
		return nil, fmt.Errorf("usp_Read_Guild_StoredItems_Detail_R: %w", err)
	}
	return scanItems(rows, true)
}

// AddGuildStoredItem runs usp_Save_Guild_StoredItem_Add_E.
func AddGuildStoredItem(guildID uint32, it *Item) error {
	if it.UID == 0 {
		it.UID = NewItemUID()
	}
	return execSQL(GameData, `INSERT INTO PS_GameData_GuildStoredItems
(GuildID, Slot, ItemID, Type, TypeID, ItemUID, quality, gem1, gem2, gem3, gem4,
gem5, gem6, craftname, [count], maketime, maketype)
VALUES(@guildid, @slot, @itemid, @type, @typeid, @itemuid, @quality, @gem1, @gem2, @gem3, @gem4,
@gem5, @gem6, @craftname, @count, @maketime, @maketype)`, sql.Named("guildid", int32(guildID)),
		sql.Named("itemuid", it.UID), sql.Named("slot", it.Slot), sql.Named("itemid", it.ItemID()),
		sql.Named("type", it.Type), sql.Named("typeid", it.TypeID), sql.Named("quality", int16(min(it.Quality, 32767))),
		sql.Named("gem1", it.Gems[0]), sql.Named("gem2", it.Gems[1]), sql.Named("gem3", it.Gems[2]),
		sql.Named("gem4", it.Gems[3]), sql.Named("gem5", it.Gems[4]), sql.Named("gem6", it.Gems[5]),
		sql.Named("craftname", it.CraftName), sql.Named("count", it.Count), sql.Named("maketime", makeTime(it)),
		sql.Named("maketype", makeType(it)))
}

// SaveGuildStoredItem runs usp_Save_Guild_StoredItem_Mod_E.
func SaveGuildStoredItem(guildID uint32, it *Item) error {
	return execSQL(GameData, `UPDATE PS_GameData_GuildStoredItems
SET Slot=@slot,Quality=@quality,Gem1=@gem1,Gem2=@gem2,Gem3=@gem3,Gem4=@gem4,Gem5=@gem5,Gem6=@gem6,
Craftname=@craftname,[Count]=@count
WHERE GuildID=@guildid AND ItemUID=@itemuid`, sql.Named("guildid", int32(guildID)),
		sql.Named("itemuid", it.UID), sql.Named("slot", it.Slot), sql.Named("quality", int16(min(it.Quality, 32767))),
		sql.Named("gem1", it.Gems[0]), sql.Named("gem2", it.Gems[1]), sql.Named("gem3", it.Gems[2]),
		sql.Named("gem4", it.Gems[3]), sql.Named("gem5", it.Gems[4]), sql.Named("gem6", it.Gems[5]),
		sql.Named("count", it.Count), sql.Named("craftname", it.CraftName))
}

// DeleteGuildStoredItems runs usp_Save_Guild_StoredItem_Del_E.
func DeleteGuildStoredItems(guildID uint32, uids ...int64) error {
	if len(uids) == 0 {
		return nil
	}
	return execSQL(GameData, `DELETE FROM PS_GameData_GuildStoredItems WHERE GuildID=@guildid AND ItemUID IN (SELECT value FROM json_each('['||@idlist||']'))`, sql.Named("guildid", int32(guildID)),
		sql.Named("idlist", idList(uids)))
}
