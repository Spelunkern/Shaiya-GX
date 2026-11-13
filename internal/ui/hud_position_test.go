package ui

import "testing"

func TestHUDPositionsIndependentAndResponsive(t *testing.T) {
	h := &HUD{}
	h.Position.Move(40, 90, 1024, 768, 213, 114)
	h.TargetPosition.Move(420, 120, 1024, 768, 256, 64)
	if x, y := h.Origin(1024, 768); x != 40 || y != 90 {
		t.Fatal(x, y)
	}
	if x, y := h.TargetOrigin(1024, 768); x != 420 || y != 120 {
		t.Fatal(x, y)
	}
	if x, y := h.Origin(320, 240); x < 0 || x > 107 || y < 0 || y > 126 {
		t.Fatal(x, y)
	}
	if x, y := h.TargetOrigin(320, 240); x < 0 || x > 64 || y < 0 || y > 176 {
		t.Fatal(x, y)
	}
}
