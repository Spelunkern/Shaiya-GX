package app

import (
	"math"
	"strings"

	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

// Game.exe 0x7085b4: fixed vertical FOV; zoom changes camera position only.
const nativeWorldFOV = 0.785398006439209

// Native character+0x14 is feet+0.9 (0x42f9c3). Drawing temporarily
// subtracts 0.9 at 0x449bac and restores it at 0x449c2e. Go stores feet.
const nativeCharacterOriginHeight float32 = .9

const gmZoomMax = 100

func (a *App) cameraLimitCommand(text string) bool {
	fields := strings.Fields(text)
	if len(fields) == 0 || !strings.EqualFold(fields[0], "/camlimit") {
		return false
	}
	if a.state == nil {
		return true
	}
	if !a.state.Self.IsGM {
		a.questMessageID(12026)
		return true
	}
	if len(fields) != 1 {
		a.questMessageID(12219)
		return true
	}
	v := a.wv
	if v == nil || v.world == nil {
		a.questMessageID(12220)
		return true
	}
	if v.extendedCameraZoom {
		v.restoreCameraZoomLimit()
		a.questMessageID(12221)
	} else {
		v.extendedCameraZoom = true
		a.questMessageID(12222)
	}
	return true
}

func (v *worldView) restoreCameraZoomLimit() {
	v.extendedCameraZoom = false
	if !v.firstPerson {
		v.eye = v.limitCameraDistance(v.eye, zoomMax)
		if v.cameraReady {
			v.cameraEye = v.limitCameraDistance(v.cameraEye, zoomMax)
		}
	}
}

func (v *worldView) limitCameraDistance(eye render.V3, limit float32) render.V3 {
	origin := v.cameraOrigin()
	offset := eye.Sub(origin)
	if distance := offset.Len(); distance > limit {
		return origin.Add(offset.Scale(limit / distance))
	}
	return eye
}

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
				limit := float32(zoomMax)
				if v.extendedCameraZoom {
					limit = gmZoomMax
				}
				v.eye = origin.Sub(dir.Scale(min(max(distance-step, zoomMin), limit)))
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
	if v.extendedCameraZoom {
		v.eye = v.limitCameraDistance(v.eye, gmZoomMax)
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
	var eye render.V3
	var hit bool
	if !v.firstPerson {
		if v.camRayValid && v.camRayWorld == v.world && v.camRayOrigin == origin && v.camRayEye == v.eye && v.world.Ready() {
			eye, hit = v.camRayResult, v.camRayHit
		} else {
			eye, hit = v.world.CameraIntersection(origin, v.eye)
			if v.world.Ready() {
				v.camRayWorld, v.camRayOrigin, v.camRayEye = v.world, origin, v.eye
				v.camRayResult, v.camRayHit, v.camRayValid = eye, hit, true
			} else {
				v.camRayValid = false
			}
		}
	}
	if hit && !v.firstPerson {
		v.cameraEye = eye
	} else {
		v.cameraEye = cameraApproach(v.cameraEye, v.eye.Add(v.shakeOffset), dt)
	}
	if v.extendedCameraZoom && !v.firstPerson {
		v.cameraEye = v.limitCameraDistance(v.cameraEye, gmZoomMax)
	}
	v.cameraTarget = cameraApproach(v.cameraTarget, v.target().Add(v.shakeOffset), dt)
	if !v.firstPerson && v.cameraEye.Sub(origin).Len() < .3 {
		v.cameraEye = origin.Sub(v.facing.Scale(.3))
	}
}

func (a *App) InWorld() bool { return a.screen == screenWorld && a.wv != nil }
