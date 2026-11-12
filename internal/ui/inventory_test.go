package ui

import "testing"

func TestInventoryNativeCellsAndGaps(t *testing.T) {
	for _, tc := range []struct {
		bag, slot  uint8
		x, y, size int
	}{
		{1, 0, 15, 283, 32}, {5, 23, 210, 400, 32},
		{0, 0, 24, 53, 32}, {0, 5, 196, 133, 32}, {0, 8, 191, 53, 16}, {0, 13, 215, 53, 16},
	} {
		x, y, size, ok := InventoryCell(tc.bag, tc.slot)
		if !ok || x != tc.x || y != tc.y || size != tc.size {
			t.Fatalf("cell %d/%d=%d,%d,%d", tc.bag, tc.slot, x, y, size)
		}
		// Equipment is always visible, regardless of the selected bag.
		bag := tc.bag
		if bag == 0 {
			bag = 3
		}
		key, ok := InventoryHit(x+size-1, y+size-1, bag)
		if !ok || key != [2]uint8{tc.bag, tc.slot} {
			t.Fatalf("hit=%v %v", key, ok)
		}
	}
	for _, p := range [][2]int{{47, 283}, {15, 315}, {250, 400}, {190, 52}} {
		if _, ok := InventoryHit(p[0], p[1], 1); ok {
			t.Fatalf("gap %v selected an item", p)
		}
	}
	if _, _, _, ok := InventoryCell(0, 14); ok {
		t.Fatal("invented equipment slot")
	}
	if _, _, _, ok := InventoryCell(1, 24); ok {
		t.Fatal("invented bag slot")
	}
}
