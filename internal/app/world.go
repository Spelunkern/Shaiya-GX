package app

import (
	"log"
	"math"

	"github.com/Spelunkern/shaiya-go-client/internal/game"
	"github.com/Spelunkern/shaiya-go-client/internal/platform"
	"github.com/Spelunkern/shaiya-go-client/internal/proto"
	"github.com/Spelunkern/shaiya-go-client/internal/render"
	"github.com/Spelunkern/shaiya-go-client/internal/scene"
)

// worldView is the in-game 3D view: map, own character and the native
// third-person camera (0x444b10 reset, 0x4397d0 orbit/zoom, 0x43bfa0
// movement; Phoenix tools/reference/ep5_camera_controls.md).
type worldView struct {
	mapID  uint16
	world  *scene.World
	self   *scene.Character
	pos    render.V3
	facing render.V3 // horizontal unit vector
	eye    render.V3 // desired eye; follows the position delta
	moving bool
	sendT  float64
	motion uint8
}

const (
	runSpeed   = 5.0 // native normal speed base
	zoomMin    = 2.1 // distance to the character origin
	zoomMax    = 13.0
	camArrowYw = 150.0 * math.Pi / 180
	camArrowPt = 80.0 * math.Pi / 180
	mouseScale = 0.1 * math.Pi / 180 // per pixel at sensitivity 3
)

func (a *App) enterWorld() {
	st := a.state
	v := &worldView{mapID: st.MapID, pos: render.V3{st.Self.X, st.Self.Y, st.Self.Z}}
	w, err := scene.LoadMap(a.assets, st.MapID)
	if err != nil {
		log.Printf("map %d: %v", st.MapID, err)
	}
	v.world = w
	// Facing from the 0x0105 direction (u16 angle); default +Z.
	ang := float64(st.Details.Dir) / 65536 * 2 * math.Pi
	v.facing = render.V3{float32(math.Sin(ang)), 0, float32(math.Cos(ang))}
	if w != nil && w.HasTerrain() {
		v.pos[1] = w.Height(v.pos[0], v.pos[2])
	}
	for _, c := range a.chars.Chars {
		if c != nil && c.ID == st.SelfID {
			if ch, err := scene.BuildCharacter(a.assets, a.appearance(c), "000_normal"); err == nil {
				v.self = ch
			} else {
				log.Printf("self character: %v", err)
			}
		}
	}
	for f := range a.assets.Missing {
		log.Printf("map %d: missing %s", st.MapID, f)
		delete(a.assets.Missing, f)
	}
	v.resetCamera()
	a.wv = v
}

func (v *worldView) target() render.V3 { return v.pos.Add(render.V3{0, 0.5, 0}) }

func (v *worldView) resetCamera() {
	v.eye = v.pos.Sub(v.facing.Scale(5)).Add(render.V3{0, 1.5, 0})
}

// rotateEye turns the eye around the target: yaw about world up, pitch
// about view x up; pitch is rejected when the eye-to-origin direction gets
// within 0.95 of the vertical.
func (v *worldView) rotateEye(yaw, pitch float64) {
	t := v.target()
	off := v.eye.Sub(t)
	c, s := float32(math.Cos(yaw)), float32(math.Sin(yaw))
	off = render.V3{off[0]*c - off[2]*s, off[1], off[0]*s + off[2]*c}
	if pitch != 0 {
		axis := off.Cross(render.V3{0, 1, 0}).Norm()
		rotated := rotateAxis(off, axis, float32(pitch))
		dir := t.Add(rotated).Sub(v.pos).Norm()
		if dir[1] < 0.95 && dir[1] > -0.95 {
			off = rotated
		}
	}
	v.eye = t.Add(off)
}

func rotateAxis(p, k render.V3, ang float32) render.V3 {
	c, s := float32(math.Cos(float64(ang))), float32(math.Sin(float64(ang)))
	return p.Scale(c).Add(k.Cross(p).Scale(s)).Add(k.Scale(k.Dot(p) * (1 - c)))
}

