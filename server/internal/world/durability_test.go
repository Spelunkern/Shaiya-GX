package world

import (
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"testing"
)

func TestLegacyQualityNeverReducesEquipmentStats(t *testing.T) {
	db.SetDefsForTest(&db.Defs{Items: map[db.ItemKey]*db.ItemDef{
		{Type: 1, TypeID: 1}:  {Type: 1, TypeID: 1, Quality: 100, Effect1: 41, Effect2: 10},
		{Type: 19, TypeID: 1}: {Type: 19, TypeID: 1, Quality: 100, Effect3: 30, Effect4: 20},
	}})
	for _, quality := range []uint16{0, 1, 10, 50, 100} {
		items := []db.Item{{Bag: 0, Slot: weaponSlot, Type: 1, TypeID: 1, Quality: quality}, {Bag: 0, Slot: 6, Type: 19, TypeID: 1, Quality: quality}}
		stats := GetEquipmentStats(items)
		if stats.ItemAttack != 41 || stats.Variance != 10 || stats.Defense != 30 || stats.Resistance != 20 {
			t.Fatalf("legacy quality %d changed stats: %+v", quality, stats)
		}
	}
}
