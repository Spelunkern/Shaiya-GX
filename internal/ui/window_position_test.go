package ui

import "testing"

func TestWindowPositionKeepsNormalizedCenterAndBounds(t *testing.T) {
	var p WindowPosition
	if x, y := p.Origin(1024, 768, 256, 486, 752, 82); x != 752 || y != 82 {
		t.Fatal(x, y)
	}
	p.Move(200, 120, 1024, 768, 256, 486)
	if x, y := p.Origin(1024, 768, 256, 486, 0, 0); x != 200 || y != 120 {
		t.Fatal(x, y)
	}
	if x, y := p.Origin(2048, 1536, 256, 486, 0, 0); x != 528 || y != 483 {
		t.Fatal("normalized center not retained", x, y)
	}
	if x, y := p.Origin(320, 240, 256, 486, 0, 0); x < 0 || x > 64 || y != 0 {
		t.Fatal("offscreen after resize", x, y)
	}
	if x, y := p.Origin(1024, 768, 256, 486, 0, 0); x != 200 || y != 120 {
		t.Fatal("small viewport overwrote saved position", x, y)
	}
	p.Move(-100, 2000, 1024, 768, 256, 486)
	if x, y := p.Origin(1024, 768, 256, 486, 0, 0); x != 0 || y != 282 {
		t.Fatal("drag escaped viewport", x, y)
	}
}