func (a *App) updateWorld(in *platform.Input, dt float64) {
	v := a.wv
	if v == nil {
		return
	}
	// Zoom toward the character origin, step clamped to 1.5.
	if in.Wheel != 0 {
		dir := v.pos.Sub(v.eye).Norm()
		step := float32(math.Max(-1.5, math.Min(1.5, in.Wheel*0.005)))
		eye := v.eye.Add(dir.Scale(step))
		d := v.pos.Sub(eye).Len()
		if d >= zoomMin && d <= zoomMax {
			v.eye = eye
		}
	}
	// Orbit: right drag (0.1 deg/px) and arrows (150 / 80 deg/s).
	yaw, pitch := 0.0, 0.0
	if in.RightDown && !in.RightPressed {
		yaw -= float64(in.MouseX-a.lastMouseX) * mouseScale
		pitch += float64(in.MouseY-a.lastMouseY) * mouseScale
	}
	if in.Held('<') {
		yaw += camArrowYw * dt
	}
	if in.Held('>') {
		yaw -= camArrowYw * dt
	}
	if in.Held('^') {
		pitch += camArrowPt * dt
	}
	if in.Held('v') {
		pitch -= camArrowPt * dt
	}
	a.lastMouseX, a.lastMouseY = in.MouseX, in.MouseY
	// Movement: W forward along the camera's XZ forward, S back, Q/E strafe,
	// A/D turn character and camera together (speed * 15 deg/s).
	fwd := v.target().Sub(v.eye)
	fwd[1] = 0
	fwd = fwd.Norm()
	right := render.V3{0, 1, 0}.Cross(fwd)
	var dir render.V3
	back := false
	if in.Held('W') {
		dir = dir.Add(fwd)
	}
	if in.Held('S') {
		dir = dir.Sub(fwd)
		back = true
	}
	if in.Held('Q') {
		dir = dir.Sub(right)
	}
	if in.Held('E') {
		dir = dir.Add(right)
	}
	turn := runSpeed * 15 * math.Pi / 180 * dt
	if in.Held('A') {
		yaw += turn
	}
	if in.Held('D') {
		yaw -= turn
	}
	if yaw != 0 || pitch != 0 {
		v.rotateEye(yaw, pitch)
	}
	moving := dir.Len() > 0
	if moving {
		dir = dir.Norm()
		if !back {
			v.facing = dir
		} else {
			v.facing = fwd
		}
		speed := float32(runSpeed)
		if back {
			speed /= 2 // NO VERIFICADO: backward speed table
		}
		step := dir.Scale(speed * float32(dt))
		v.pos = v.pos.Add(step)
		if v.world != nil && v.world.HasTerrain() {
			v.pos[1] = v.world.Height(v.pos[0], v.pos[2])
		}
		v.eye = v.eye.Add(step)
		v.eye[1] = v.pos[1] + (v.eye[1] - (v.pos[1] - step[1]))
	}
	if v.self != nil {
		switch {
		case moving && back:
			v.self.SetClip("003_bstep")
		case moving:
			v.self.SetClip("002_run")
		default:
			v.self.SetClip("000_normal")
		}
		v.self.Update(float32(dt))
	}
	// 0x0501: on start/stop, and every 0.5 s while moving (NO VERIFICADO:
	// the original's send cadence).
	v.sendT -= dt
	if moving != v.moving || (moving && v.sendT <= 0) {
		motion := uint8(0)
		if moving {
			motion = 1
		}
		dirAngle := uint16(math.Atan2(float64(v.facing[0]), float64(v.facing[2])) / (2 * math.Pi) * 65536)
		a.world.Send(proto.MovePacket(motion, dirAngle, v.pos[0], v.pos[1], v.pos[2]))
		v.sendT, v.moving, v.motion = 0.5, moving, motion
	}
	a.state.Self.X, a.state.Self.Y, a.state.Self.Z = v.pos[0], v.pos[1], v.pos[2]
}

func (a *App) drawWorldView(w, h int) {
	v := a.wv
	if v == nil || v.world == nil {
		return
	}
	s := a.scene3d
	s.View = render.LookAtLH(v.eye, v.target(), render.V3{0, 1, 0})
	aspect := projectionAspect(w, h)
	s.Proj = render.PerspectiveFovLH(math.Pi/4, aspect, 0.1, 400)
	s.SkyProj = render.PerspectiveFovLH(math.Pi/4, aspect, 10, 100000)
	s.FogColor, s.FogStart, s.FogEnd = v.world.FogColor, v.world.FogStart, v.world.FogEnd
	s.LightDir = render.V3{-1, -1, -1}
	s.Begin()
	if v.world.Sky != nil {
		v.world.Sky.Update(float32(a.frameDT))
		v.world.Sky.Draw(s, v.eye)
	}
	v.world.Draw(s)
	if v.self != nil {
		v.self.World = render.Basis(v.pos, v.facing.Scale(-1), render.V3{0, 1, 0})
		v.self.Draw(a.skinner, s)
	}
	s.End()
}

var _ = game.PhaseInWorld
