package app

import "testing"

func TestInventoryEquipmentTypesAndNoZeroPrice(t *testing.T) {
	for _, typ := range []uint8{1, 10, 15, 45, 58, 65} {
		if got := inventoryEquipSlots(typ); len(got) != 1 || got[0] != 5 {
			t.Fatal(typ, got)
		}
	}
	for typ, want := range map[uint8]uint8{16: 0, 17: 1, 18: 2, 20: 3, 21: 4, 34: 6, 24: 7, 23: 8, 42: 13, 66: 0, 92: 4} {
		if got := inventoryEquipSlots(typ); len(got) != 1 || got[0] != want {
			t.Fatal(typ, got)
		}
	}
	for _, typ := range []uint8{0, 25, 95, 100, 255} {
		if len(inventoryEquipSlots(typ)) != 0 {
			t.Fatal("consumable became equipment", typ)
		}
	}
	if itemSellingPrice(0) != "" || itemSellingPrice(17) != "Sell: 17 gold" {
		t.Fatal("selling price presentation")
	}
	if debuffFlash(.25) != .75 || debuffFlash(1.25) != .75 {
		t.Fatal("debuff pulse must use elapsed seconds")
	}
}
