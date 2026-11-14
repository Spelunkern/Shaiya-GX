package world

import (
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"testing"
)

func TestRiskyLapisNeverDestroysEquipment(t *testing.T) {
	old := db.MustDefs()
	t.Cleanup(func() { db.SetDefsForTest(old) })
	db.SetDefsForTest(&db.Defs{Items: map[db.ItemKey]*db.ItemDef{
		{Type: 1, TypeID: 1}:   {Type: 1, TypeID: 1, Slot: 6},
		{Type: 30, TypeID: 1}:  {Type: 30, TypeID: 1, Slot: 1, ReqLevel: 1, ReqIg: 255, ReqVg: 1},
		{Type: 100, TypeID: 2}: {Type: 100, TypeID: 2, Special: specialGemExtract, ReqVg: 0},
		{Type: 100, TypeID: 1}: {Type: 100, TypeID: 1, Special: specialGemProtect},
	}})
	s, _ := gmWireSession(t, db.Character{ID: 999999, Level: 1, Money: 4000000000}, false)
	for _, extraction := range []bool{false, true} {
		failures := 0
		for range 100 {
			item := db.Item{UID: 1, Bag: 1, Slot: 0, Type: 1, TypeID: 1, Count: 1}
			if extraction {
				item.Gems[0] = 1
			}
			s.items = []db.Item{item, {UID: 2, Bag: 1, Slot: 1, Type: 30, TypeID: 1, Count: 2}, {UID: 3, Bag: 1, Slot: 2, Type: 100, TypeID: 1, Count: 1}}
			if extraction {
				s.items = append(s.items, db.Item{UID: 4, Bag: 1, Slot: 3, Type: 100, TypeID: 2, Count: 1})
				s.handleGemRemove(proto.GemRemoveHammerPacket([2]uint8{1, 0}, [2]uint8{1, 3}, 0))
			} else {
				s.handleGemAdd(proto.GemAddPacket(1, 1, 1, 0))
			}
			i := findItemByUID(s.items, 1)
			if i < 0 || s.items[i].Count != 1 {
				t.Fatal("equipment destroyed", extraction)
			}
			if s.items[i].Gems[0] == 0 {
				failures++
			}
			ci := findItemByUID(s.items, 3)
			if ci < 0 || s.items[ci].Count != 1 {
				t.Fatal("unnecessary charm consumption")
			}
		}
		if failures == 0 {
			t.Fatal("no failures exercised")
		}
	}
}
