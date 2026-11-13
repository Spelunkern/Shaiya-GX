package ui

import (
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"testing"
)

func TestActionBarReleaseRequiresSamePressedSlot(t *testing.T) {
	b := &ActionBar{held: -1, Hover: -1}
	x, y := ActionBarOrigin(1024, 768)
	input := platform.Input{MouseX: x + 31, MouseY: y + 11, Pressed: true}
	if got, captured := b.Update(&input, 1024, 768); got != -1 || !captured {
		t.Fatal("press should capture without activating")
	}
	input.Pressed, input.Clicked = false, true
	input.MouseX += 40
	if got, _ := b.Update(&input, 1024, 768); got != -1 {
		t.Fatal("drag activated another slot")
	}
	input.Clicked, input.Pressed = false, true
	b.Update(&input, 1024, 768)
	input.Clicked, input.Pressed = true, false
	if got, _ := b.Update(&input, 1024, 768); got != 1 {
		t.Fatalf("clicked slot=%d", got)
	}
	input.MouseX = x + 63 // spacing between icons
	if got, _ := b.Update(&input, 1024, 768); got != -1 {
		t.Fatal("spacing activated a slot")
	}
}

func TestSkillIconUsesNativeSheetAndOneBasedIndex(t *testing.T) {
	for _, tc := range []struct {
		icon uint16
		file string
		u, v float32
	}{
		{1, "icon/icon_skill.png", 0, 0},
		{16, "icon/icon_skill.png", 15.0 / 16, 0},
		{17, "icon/icon_skill.png", 0, 1.0 / 16},
		{1001, "icon/icon_skill2_dds.png", 0, 0},
		{2256, "icon/icon_skill3_dds.png", 15.0 / 16, 15.0 / 16},
	} {
		file, uv, ok := SkillIcon(tc.icon)
		if !ok || file != tc.file || uv.U0 != tc.u || uv.V0 != tc.v {
			t.Errorf("icon %d: %s %+v", tc.icon, file, uv)
		}
	}
	if _, _, ok := SkillIcon(999); ok {
		t.Fatal("accepted icon outside atlas")
	}
}

func TestActionBarLayoutRetainsSlotsAndOrientation(t *testing.T) {
	b := &ActionBar{Hover: -1, held: -1}
	const w, h = 1024, 768
	x, y := b.Origin(0, w, h)
	if got, _ := b.Layout(&platform.Input{MouseX: x + 5, MouseY: y + 8, Pressed: true, MouseDown: true}, w, h); !got {
		t.Fatal("background failed to start drag")
	}
	b.Layout(&platform.Input{MouseX: 105, MouseY: 108, MouseDown: true}, w, h)
	if captured, changed := b.Layout(&platform.Input{MouseX: 105, MouseY: 108, Clicked: true}, w, h); !captured || !changed {
		t.Fatal("release did not persist layout")
	}
	if x, y := b.Origin(0, w, h); x != 100 || y != 100 {
		t.Fatal("drag offset lost", x, y)
	}
	x, y = b.SlotOrigin(3, w, h)
	if b.SlotAt(x+1, y+1, w, h) != 3 {
		t.Fatal("moved icon hit test lost slot")
	}
	if captured, _ := b.Layout(&platform.Input{MouseX: x + 1, MouseY: y + 1, Pressed: true, MouseDown: true}, w, h); captured {
		t.Fatal("icon press started frame drag")
	}
	bx, by := b.Origin(1, w, h)
	b.Layout(&platform.Input{MouseX: bx + 425, MouseY: by + 31, Pressed: true, MouseDown: true}, w, h)
	b.Layout(&platform.Input{MouseX: bx + 425, MouseY: by + 31, Clicked: true}, w, h)
	if !b.Position[1].Vertical {
		t.Fatal("orientation button did not rotate second bar")
	}
	x, y = b.SlotOrigin(19, w, h)
	if b.SlotAt(x+1, y+1, w, h) != 19 {
		t.Fatal("vertical second bar hit test lost slot")
	}
}

func TestOverlappingBarsUseTheVisibleTopSlot(t *testing.T) {
	b := &ActionBar{}
	x, y := b.Origin(0, 1024, 768)
	b.Position[1].Move(x, y, 1024, 768, 448, 54)
	sx, sy := b.SlotOrigin(0, 1024, 768)
	if index := b.SlotAt(sx+1, sy+1, 1024, 768); index != 10 {
		t.Fatal("click passed through the top bar", index)
	}
}
