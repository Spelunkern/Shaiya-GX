package ui

import "testing"

func TestPlayerMenuNativeRowsExcludePadding(t *testing.T) {
	width, height := PlayerMenuSize(nil, []string{"Party", "Trade", "Whisper"})
	if width != 98 || height != 77 {
		t.Fatal("classic popup bounds", width, height)
	}
	for _, tc := range []struct{ x, y, row int }{{110, 210, 0}, {110, 224, 0}, {110, 225, 1}, {110, 254, 2}, {110, 255, -1}, {103, 212, -1}, {110, 205, -1}, {110, 270, -1}} {
		if got := PlayerMenuRowAt(tc.x, tc.y, 100, 200, width, 3); got != tc.row {
			t.Fatalf("point %d,%d: row %d, want %d", tc.x, tc.y, got, tc.row)
		}
	}
}

func TestPopupPickingRespectsChatPanelsAndTheirGaps(t *testing.T) {
	c := &ChatPanel{Rows: [2]int{5, 5}}
	for _, tc := range []struct {
		y      int
		inside bool
	}{{430, true}, {537, false}, {550, true}, {690, true}, {710, false}} {
		if c.Contains(100, tc.y, 768) != tc.inside {
			t.Fatal("chat pointer bounds", tc)
		}
	}
	if c.Contains(350, 550, 768) {
		t.Fatal("chat blocked the world outside its width")
	}
}
