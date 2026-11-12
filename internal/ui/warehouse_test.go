package ui

import "testing"

func TestWarehouseAllTabsKeepGlobalSlots(t *testing.T) {
	seen := map[uint8]bool{}
	for page := 0; page < 6; page++ {
		for i := 0; i < 40; i++ {
			slot := uint8(page*40 + i)
			x, y, ok := WarehouseCell(page, slot)
			hit, hitOK := WarehouseHit(x+1, y+1, page)
			if !ok || !hitOK || hit != slot || x+32 > WarehouseWidth || y+32 > WarehouseHeight || seen[slot] {
				t.Fatal(page, slot, x, y, hit)
			}
			seen[slot] = true
			if _, ok := WarehouseHit(x+34, y+1, page); ok {
				t.Fatal("gap treated as an item cell")
			}
		}
	}
	if len(seen) != 240 {
		t.Fatal(len(seen))
	}
}
