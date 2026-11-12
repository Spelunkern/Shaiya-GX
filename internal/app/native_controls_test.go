package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"math"
	"testing"
)

func TestNativeCameraZoomFlipAndModes(t *testing.T) {
	v := &worldView{pos: render.V3{100, 20, 100}, facing: render.V3{0, 0, 1}}
	v.resetCamera()
	original := v.eye
	v.cameraInput(&platform.Input{Wheel: 120}, 1.0/60, 0, 0)
	if delta := original.Sub(v.cameraOrigin()).Len() - v.eye.Sub(v.cameraOrigin()).Len(); math.Abs(float64(delta)-.6) > .0001 {
		t.Fatal("wheel does not dolly by .6", delta)
	}
	for range 100 {
		v.cameraInput(&platform.Input{Wheel: 120}, .016, 0, 0)
	}
	if d := v.eye.Sub(v.cameraOrigin()).Len(); math.Abs(float64(d)-zoomMin) > .0001 {
		t.Fatal("zoom did not clamp", d)
	}
	for range 100 {
		v.cameraInput(&platform.Input{Down: "-"}, .016, 0, 0)
	}
	if d := v.eye.Sub(v.cameraOrigin()).Len(); math.Abs(float64(d)-zoomMax) > .0001 {
		t.Fatal("page down did not clamp", d)
	}
	before := v.eye.Sub(v.target())
	v.cameraInput(&platform.Input{Keys: []platform.Key{platform.KeyHome}}, .016, 0, 0)
	after := v.eye.Sub(v.target())
	if math.Abs(float64(before[0]+after[0])) > .0001 || math.Abs(float64(before[2]+after[2])) > .0001 || before[1] != after[1] {
		t.Fatal("home did not flip the eye", before, after)
	}
	v.cameraInput(&platform.Input{Keys: []platform.Key{platform.KeyF10}}, .016, 0, 0)
	if !v.firstPerson || v.eye.Sub(v.cameraOrigin().Add(render.V3{0, .8, 0})).Len() > .0001 {
		t.Fatal("first person eye", v.eye)
	}
	old := v.firstDirection
	v.orbitCamera(.2, .1)
	if v.firstDirection == old || v.eye.Sub(v.cameraOrigin().Add(render.V3{0, .8, 0})).Len() > .0001 {
		t.Fatal("first person orbit moved the eye")
	}
	for range 100 {
		v.orbitCamera(0, .02)
	}
	if math.Abs(float64(v.firstDirection[1])) >= .99 {
		t.Fatal("first person crossed a pole")
	}
	v.cameraInput(&platform.Input{Keys: []platform.Key{platform.KeyF9}}, .016, 0, 0)
	if v.firstPerson || v.eye != original {
		t.Fatal("F9 did not reset", v.eye, original)
	}
}

func TestNativeJumpTrajectoryAndLanding(t *testing.T) {
	for _, fps := range []int{30, 60, 144} {
		v := &worldView{pos: render.V3{10, 5, 10}, eye: render.V3{10, 7, 5}}
		v.beginJump()
		dt := float32(1) / float32(fps)
		expectedFall := float32(-2.98)
		expectedY := float32(5)
		frames := 0
		for ; v.jumping && frames < fps*3; frames++ {
			expectedFall = max(-20, expectedFall*(1+2.8*dt))
			expectedY = max(float32(5), expectedY+(9+expectedFall)*dt)
			v.advanceJump(dt)
			if math.Abs(float64(v.pos[1]-expectedY)) > .00001 {
				t.Fatal("jump diverged from native recurrence", fps, frames, v.pos[1], expectedY)
			}
			if v.eye.Sub(v.pos).Sub(render.V3{0, 2, -5}).Len() > .0001 {
				t.Fatal("jump lost camera offset")
			}
		}
		if frames < fps/2 || frames > fps*2 || v.jumping || v.pos[1] != 5 {
			t.Fatal("jump failed to land", fps, frames, v.pos)
		}
	}
}

func TestNativeAutoRunEdgesAndWalkKey(t *testing.T) {
	a := inputApp()
	a.wv = &worldView{facing: render.V3{0, 0, 1}}
	a.wv.resetCamera()
	// Modal suppression lets us test the release state without travelling.
	a.updateWorld(&platform.Input{Released: "W"}, .1)
	a.updateWorld(&platform.Input{Released: "W", SuppressAutoRun: true}, .2)
	if a.wv.autoRun {
		t.Fatal("modal input started autorun")
	}
	a.updateWorld(&platform.Input{Keys: []platform.Key{platform.KeyTab}}, 0)
	if !a.wv.walking {
		t.Fatal("Tab did not select walk")
	}
}

