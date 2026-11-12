package ui

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/platform"
)

func TestAppearanceEditorCyclesAndRestrictsSex(t *testing.T) {
	e := &AppearanceEditor{}
	for i := range e.arrows {
		e.arrows[i] = Button{OffX: 142 + i%2*60, OffY: 25 + i/2*28, HitW: 23, HitH: 23}
	}
	click := func(i int) {
		x, y := appearanceEditorOrigin(800, 600)
		x += e.arrows[i].OffX + 1
		y += e.arrows[i].OffY + 1
		e.Update(&platform.Input{MouseX: x, MouseY: y, Pressed: true, MouseDown: true}, 800, 600)
		e.Update(&platform.Input{MouseX: x, MouseY: y, Clicked: true}, 800, 600)
	}
	e.Open([4]uint8{}, false)
	click(0)
	if e.Values[0] != 4 {
		t.Fatal("hair did not wrap backward")
	}
	click(1)
	if e.Values[0] != 0 {
		t.Fatal("hair did not wrap forward")
	}
	click(7)
	if e.Values[3] != 0 {
		t.Fatal("sex changed on appearance-only item")
	}
	e.Open([4]uint8{}, true)
	click(7)
	if e.Values[3] != 1 {
		t.Fatal("sex change item did not switch sex")
	}
}
