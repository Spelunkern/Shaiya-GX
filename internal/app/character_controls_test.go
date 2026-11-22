package app

import (
	"math"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

func TestMouseAndKeyboardMovement(t *testing.T) {
	for _, tc := range []struct {
		in            platform.Input
		forward, back bool
		side, turn    int
	}{
		{in: platform.Input{Down: "A"}, turn: 1},
		{in: platform.Input{Down: "D"}, turn: -1},
		{in: platform.Input{Down: "A", RightDown: true}, turn: 1},
		{in: platform.Input{Down: "D", RightDown: true}, turn: -1},
		{in: platform.Input{MouseDown: true, RightDown: true}, forward: true},
		{in: platform.Input{Down: "WQ"}, forward: true, side: -1},
		{in: platform.Input{Down: "SE"}, back: true, side: 1},
		{in: platform.Input{Down: "WASD", RightDown: true, MouseDown: true, SuppressAutoRun: true}},
	} {
		got := characterControls(&tc.in, false)
		if got.forward != tc.forward || got.back != tc.back || got.side != tc.side || got.turn != tc.turn {
			t.Fatalf("%+v: %+v", tc.in, got)
		}
	}
	heading := render.V3{0, 0, 1}
	for _, keys := range []string{"W", "WQ", "WE", "S", "SQ", "SE", "Q", "E"} {
		d := controlVector(heading, render.V3{1, 0, 0}, characterControls(&platform.Input{Down: keys}, false))
		if math.Abs(float64(d.Len()-1)) > .00001 {
			t.Fatal("diagonal changes speed", keys, d)
		}
	}
	if d := controlVector(heading, render.V3{1, 0, 0}, characterControls(&platform.Input{Down: "WSQE"}, false)); d.Len() != 0 {
		t.Fatal("opposed keys did not cancel", d)
	}
	if playerMovementSpeed(&proto.CharacterSpeed{Move: 255}, true) != 0 {
		t.Fatal("mount bypassed server movement lock")
	}
}

func TestRightDragPreservesCombatFacing(t *testing.T) {
	a := inputApp()
	v := a.wv
	v.facing = render.V3{0, 0, 1}
	v.eye = render.V3{0, 2, -5}
	v.faceCombatTarget(render.V3{1, 0, 0})
	v.attackMoveUntil = 1
	position, facing, eye := v.pos, v.facing, v.eye
	a.updateWorld(&platform.Input{RightDown: true, MouseLook: true, MouseDX: 100}, 1.0/60)
	if v.pos != position || v.eye.Sub(eye).Len() < .001 {
		t.Fatal("right-drag should orbit without moving the player")
	}
	if v.facing.Sub(facing).Len() > .001 {
		t.Fatalf("right-drag changed combat facing from %v to %v", facing, v.facing)
	}

	a.clock = 2
	a.updateWorld(&platform.Input{RightDown: true, MouseLook: true}, 1.0/60)
	if v.combatHeading.Len() != 0 || v.facing[2] < .9 {
		t.Fatalf("right-drag did not resume steering after combat: %v", v.facing)
	}
}

func TestDiagonalMovementKeepsHeadingWhileCameraTurns(t *testing.T) {
	for _, tc := range []struct {
		keys string
		side float32
	}{
		{keys: "WQ", side: -1},
		{keys: "WE", side: 1},
	} {
		t.Run(tc.keys, func(t *testing.T) {
			a := inputApp()
			v := a.wv
			v.facing = render.V3{0, 0, 1}
			v.eye = render.V3{-5, 2, 0}
			v.moving, v.motion, v.sendT = true, 1, 1 // No movement packet is due.
			checkStep := func(in platform.Input, direction render.V3) {
				t.Helper()
				start := v.pos
				a.updateWorld(&in, 1.0/60)
				want := direction.Norm().Scale(runSpeed / 60)
				if moved := v.pos.Sub(start); moved.Sub(want).Len() > .001 {
					t.Fatalf("%+v: moved %v, want %v", in, moved, want)
				}
			}

			startDirection := render.V3{tc.side, 0, 1}
			checkStep(platform.Input{Down: tc.keys}, startDirection)
			checkStep(platform.Input{Down: tc.keys, MiddlePressed: true}, startDirection)
			checkStep(platform.Input{Down: "W"}, render.V3{-1, 0, 0})
			checkStep(platform.Input{Down: tc.keys}, render.V3{-1, 0, tc.side})
		})
	}
}

func TestGameMenuKeepsMovementAndOwnsOnlyItsPointer(t *testing.T) {
	a := inputApp()
	a.wv.autoRun, a.wv.travel = true, true
	controls := a.worldControls(&platform.Input{Down: "W", Keys: []platform.Key{platform.KeyEscape}}, 1024, 768)
	if !a.menuOpen || controls.SuppressAutoRun || controls.Down != "W" || !a.wv.autoRun || !a.wv.travel {
		t.Fatal("opening options stopped travel", controls)
	}
	x, y := a.gameMenu.Origin(1024, 768)
	controls = a.worldControls(&platform.Input{MouseX: x + 80, MouseY: y + 80, Pressed: true, MouseDown: true, Down: "W"}, 1024, 768)
	if controls.Pressed || controls.MouseDown || controls.Down != "W" || controls.SuppressAutoRun {
		t.Fatal("options did not separate pointer and movement", controls)
	}
	controls = a.worldControls(&platform.Input{MouseX: 750, MouseY: 400, Clicked: true, Down: "W"}, 1024, 768)
	if controls.Clicked || controls.Down != "W" {
		t.Fatal("options release reached world", controls)
	}
	controls = a.worldControls(&platform.Input{MouseX: 750, MouseY: 400, Pressed: true, MouseDown: true, Down: "W"}, 1024, 768)
	if !controls.Pressed || !controls.MouseDown || controls.Down != "W" {
		t.Fatal("options blocked world outside", controls)
	}
	a.wv.mouselook = true
	controls, captured := a.gameMenuControls(&platform.Input{MouseX: x + 80, MouseY: y + 80, RightDown: true, MouseLook: true, MouseDX: 12}, 1024, 768)
	if captured || !controls.RightDown || controls.MouseDX != 12 {
		t.Fatal("options captured a camera drag started outside", controls)
	}
	a.worldControls(&platform.Input{Keys: []platform.Key{platform.KeyEscape}}, 1024, 768)
	if a.menuOpen || a.state == nil || !a.wv.autoRun || !a.wv.travel {
		t.Fatal("closing options changed travel/session")
	}
}

func TestCombatFacingDoesNotRedirectMovementHeading(t *testing.T) {
	v := &worldView{facing: render.V3{0, 0, 1}}
	v.faceCombatTarget(render.V3{1, 0, 0})
	v.faceCombatTarget(render.V3{-1, 0, 0})
	v.restoreMovementHeading()
	if v.facing != (render.V3{0, 0, 1}) || v.combatHeading.Len() != 0 {
		t.Fatal("combat replaced movement heading", v.facing, v.combatHeading)
	}
}
