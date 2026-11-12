package app

import (
	"math"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
)

// Native 0x43bfa0: base 5 (0x409250), offsets -2/-1/0/+1/+2,
// and 255 blocks movement. Unknown indices retain the native base speed.
func movementSpeed(s *proto.CharacterSpeed) float32 {
	if s == nil {
		return runSpeed
	}
	if s.Move == 255 {
		return 0
	}
	if s.Move <= 4 {
		return runSpeed + float32(s.Move) - 2
	}
	return runSpeed
}

// Native movement replaces the normal speed with the mount's speed. The base
// tier is known to be 7; some special variants use 8 but their tier is not
// carried by Rapture's mount-shape packet.
func playerMovementSpeed(s *proto.CharacterSpeed, mounted bool) float32 {
	if s != nil && s.Move == 255 {
		return 0
	}
	if mounted {
		return 7
	}
	return movementSpeed(s)
}

// Phoenix's Shaiya mount catalog contains no aquatic mounts; their normal
// movement animations are also the available fallback in water. The swimming
// speed remains provisional pending a measurement from the original client.
func swimmingMovementSpeed(speed float32, swimming bool) float32 {
	if swimming && speed > 0 {
		return 2.8
	}
	return speed
}

// place keeps the complete camera offset, including elevation and user zoom.
func (v *worldView) place(pos render.V3) {
	v.eye = v.eye.Add(pos.Sub(v.pos))
	v.pos = pos
}

func (v *worldView) movePacket(motion uint8) []byte {
	direction := uint16(math.Atan2(float64(v.facing[0]), float64(v.facing[2])) / (2 * math.Pi) * 65536)
	return proto.MovePacket(motion, direction, v.pos[0], v.pos[1], v.pos[2])
}

func (v *worldView) step(delta render.V3) bool {
	pos := v.pos.Add(delta)
	if v.world != nil && v.world.HasTerrain() {
		size := float32(v.world.WLD.MapSize)
		pos[0], pos[2] = min(max(pos[0], 0), size), min(max(pos[2], 0), size)
		if floor, deep := v.world.WaterFloor(pos[0], pos[2]); deep {
			if v.world.SwimmingAt(v.pos[0], v.pos[1], v.pos[2]) {
				pos[1] = min(max(pos[1], floor), scene.SwimSurfaceY)
			} else {
				// Falling into deep water takes time; a ledge must not teleport
				// the player or camera directly to the swimming level.
				pos[1] = max(v.pos[1], scene.SwimEntryY)
			}
		} else {
			if !v.jumping {
				pos[1] = v.world.Height(pos[0], pos[2])
			}
		}
	}
	moved := pos != v.pos
	v.place(pos)
	return moved
}

// Apply gravity after walking over deep water. The entry threshold stops the
// fall and lets buoyancy take over; all vertical movement is sent to Rapture.
func (v *worldView) settleIntoWater(dt float32) bool {
	if v.world == nil {
		return false
	}
	_, deep := v.world.WaterFloor(v.pos[0], v.pos[2])
	if !deep || v.pos[1] <= scene.SwimSurfaceY {
		v.waterFallSpeed = 0
		return false
	}
	v.waterFallSpeed = min(v.waterFallSpeed+9.8*dt, 30)
	pos := v.pos
	pos[1] = max(scene.SwimEntryY, pos[1]-v.waterFallSpeed*dt)
	if pos[1] <= scene.SwimSurfaceY {
		pos[1] = scene.SwimEntryY
		v.waterFallSpeed = 0
	}
	if pos == v.pos {
		return false
	}
	v.place(pos)
	return true
}

// Phoenix rises gently near the surface; a deeply submerged position stays put.
func (v *worldView) floatAtSurface(dt float32) bool {
	if v.world == nil || !v.world.SwimmingAt(v.pos[0], v.pos[1], v.pos[2]) || v.pos[1] < scene.SwimEntryY || v.pos[1] >= scene.SwimSurfaceY {
		return false
	}
	pos := v.pos
	pos[1] = min(scene.SwimSurfaceY, pos[1]+1.05*dt)
	if pos == v.pos {
		return false
	}
	v.place(pos)
	return true
}

// Swimming forward follows camera pitch; side steps remain horizontal.
func swimDirection(horizontal render.V3, view render.V3, forward, backward bool) render.V3 {
	y := view.Norm()[1]
	if y >= -0.34 && y <= 0.02 {
		y = 0
	}
	if forward {
		horizontal[1] += y
	}
	if backward {
		horizontal[1] -= y
	}
	return horizontal.Norm()
}

func (v *worldView) riseInWater(dt float32) bool {
	if v.world == nil || !v.world.SwimmingAt(v.pos[0], v.pos[1], v.pos[2]) {
		return false
	}
	pos := v.pos
	pos[1] = min(scene.SwimSurfaceY, pos[1]+1.8*dt)
	if pos == v.pos {
		return false
	}
	v.place(pos)
	return true
}

// Only packets about this character may reconcile local prediction. A map
// change will build a new view from State instead of moving the old camera.
func (a *App) confirmedPosition(p []byte) {
	v := a.wv
	if v == nil || v.mapID != a.state.MapID {
		return
	}
	switch proto.Op(p) {
	case proto.CharacterMove, proto.CharacterMapTeleport, proto.CharacterResurrect:
		if proto.NewReader(p).U32() != a.state.SelfID {
			return
		}
		v.place(render.V3{a.state.Self.X, a.state.Self.Y, a.state.Self.Z})
		v.combatHeading = render.V3{}
		v.approachTarget = 0
		v.snapRenderedCamera()
		v.moving, v.motion, v.sendT = false, 0, 0
		v.waterDirty = false
		v.waterFallSpeed = 0
		v.jumping = false
		v.evading = false
		v.travel = false
		if v.self != nil && v.self.PlayingAction(8) {
			v.self.ReleaseAction()
		}
		if proto.Op(p) != proto.CharacterMove {
			v.autoRun = false
		}
		if proto.Op(p) == proto.CharacterMove {
			v.motion = a.state.Self.Motion
			v.moving = v.motion&0x3f != 0
			angle := float64(a.state.Self.Dir) / 65536 * 2 * math.Pi
			v.facing = render.V3{float32(math.Sin(angle)), 0, float32(math.Cos(angle))}
		}
	}
}

func (v *worldView) turnFacing(yaw float64) {
	c, s := float32(math.Cos(yaw)), float32(math.Sin(yaw))
	f := v.facing
	v.facing = render.V3{f[0]*c - f[2]*s, 0, f[0]*s + f[2]*c}
}

// Original 0x43f2b0: backward movement has its own tiers, not runSpeed/2.
func backwardMovementSpeed(s *proto.CharacterSpeed, mounted bool) float32 {
	if s != nil && s.Move == 255 {
		return 0
	}
	if mounted {
		return 2
	}
	if s != nil && s.Move <= 4 {
		return .5 + float32(s.Move)*.5
	}
	return 1.5
}
