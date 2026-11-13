package ui

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/platform"
)

func TestBottomBarClickRequiresSamePressAndCapturesBackground(t *testing.T) {
	for _, size := range [][2]int{{640, 480}, {800, 600}, {1024, 734}, {1280, 720}} {
		w, h := size[0], size[1]
		b := &BottomBar{}
		b.controls(w, h, false, false, false, false)
		inventory, skills := b.buttons[2], b.buttons[3]
		in := platform.Input{MouseX: inventory.X() + 1, MouseY: inventory.Y() + 1, Clicked: true}
		if action, captured := b.Update(&in, w, h, false, false, false, false); action != BottomNone || !captured {
			t.Fatal(size, "unowned release activated inventory or reached world")
		}
		in.Clicked, in.Pressed, in.MouseDown = false, true, true
		b.Update(&in, w, h, false, false, false, false)
		in.Pressed, in.Clicked, in.MouseDown = false, true, false
		in.MouseX = skills.X() + 1
		if action, _ := b.Update(&in, w, h, false, false, false, false); action != BottomNone {
			t.Fatal(size, "drag activated another button")
		}
		in.Clicked, in.Pressed, in.MouseDown = false, true, true
		b.Update(&in, w, h, false, false, false, false)
		in.Pressed, in.Clicked, in.MouseDown = false, true, false
		if action, _ := b.Update(&in, w, h, false, false, false, false); action != BottomSkills {
			t.Fatal(size, "normal skill click did not activate")
		}
		in = platform.Input{MouseX: w / 2, MouseY: h - 2, RightPressed: true}
		if _, captured := b.Update(&in, w, h, false, false, false, false); !captured {
			t.Fatal(size, "bottom background did not capture camera drag")
		}
		in.RightPressed = false
		if _, captured := b.Update(&in, w, h, false, false, false, false); captured {
			t.Fatal(size, "idle hover captured keyboard/movement")
		}
	}
}

func TestBottomBarGuildButton(t *testing.T) {
	b := &BottomBar{}
	b.controls(1024, 768, false, false, false, false)
	button := b.buttons[6]
	in := platform.Input{MouseX: button.X() + 1, MouseY: button.Y() + 1, Pressed: true, MouseDown: true}
	b.Update(&in, 1024, 768, false, false, false, false)
	in.Pressed, in.MouseDown, in.Clicked = false, false, true
	if action, captured := b.Update(&in, 1024, 768, false, false, false, false); action != BottomGuild || !captured {
		t.Fatal("guild button did not activate or capture", action, captured)
	}
}

func TestBottomBarMallAndStatusButtons(t *testing.T) {
	for index, want := range map[int]BottomAction{0: BottomMall, 1: BottomStatus, 5: BottomQuests, 6: BottomGuild} {
		b := &BottomBar{}
		b.controls(1024, 768, false, false, false, false)
		button := b.buttons[index]
		in := platform.Input{MouseX: button.X() + 1, MouseY: button.Y() + 1, Pressed: true, MouseDown: true}
		b.Update(&in, 1024, 768, false, false, false, false)
		in.Pressed, in.MouseDown, in.Clicked = false, false, true
		if got, captured := b.Update(&in, 1024, 768, false, false, false, false); got != want || !captured {
			t.Fatalf("button %d: %v %v", index, got, captured)
		}
	}
}
