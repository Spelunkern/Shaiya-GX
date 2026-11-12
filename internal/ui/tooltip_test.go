package ui

import "testing"

func TestTooltipStaysInsideViewport(t *testing.T) {
	for _, p := range [][2]int{{5, 5}, {1023, 767}, {0, 760}, {700, 500}} {
		x, y := TooltipOrigin(p[0], p[1], 208, 240, 1024, 768)
		if x < 0 || y < 0 || x+208 > 1024 || y+240 > 768 {
			t.Fatal("tooltip outside viewport", p, x, y)
		}
		if p[0] == 5 && x != 21 {
			t.Fatal("tooltip did not switch to right", x)
		}
	}
}

func TestTooltipReferencePlacement(t *testing.T) {
	for _, item := range []bool{false, true} {
		x, y := TooltipOriginFor(5, 5, 208, 240, 1024, 768, item, 1)
		wantX := 21
		if item {
			wantX = 13
		}
		if x != wantX || y != 15 {
			t.Fatalf("item=%v: got (%d,%d), want (%d,15)", item, x, y, wantX)
		}
		x, y = TooltipOriginFor(95, 75, 208, 240, 100, 80, item, 1.0/3)
		if x < 0 || y < 0 || x+69 > 100 || y+80 > 80 {
			t.Fatalf("scaled tooltip outside viewport: %d,%d", x, y)
		}
	}
}
