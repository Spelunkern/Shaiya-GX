package ui

import (
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"testing"
)

func TestDeathDialogNativeButtonsAndCapture(t *testing.T) {
	d, e := NewDeathDialog(marketTextures{})
	if e != nil {
		t.Fatal(e)
	}
	w, h := 1024, 768
	x, y := w/2-128, h/2-128
	click := func(px, py int) int {
		d.Update(&platform.Input{MouseX: px, MouseY: py, Pressed: true, MouseDown: true}, w, h)
		return d.Update(&platform.Input{MouseX: px, MouseY: py, Clicked: true}, w, h)
	}
	d.Set(DeathParty, "dead")
	if got := click(x+129, y+115); got != 2 {
		t.Fatal("party", got)
	}
	d.Set(DeathLocation, "where")
	if got := click(x+144, y+115); got != 2 {
		t.Fatal("here", got)
	}
	if got := click(x+39, y+115); got != 1 {
		t.Fatal("bind", got)
	}
	d.Set(DeathWaiting, "wait")
	if got := click(x+39, y+115); got != 0 {
		t.Fatal("wait accepted click", got)
	}
}