func TestNativeRenderedCameraLagsAndResetSnaps(t *testing.T) {
	v := &worldView{pos: render.V3{100, 5, 100}, facing: render.V3{0, 0, 1}}
	v.resetCamera()
	old := v.cameraEye
	v.place(v.pos.Add(render.V3{1, 0, 0}))
	v.updateRenderedCamera(.016)
	if delta := v.cameraEye.Sub(old); math.Abs(float64(delta[0]-.16)) > .0001 {
		t.Fatal("camera lost native 10*dt response", delta)
	}
	if v.cameraEye == v.eye {
		t.Fatal("desired and rendered cameras were conflated")
	}
	v.updateRenderedCamera(.2)
	if v.cameraEye != v.eye || v.cameraTarget != v.target() {
		t.Fatal("camera overshot instead of snapping")
	}
	v.rotateEye(.4, 0)
	v.resetCamera()
	if v.cameraEye != v.eye {
		t.Fatal("reset left a stale rendered camera")
	}
}

func TestNativeBackwardSpeedTiers(t *testing.T) {
	for tier, want := range []float32{.5, 1, 1.5, 2, 2.5} {
		if got := backwardMovementSpeed(&proto.CharacterSpeed{Move: uint8(tier)}, false); got != want {
			t.Fatal(tier, got, want)
		}
	}
	if backwardMovementSpeed(nil, false) != 1.5 || backwardMovementSpeed(nil, true) != 2 || backwardMovementSpeed(&proto.CharacterSpeed{Move: 255}, false) != 0 {
		t.Fatal("backward base/mount/blocked tiers")
	}
}

// Fixtures from isolated Game.exe 0x4397d0 execution, not controller outputs.
func TestCameraAgainstNativeInstructionFixtures(t *testing.T) {
	for _, tc := range []struct {
		key string
		eye render.V3
	}{
		{"^", render.V3{0, 1.6160738468170166, -4.975377082824707}},
		{"v", render.V3{0, 1.383384346961975, -5.021914958953857}},
		{">", render.V3{-.21809694170951843, 1.5, -4.995241165161133}},
		{"<", render.V3{.21809694170951843, 1.5, -4.995241165161133}},
		{"+", render.V3{0, 1.4401358366012573, -4.800452709197998}},
		{"-", render.V3{0, 1.5598641633987427, -5.199547290802002}},
	} {
		v := &worldView{facing: render.V3{0, 0, 1}}
		v.resetCamera()
		yaw, pitch := v.cameraInput(&platform.Input{Down: tc.key}, 1.0/60, 0, 0)
		v.orbitCamera(yaw, pitch)
		if v.eye.Sub(tc.eye.Add(render.V3{0, nativeCharacterOriginHeight, 0})).Len() > 2e-5 {
			t.Fatal(tc.key, v.eye, tc.eye)
		}
	}
	for _, tc := range []struct {
		fps   int
		angle float64
	}{{30, -9.966666917274348}, {60, -9.98333360449017}, {144, -9.993055896051843}} {
		v := &worldView{facing: render.V3{0, 0, 1}}
		v.resetCamera()
		yaw, pitch := v.cameraInput(&platform.Input{RightDown: true, MouseX: 100}, 1.0/float64(tc.fps), 0, 0)
		v.orbitCamera(yaw, pitch)
		angle := math.Atan2(float64(v.eye[0]), -float64(v.eye[2])) * 180 / math.Pi
		if math.Abs(angle-tc.angle) > .0001 {
			t.Fatal(tc.fps, angle, tc.angle)
		}
	}
}

func TestNativeCameraCombinedOrbitAtPole(t *testing.T) {
	// Three seconds of Up+Right executed in Game.exe 0x4397d0. At the
	// pole, the native rejection stops both axes, including yaw.
	for _, tc := range []struct {
		fps int
		eye render.V3
	}{
		{30, render.V3{-1.640632, 5.290797, .5971413}},
		{60, render.V3{-1.6406314, 5.2907934, .5971416}},
		{144, render.V3{-1.6733123, 5.2805467, .5884455}},
	} {
		v := &worldView{facing: render.V3{0, 0, 1}}
		v.resetCamera()
		for range tc.fps * 3 {
			yaw, pitch := v.cameraInput(&platform.Input{Down: "^>"}, 1/float64(tc.fps), 0, 0)
			v.orbitCamera(yaw, pitch)
		}
		if v.eye.Sub(tc.eye.Add(render.V3{0, nativeCharacterOriginHeight, 0})).Len() > .0001 {
			t.Fatalf("%d FPS combined orbit: got %v, native %v", tc.fps, v.eye, tc.eye)
		}
	}
}

