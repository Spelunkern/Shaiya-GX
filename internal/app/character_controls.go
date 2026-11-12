package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

// Phoenix player_controller.gd _read_controls/_update_movement: character-relative
// travel, right-drag steering, mouse chord forward and normalized diagonals.
// Speeds and wire motions continue to use the EP5/Rapture definitions.
type movementControls struct {
	forward, back bool
	side, turn    int
}

func characterControls(in *platform.Input, autoRun bool) movementControls {
	if in.SuppressAutoRun {
		return movementControls{}
	}
	c := movementControls{forward: in.Held('W') || autoRun || in.MouseDown && in.RightDown, back: in.Held('S')}
	if in.Held('Q') || in.RightDown && in.Held('A') {
		c.side--
	}
	if in.Held('E') || in.RightDown && in.Held('D') {
		c.side++
	}
	if !in.RightDown {
		if in.Held('A') {
			c.turn++
		}
		if in.Held('D') {
			c.turn--
		}
	}
	return c
}

func cameraHeading(view, fallback render.V3) render.V3 {
	view[1] = 0
	if view.Len() < .0001 {
		return fallback
	}
	return view.Norm()
}

func controlDirection(heading render.V3, c movementControls) render.V3 {
	dir := render.V3{}
	if c.forward {
		dir = dir.Add(heading)
	}
	if c.back {
		dir = dir.Sub(heading)
	}
	dir = dir.Add(render.V3{0, 1, 0}.Cross(heading).Scale(float32(c.side)))
	return dir.Norm()
}

// Phoenix keeps combat facing separate from the heading resumed by controls.
func (v *worldView) faceCombatTarget(direction render.V3) {
	if v.combatHeading.Len() == 0 {
		v.combatHeading = v.facing
	}
	v.facing = direction
}

func (v *worldView) restoreMovementHeading() {
	if v.combatHeading.Len() > 0 {
		v.facing = v.combatHeading
		v.combatHeading = render.V3{}
	}
}
