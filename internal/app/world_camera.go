package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"math"
)

// Game.exe 0x7085b4: fixed vertical FOV; zoom changes camera position only.
const nativeWorldFOV = 0.785398006439209

// Native character+0x14 is feet+0.9 (0x42f9c3). Drawing temporarily
// subtracts 0.9 at 0x449bac and restores it at 0x449c2e. Go stores feet.
const nativeCharacterOriginHeight float32 = .9

func (v *worldView) cameraOrigin() render.V3 {
	return v.pos.Add(render.V3{0, nativeCharacterOriginHeight, 0})
}

// 0x4397d0 preserves the yawed XZ when pitch crosses the ground. Clamping
// only Y would change the orbit radius every frame the player keeps dragging.
func cameraOrbitFloor(yawed, candidate render.V3, floor float32) render.V3 {
	if candidate[1] < floor {
		return render.V3{yawed[0], floor, yawed[2]}
	}
	return candidate
}

// cameraInput follows 0x4397d0 and 0x43c6ca..0x43c73a.
func (v *worldView) cameraInput(in *platform.Input, dt float64, lastX, lastY int) (yaw, pitch float64) {
	if in.HasKey(platform.KeyF9) {
		v.resetCamera()
	}
	if in.HasKey(platform.KeyF10) {
		if v.firstPerson {
			v.resetCamera()
		} else {
			v.firstDirection = v.target().Sub(v.eye).Norm()
			v.firstPerson = true
			v.eye = v.cameraOrigin().Add(render.V3{0, .8, 0})
			v.snapRenderedCamera()
		}
	}
	if !v.firstPerson {
		wheel := in.Wheel
		if in.Held('+') {
			wheel = 2500 * dt
		}
		if in.Held('-') {
			wheel = -2500 * dt
		}
		if wheel != 0 {
			origin := v.cameraOrigin()
			dir := origin.Sub(v.eye).Norm()
			distance := origin.Sub(v.eye).Len()
			if distance > 0 {
				step := float32(math.Max(-1.5, math.Min(1.5, wheel*.005)))
				v.eye = origin.Sub(dir.Scale(min(max(distance-step, zoomMin), zoomMax)))
			}
		}
		if in.MiddlePressed || in.HasKey(platform.KeyHome) {
			v.rotateEye(math.Pi, 0)
			v.snapRenderedCamera()
		}
	}
	yawRate, pitchRate := camArrowYw, camArrowPt
	if v.firstPerson {
		yawRate, pitchRate = 25*math.Pi/180, -25*math.Pi/180
	}
	if in.RightDown && !in.RightPressed && dt > 0 {
		dx, dy := float64(in.MouseX-lastX), float64(in.MouseY-lastY)
		if in.MouseLook {
			dx, dy = in.MouseDX, in.MouseDY
		}
		if v.firstPerson {
			yaw = -3 * dx * dt * math.Pi / 180
			pitch = -5 * dy * dt * math.Pi / 180
		} else {
			// The original quantizes its angular speed to integral degrees/second.
			yaw = -math.Trunc(5*dx*float64(float32(.02))/float64(float32(dt))) * float64(float32(dt)) * math.Pi / 180
			pitch = math.Trunc(5*dy*float64(float32(.02))/float64(float32(dt))) * float64(float32(dt)) * math.Pi / 180
		}
	}
	if in.Held('<') {
		yaw += yawRate * dt
	}
	if in.Held('>') {
		yaw -= yawRate * dt
	}
	if in.Held('^') {
		pitch += pitchRate * dt
	}
	if in.Held('v') {
		pitch -= pitchRate * dt
	}
	return
}

func (v *worldView) orbitCamera(yaw, pitch float64) {
	if !v.firstPerson {
		v.rotateEye(yaw, pitch)
		return
	}
	d := v.firstDirection
	c, s := float32(math.Cos(yaw)), float32(math.Sin(yaw))
	d = render.V3{d[0]*c - d[2]*s, d[1], d[0]*s + d[2]*c}
	axis := d.Cross(render.V3{0, 1, 0}).Norm()
	candidate := rotateAxis(d, axis, float32(pitch)).Norm()
	if candidate[1] > -.99 && candidate[1] < .99 {
		d = candidate
	}
	v.firstDirection = d
}

func (v *worldView) constrainCamera() {
	if v.firstPerson {
		v.eye = v.cameraOrigin().Add(render.V3{0, .8, 0})
		return
	}
	if v.world.HasTerrain() {
		v.eye[1] = max(v.eye[1], v.world.Height(v.eye[0], v.eye[2])+.5)
	}
}

// MouseLookActive is evaluated after UI ownership has filtered pointer input.
func (a *App) MouseLookActive() bool {
	return a.screen == screenWorld && a.wv != nil && a.wv.mouselook
}

// 0x54fe70 / 0x550090: desired and rendered cameras have distinct state.
// The unobstructed response is capped linear interpolation, not exponential.
func (v *worldView) snapRenderedCamera() {
	v.cameraEye, v.cameraTarget, v.cameraReady = v.eye, v.target(), true
}

func cameraApproach(current, target render.V3, dt float32) render.V3 {
	delta := target.Sub(current)
	if delta.Len()*(1-10*dt) <= .005 {
		return target
	}
	return current.Add(delta.Scale(10 * dt))
}

func (v *worldView) updateRenderedCamera(dt float32) {
	if !v.cameraReady {
		v.snapRenderedCamera()
		return
	}
	if dt <= 0 {
		return
	}
	origin := v.cameraOrigin()
	if eye, hit := v.world.CameraIntersection(origin, v.eye); hit && !v.firstPerson {
		v.cameraEye = eye
	} else {
		v.cameraEye = cameraApproach(v.cameraEye, v.eye, dt)
	}
	v.cameraTarget = cameraApproach(v.cameraTarget, v.target(), dt)
	if !v.firstPerson && v.cameraEye.Sub(origin).Len() < .3 {
		v.cameraEye = origin.Sub(v.facing.Scale(.3))
	}
}
