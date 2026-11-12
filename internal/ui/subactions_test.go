package ui

import (
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"testing"
)

func TestSubActionsGridCapturesAndRejectsDisabled(t *testing.T) {
	p, err := NewSubActions(marketTextures{})
	if err != nil {
		t.Fatal(err)
	}
	w, h := 1024, 734
	x, y := p.Origin(w, h)
	for _, i := range []int{0, 3, 4, 5, 10, 14, 1, 2, 15} {
		cx, cy := SubActionCell(i)
		in := platform.Input{MouseX: x + cx + 2, MouseY: y + cy + 2, Pressed: true, MouseDown: true}
		if _, captured := p.Update(&in, w, h); !captured {
			t.Fatal("press escaped panel", i)
		}
		in.Pressed = false
		in.MouseDown = false
		in.Clicked = true
		got, captured := p.Update(&in, w, h)
		if !captured || subActionEnabled(i) && got != i || !subActionEnabled(i) && got != -1 {
			t.Fatalf("row %d got %d captured=%v", i, got, captured)
		}
	}
	cx, cy := SubActionCell(5)
	in := platform.Input{MouseX: x + cx + 2, MouseY: y + cy + 2, Clicked: true}
	if got, _ := p.Update(&in, w, h); got != -1 {
		t.Fatal("release without press activated", got)
	}
	in = platform.Input{MouseX: x + 251, MouseY: y + 2, Pressed: true, MouseDown: true}
	p.Update(&in, w, h)
	in.Pressed, in.MouseDown, in.Clicked = false, false, true
	if got, captured := p.Update(&in, w, h); got != SubActionClose || !captured {
		t.Fatal("native close button failed", got, captured)
	}
}
