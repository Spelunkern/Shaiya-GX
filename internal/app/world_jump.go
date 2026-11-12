package app

import "github.com/Spelunkern/Shaiya-GX/internal/scene"

// 0x43efab, 0x44287b..0x44302e: jump adds 9 to the exponentially
// accelerated fall speed. This is visual prediction, never damage or immunity.
func (v *worldView) beginJump() {
	v.jumping, v.jumpFall, v.jumpFloor = true, -2.98, v.pos[1]
	if v.self != nil {
		v.self.ReleaseAction()
		v.self.PlayAction(8, false)
	}
}

func (v *worldView) advanceJump(dt float32) bool {
	if !v.jumping || dt <= 0 {
		return false
	}
	v.jumpFall = max(-20, v.jumpFall*(1+2.8*dt))
	pos := v.pos
	pos[1] += (9 + v.jumpFall) * dt
	floor := v.jumpFloor
	if v.world.HasTerrain() {
		floor = v.world.Height(pos[0], pos[2])
		if _, deep := v.world.WaterFloor(pos[0], pos[2]); deep {
			floor = scene.SwimEntryY
		}
	}
	if pos[1] <= floor {
		pos[1] = floor
		v.jumping = false
		if v.self != nil && v.self.PlayingAction(8) {
			v.self.ReleaseAction()
		}
	}
	changed := pos != v.pos
	v.place(pos)
	return changed
}
