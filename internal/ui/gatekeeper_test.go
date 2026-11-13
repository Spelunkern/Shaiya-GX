package ui

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"testing"
)

func TestPortalOnlyOffersPopulatedDestinationsAndCapturesItsPress(t *testing.T) {
	v := &Gatekeeper{}
	for i := range v.moves {
		v.moves[i] = Button{OffX: 434, OffY: 80 + 101*i, HitW: 58, HitH: 40}
	}
	gates := [3]sdata.NPCGate{{MapID: 1, X: 2, Name: "Village"}, {MapID: 0, X: 155, Name: "Karis"}}
	x, y := v.Origin(1024, 734)
	click := func(index int, busy bool) int {
		in := platform.Input{MouseX: x + 440, MouseY: y + 90 + 101*index, Pressed: true, MouseDown: true}
		v.Update(&in, 1024, 734, gates, busy)
		in.Pressed, in.MouseDown, in.Clicked = false, false, true
		choice, _ := v.Update(&in, 1024, 734, gates, busy)
		return choice
	}
	if got := click(0, false); got != 0 {
		t.Fatal("available portal did not activate", got)
	}
	if got := click(1, false); got != 1 {
		t.Fatal("map-zero destination was treated as empty", got)
	}
	if got := click(2, false); got != -1 {
		t.Fatal("empty portal activated", got)
	}
	if got := click(0, true); got != -1 {
		t.Fatal("pending portal sent a duplicate request", got)
	}
	if got, _ := v.Update(&platform.Input{MouseX: x + 440, MouseY: y + 90, Clicked: true}, 1024, 734, gates, false); got != -1 {
		t.Fatal("unowned release activated portal")
	}
}
