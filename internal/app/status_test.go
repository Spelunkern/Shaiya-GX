package app

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

func TestStatusHotkeyFooterAndPointerOwnership(t *testing.T) {
	a := inputApp()
	a.inventory.open = true
	a.worldControls(&platform.Input{Chars: []rune{'T'}}, 1024, 768)
	if !a.status.open || a.inventory.open {
		t.Fatal("native Status hotkey did not replace inventory")
	}
	x, y := ui.StatusOrigin(1024, 768, false)
	a.worldControls(&platform.Input{MouseX: x + 2, MouseY: y + 157, Pressed: true, MouseDown: true, Down: "W"}, 1024, 768)
	controls := a.worldControls(&platform.Input{MouseX: x + 2, MouseY: y + 157, Clicked: true, Down: "W"}, 1024, 768)
	if !a.statusUI.Expanded || controls.Clicked || controls.Down != "W" {
		t.Fatal("status tab leaked a click or swallowed held movement")
	}
	controls = a.worldControls(&platform.Input{Keys: []platform.Key{platform.KeyEscape}}, 1024, 768)
	if a.status.open || len(controls.Keys) != 0 {
		t.Fatal("status close reached world logout")
	}
	a.worldControls(&platform.Input{MouseX: 682, MouseY: 715, Pressed: true, MouseDown: true}, 1024, 768)
	a.worldControls(&platform.Input{MouseX: 682, MouseY: 715, Clicked: true}, 1024, 768)
	if !a.status.open {
		t.Fatal("status footer button did not open")
	}
	a.toggleSkills(false)
	if a.status.open || !a.skills.open {
		t.Fatal("status overlapped another right-side panel")
	}
}
