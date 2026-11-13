package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"testing"
)

func TestConfirmedCraftFeedsWeaponAppearanceOnly(t *testing.T) {
	a := inputApp()
	a.state.SelfID = 1
	a.state.InventoryReady = true
	a.items = map[sdata.ItemKey]*sdata.Item{{Type: 1, TypeID: 1}: {Model: 2}}
	c := proto.CharacterSlot{ID: 1, EquipType: [8]uint8{5: 1}, EquipTypeID: [8]uint8{5: 1}}
	for _, tc := range []struct {
		craft string
		want  uint8
	}{{"00000000000000000004", 4}, {"00000000000000000005", 5}, {"00000000000000000010", 10}, {"00000000000000000049", 49}, {"00000000000000000055", 0}, {"bad", 0}} {
		a.state.Inventory[[2]uint8{0, 5}] = proto.InventoryItem{Type: 1, TypeID: 1, Craft: tc.craft}
		if got := a.appearance(&c).WeaponEnchant; got != tc.want {
			t.Fatal(tc, got)
		}
	}
	c.ID = 2
	if a.appearance(&c).WeaponEnchant != 0 {
		t.Fatal("local craft leaked into another character")
	}
}
