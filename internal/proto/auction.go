package proto

import (
	"encoding/json"
	"errors"
)

const GXAuction Opcode = 0x7F1F
const AuctionBoard = 117
const AuctionPageSize = 8

const (
	AuctionAnyFaction uint8 = iota
	AuctionLight
	AuctionFury
)

const (
	AuctionBrowse = iota
	AuctionSelling
	AuctionBidding
	AuctionItems
	AuctionGold
	AuctionWatch
)

var auctionHours = [...]int64{7, 24, 72, 168}

func AuctionHours(duration uint8) int64 {
	if int(duration) >= len(auctionHours) {
		return 0
	}
	return auctionHours[duration]
}

func AuctionDeposit(start uint32, duration uint8) uint32 {
	return uint32(AuctionHours(duration)) * 5 * uint32((uint64(start)+499999)/500000)
}

type AuctionRequest struct {
	Item                         *InventoryItem
	Sequence                     uint64
	Sort                         int
	Action                       string
	NPC                          uint32
	ID, Version                  int64
	View, Page                   int
	Query                        string
	Category, MinLevel, MaxLevel int
	Subcategory                  int    `json:",omitempty"`
	Faction                      uint8  `json:",omitempty"`
	NameGrade                    *uint8 `json:",omitempty"`
	ItemType, ItemID             uint8
	Duration                     uint8
	Count                        uint16
	Start, Buyout, Amount        uint32
	Source                       string
}

type AuctionRow struct {
	ID, Version                 int64
	Name, Seller, Craft, Reason string
	Type, TypeID                uint8
	Level, Count, Quality       uint16
	Gems                        [6]uint8
	Price, Buyout, Gold         uint32
	Ends                        int64
	Active, Mine, HighBidder    bool
}

type AuctionReply struct {
	Sequence       uint64
	NPC            uint32
	View, Page     int
	More           bool
	Error, Message string
	Rows           []AuctionRow
}

func AuctionPacket(value any) []byte {
	body, err := json.Marshal(value)
	if err != nil || len(body) > 12000 {
		return nil
	}
	w := NewWriter(GXAuction)
	w.Raw(body)
	return w.Bytes()
}

func DecodeAuction(p []byte, limit int, value any) error {
	if len(p) < 3 || len(p) > limit || Op(p) != GXAuction {
		return errors.New("invalid auction packet")
	}
	return json.Unmarshal(p[2:], value)
}

var AuctionCategories = [...]string{"All", "Weapons", "Armor", "Accessories", "Lapis", "Miscellaneous"}

func AuctionCategory(realType, itemType uint8) int {
	switch {
	case realType >= 1 && realType <= 15:
		return 1
	case realType >= 16 && realType <= 21 || realType == 24:
		return 2
	case realType == 22 || realType == 23 || realType == 25:
		return 3
	case itemType == 30 || itemType == 95:
		return 4
	default:
		return 5
	}
}

// AuctionSubcategories follows the GX catalog's native item families.
var AuctionSubcategories = [...][]struct {
	Name  string
	Types []uint8
}{
	nil,
	{
		{"One-handed", []uint8{1, 3, 45, 47}},
		{"Two-handed", []uint8{2, 4, 46, 48}},
		{"Dual weapons", []uint8{5, 49, 50}},
		{"Spears", []uint8{6, 51, 52}},
		{"Heavy weapons", []uint8{7, 8, 53, 54, 55, 56}},
		{"Knuckles", []uint8{9, 15, 57, 65}},
		{"Daggers", []uint8{10, 58}},
		{"Staves", []uint8{12, 60, 61}},
		{"Bows", []uint8{13, 62, 63}},
		{"Projectiles", []uint8{11, 14, 59, 64}},
	},
	{
		{"Headgear", []uint8{16, 31, 66, 72, 81, 87}},
		{"Upper armor", []uint8{17, 32, 67, 73, 82, 88}},
		{"Lower armor", []uint8{18, 33, 68, 74, 83, 89}},
		{"Shields", []uint8{19, 34, 69, 75, 84, 90}},
		{"Gloves", []uint8{20, 35, 70, 76, 85, 91}},
		{"Boots", []uint8{21, 36, 71, 77, 86, 92}},
		{"Cloaks", []uint8{24, 39}},
	},
	{
		{"Rings", []uint8{22, 37}},
		{"Necklaces", []uint8{23, 38}},
		{"Bracelets", []uint8{40, 41}},
	},
	{
		{"Lapis", []uint8{30}},
		{"Lapisians", []uint8{95}},
	},
	{
		{"Consumables", []uint8{25, 44, 100}},
		{"Mounts", []uint8{42}},
	},
}

func (r AuctionRequest) ValidSearchFilters() bool {
	return r.Category >= 0 && r.Category < len(AuctionSubcategories) &&
		r.Subcategory >= 0 && r.Subcategory <= len(AuctionSubcategories[r.Category]) &&
		r.Faction <= AuctionFury && (r.NameGrade == nil || *r.NameGrade <= 5 || *r.NameGrade == 7)
}