func TestCameraCapturedDisplacementIgnoresUIPosition(t *testing.T) {
	v := &worldView{}
	in := platform.Input{RightDown: true, MouseLook: true, MouseX: 400, MouseY: 300, MouseDX: 100, MouseDY: -20}
	yaw, pitch := v.cameraInput(&in, 1.0/60, -10000, 20000)
	wantYaw, wantPitch := v.cameraInput(&platform.Input{RightDown: true, MouseX: 100, MouseY: -20}, 1.0/60, 0, 0)
	if yaw != wantYaw || pitch != wantPitch {
		t.Fatal("captured movement used the UI anchor", yaw, pitch, wantYaw, wantPitch)
	}
}

func TestCameraGroundLimitPreservesHorizontalOrbit(t *testing.T) {
	yawed := render.V3{-3, .5, -4}
	candidate := render.V3{-2.99, .38, -3.99}
	for range 1000 {
		got := cameraOrbitFloor(yawed, candidate, .5)
		if got != yawed {
			t.Fatal("ground constraint changed horizontal orbit", got)
		}
		candidate = got.Add(render.V3{.01, -.12, .01})
	}
	if got := cameraOrbitFloor(yawed, render.V3{-3, 1, -4}, .5); got != (render.V3{-3, 1, -4}) {
		t.Fatal("valid pitch was rejected", got)
	}
}

func TestNativeCameraRepeatedGroundPitch(t *testing.T) {
	// Flat-floor results from three seconds of Down in Game.exe 0x4397d0.
	for _, tc := range []struct {
		fps int
		z   float32
	}{{30, -5.0986977}, {60, -5.098697}, {144, -5.098986}} {
		v := &worldView{pos: render.V3{0, -nativeCharacterOriginHeight, 0}, facing: render.V3{0, 0, 1}}
		v.resetCamera()
		for range tc.fps * 3 {
			previous := v.eye
			yaw, pitch := v.cameraInput(&platform.Input{Down: "v"}, 1/float64(tc.fps), 0, 0)
			v.orbitCamera(yaw, pitch)
			v.eye = cameraOrbitFloor(previous, v.eye, .5)
		}
		if v.eye.Sub(render.V3{0, .5, tc.z}).Len() > .0001 {
			t.Fatal("ground pitch diverged from native", tc.fps, v.eye, tc.z)
		}
	}
}

func TestFirstPersonPitchDirectionFromNative(t *testing.T) {
	// Game.exe 0x4397d0, eye=(0,.8,0), target=(0,.8,1): Up gives
	// target.y=.7927279, Down=.8072721, mouse dy=100 gives .6550682.
	for _, tc := range []struct {
		in   platform.Input
		down bool
	}{
		{platform.Input{Down: "^"}, true},
		{platform.Input{Down: "v"}, false},
		{platform.Input{RightDown: true, MouseY: 100}, true},
	} {
		v := &worldView{firstPerson: true, firstDirection: render.V3{0, 0, 1}}
		yaw, pitch := v.cameraInput(&tc.in, 1.0/60, 0, 0)
		v.orbitCamera(yaw, pitch)
		if (v.firstDirection[1] < 0) != tc.down {
			t.Fatal("first person pitch is inverted relative to native", tc.in, v.firstDirection)
		}
	}
}

func TestCameraResetUsesNativeModelOrigin(t *testing.T) {
	// Actual x86 execution of reset and local-player draw/restore instructions:
	// logical position Y=.9, rendered feet Y=0, eye Y=2.4000001, target Y=1.3999999.
	v := &worldView{facing: render.V3{0, 0, 1}}
	v.resetCamera()
	if v.eye.Sub(render.V3{0, 2.4000001, -5}).Len() > .000001 || v.target().Sub(render.V3{0, 1.3999999, 0}).Len() > .000001 {
		t.Fatal("reset used model feet as the native logical origin", v.eye, v.target())
	}
	v.cameraInput(&platform.Input{Keys: []platform.Key{platform.KeyF10}}, 1.0/60, 0, 0)
	if math.Abs(float64(v.eye[1]-1.7)) > .000001 {
		t.Fatal("first person eye is below native head height", v.eye)
	}
	if v.pos != (render.V3{}) {
		t.Fatal("camera conversion changed player coordinates", v.pos)
	}
}
