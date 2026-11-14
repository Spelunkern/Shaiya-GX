package world

import (
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"testing"
)

func TestTargetedRecreationPreservesOtherStatsAndEnchantment(t *testing.T) {
	for stat := 0; stat < 9; stat++ {
		for _, mode := range []int{1, 2} {
			def := &db.ItemDef{Type: 16, Server: 9, ReqWis: 13}
			before := [10]uint32{1, 2, 3, 4, 5, 6, 7, 8, 9, 53}
			for range 100 {
				got, err := recreateWithRune(&db.ItemDef{ReqVg: int16(mode*100 + stat + 1)}, def, db.Item{Type: 16, CraftName: craftString(before)})
				if err != nil {
					t.Fatal(err)
				}
				c, ok := ParseCraftName(got)
				if !ok {
					t.Fatal(got)
				}
				pairs := craftPairs(c)
				for i, v := range pairs {
					if i != stat && v != before[i] {
						t.Fatalf("stat%d changed sibling %d", stat, i)
					}
				}
				if pairs[stat] < 1 || pairs[stat] > 13 || mode == 2 && pairs[stat] != 13 {
					t.Fatal("roll outside requested mode", pairs)
				}
			}
		}
	}
}

func TestTargetedRecreationRejectsInvalidPoolAndFullBudget(t *testing.T) {
	def := &db.ItemDef{Type: 1, Server: 1, ReqWis: 13}
	it := db.Item{Type: 1, CraftName: "01000000000000000002"}
	for _, vg := range []int16{102, 107, 108, 109, 207, 208, 209, 100, 110, 999, -1} {
		if _, err := recreateWithRune(&db.ItemDef{ReqVg: vg}, def, it); err == nil {
			t.Fatalf("accepted invalid rune %d", vg)
		}
	}
	if got, err := recreateWithRune(&db.ItemDef{ReqVg: 201}, def, it); err != nil || got != "13000000000000000002" {
		t.Fatal(got, err)
	}
	def.Server = 2
	if _, err := recreateWithRune(&db.ItemDef{ReqVg: 202}, def, it); err != nil {
		t.Fatal("free OJ budget refused", err)
	}
}

func TestTargetedRuneConsumptionPersistenceAndRejectedAttempt(t *testing.T) {
	s, add := forgeRuleSession(t)
	defs := db.MustDefs()
	defs.Items[db.ItemKey{Type: 101, TypeID: 10}] = &db.ItemDef{Type: 101, TypeID: 10, Special: 62, ReqVg: 201, Count: 255}
	defs.Items[db.ItemKey{Type: 101, TypeID: 16}] = &db.ItemDef{Type: 101, TypeID: 16, Special: 62, ReqVg: 207, Count: 255}
	target := add(db.Item{Bag: 1, Slot: 0, Type: 1, TypeID: 1, Count: 1, CraftName: "01020300000000000002"})
	rune := add(db.Item{Bag: 1, Slot: 1, Type: 101, TypeID: 10, Count: 255})
	bad := add(db.Item{Bag: 1, Slot: 2, Type: 101, TypeID: 16, Count: 255})
	s.handleItemCompose(proto.ComposePacket(1, 1, 1, 0))
	live := s.items[findItemByUID(s.items, target.UID)]
	if live.CraftName != "09020300000000000002" || s.items[findItemByUID(s.items, rune.UID)].Count != 254 {
		t.Fatal("targeted consume/roll failed", live)
	}
	s.handleItemCompose(proto.ComposePacket(1, 2, 1, 0))
	if s.items[findItemByUID(s.items, bad.UID)].Count != 255 || s.items[findItemByUID(s.items, target.UID)].CraftName != live.CraftName {
		t.Fatal("refusal consumed or changed an item")
	}
	saved, err := db.CharacterItems(s.char.ID)
	if err != nil || saved[findItemByUID(saved, target.UID)].CraftName != live.CraftName {
		t.Fatal("targeted craft not persisted", err)
	}
}

func TestEnchantConsumesOneMaterialEvenForLegacyCount(t *testing.T) {
	s, add := forgeRuleSession(t)
	add(db.Item{Bag: 1, Slot: 0, Type: 1, TypeID: 1, Count: 1})
	add(db.Item{Bag: 1, Slot: 1, Type: 95, TypeID: 1, Count: 255})
	mat := add(db.Item{Bag: 1, Slot: 2, Type: 25, TypeID: 43, Count: 255})
	s.handleEnchantAdd(proto.EnchantPacket(1, 1, 1, 2, 100, 1, 0))
	if s.items[findItemByUID(s.items, mat.UID)].Count != 254 {
		t.Fatal("enchant consumed more than one material")
	}
}

func TestEnchantQuoteShowsChanceBeforeConsumablesArePlaced(t *testing.T) {
	s, add := forgeRuleSession(t)
	add(db.Item{Bag: 1, Slot: 0, Type: 1, TypeID: 1, Count: 1})
	q := s.forgeQuote(proto.ForgeQuoteRequest{Kind: proto.ForgeQuoteEnchant, Item: [2]uint8{1, 0}, Lapis: [2]uint8{255, 255}, Material: [2]uint8{255, 255}, Hammer: [2]uint8{255, 255}, Count: 1})
	if q.Valid || q.Rate != enchantConfig.WeaponRate[0] || q.Material != 25043 {
		t.Fatal("preview must show rate/material without authorizing a mutation", q)
	}
}
