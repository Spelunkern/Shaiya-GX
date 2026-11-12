package app

import (
	"math"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/session"
)

func TestCameraKeepsOrbitAndZoomAcrossElevationChanges(t *testing.T) {
	v := &worldView{pos: render.V3{500, 10, 1700}, eye: render.V3{497, 14, 1695}}
	v.rotateEye(.6, .2)
	offset := v.eye.Sub(v.pos)
	for _, pos := range []render.V3{{501, 12, 1701}, {503, 35, 1705}, {504, 2, 1704}, {800, 78, 1600}} {
		v.place(pos)
		if got := v.eye.Sub(v.pos).Sub(offset).Len(); got > .001 {
			t.Fatalf("elevation change lost the camera orbit/zoom: %v", v.eye.Sub(v.pos))
		}
	}
}

func TestMountedMovementSpeed(t *testing.T) {
	if got := playerMovementSpeed(&proto.CharacterSpeed{Move: 0}, true); got != 7 {
		t.Fatalf("mounted speed = %v, want 7", got)
	}
	if got := playerMovementSpeed(&proto.CharacterSpeed{Move: 0}, false); got != 3 {
		t.Fatalf("unmounted speed = %v, want 3", got)
	}
	if got := swimmingMovementSpeed(playerMovementSpeed(nil, true), true); got != 2.8 {
		t.Fatalf("common mount swimming speed = %v, want 2.8", got)
	}
	if got := swimmingMovementSpeed(0, true); got != 0 {
		t.Fatalf("immobilized swimmer speed = %v, want 0", got)
	}
}

func TestSwimmingDirectionFollowsPitchOnlyForForwardAndBack(t *testing.T) {
	horizontal := render.V3{0, 0, 1}
	down := render.V3{0, -1, 1}
	if d := swimDirection(horizontal, down, true, false); d[1] >= 0 {
		t.Fatalf("forward should dive: %v", d)
	}
	if d := swimDirection(horizontal, down, false, true); d[1] <= 0 {
		t.Fatalf("backward should rise: %v", d)
	}
	if d := swimDirection(horizontal, down, false, false); d[1] != 0 {
		t.Fatalf("strafe should stay level: %v", d)
	}
	if d := swimDirection(horizontal, render.V3{0, -.1, 1}, true, false); d[1] != 0 {
		t.Fatalf("small downward pitch should stay level: %v", d)
	}
}

func TestServerPositionReconcilesBeforeNextPredictionFrame(t *testing.T) {
	a := inputApp()
	a.state.MapID, a.state.SelfID = 1, 10
	a.wv = &worldView{mapID: 1, pos: render.V3{10, 4, 20}, eye: render.V3{10, 6, 15}, moving: true}
	packets := make(chan []byte, 1)
	a.world = &session.World{Packets: packets}
	w := proto.NewWriter(proto.CharacterMove)
	w.U32(10)
	w.U8(0)
	w.U16(16384)
	w.F32(30)
	w.F32(50)
	w.F32(60)
	packets <- w.Bytes()
	a.pumpWorld()
	a.updateWorld(&platform.Input{}, .016)
	if a.wv.pos != (render.V3{30, 50, 60}) || a.state.Self.X != 30 || a.wv.moving {
		t.Fatal("prediction overwrote the server correction", a.wv.pos, a.state.Self)
	}
	if a.wv.eye != (render.V3{30, 52, 55}) || math.Abs(float64(a.wv.facing[0]-1)) > .001 {
		t.Fatal("correction lost camera offset or facing")
	}
}

func TestOtherCharacterTeleportDoesNotStopOurMovement(t *testing.T) {
	a := inputApp()
	a.state.MapID, a.state.SelfID = 1, 10
	a.wv = &worldView{mapID: 1, pos: render.V3{10, 4, 20}, eye: render.V3{10, 6, 15}, moving: true}
	before := *a.wv
	w := proto.NewWriter(proto.CharacterMapTeleport)
	w.U32(20)
	w.U16(2)
	w.F32(300)
	w.F32(50)
	w.F32(600)
	if _, err := a.state.Apply(w.Bytes()); err != nil {
		t.Fatal(err)
	}
	a.confirmedPosition(w.Bytes())
	if a.wv.pos != before.pos || a.wv.eye != before.eye || !a.wv.moving {
		t.Fatal("another character's teleport interrupted the local view")
	}
}

func TestNativeMovementSpeedIncludesBlockedAndUnknownIndices(t *testing.T) {
	for index, want := range map[uint8]float32{0: 3, 1: 4, 2: 5, 3: 6, 4: 7, 255: 0, 17: 5} {
		if got := movementSpeed(&proto.CharacterSpeed{Move: index}); got != want {
			t.Errorf("speed %d: got %f want %f", index, got, want)
		}
	}
	if movementSpeed(nil) != 5 {
		t.Fatal("missing speed should retain the native initial rate")
	}
}

func TestImmobilizedInputCannotMoveOrInterruptTheCamera(t *testing.T) {
	a := inputApp()
	a.state.Self.Speed = &proto.CharacterSpeed{Move: 255}
	a.wv = &worldView{pos: render.V3{10, 4, 20}, eye: render.V3{10, 6, 15}}
	a.updateWorld(&platform.Input{Down: "W"}, .5)
	if a.wv.pos != (render.V3{10, 4, 20}) || a.wv.eye != (render.V3{10, 6, 15}) || a.wv.moving {
		t.Fatal("blocked movement predicted a step")
	}
	a.updateWorld(&platform.Input{Down: ">"}, .1)
	if a.wv.pos != (render.V3{10, 4, 20}) || a.wv.eye == (render.V3{10, 6, 15}) {
		t.Fatal("immobilization also blocked camera orbit")
	}
}

func TestRemoteWalkInferenceExpiresAndDistinguishesRunStop(t *testing.T) {
	r := &remoteActor{moveX: 10, moveZ: 20}
	e := &game.Entity{X: 10, Z: 20}
	if w, m := r.observeMovement(e, 1); w || m {
		t.Fatal("stationary actor moves")
	}
	e.MoveSerial++
	e.X = 10.5
	if w, m := r.observeMovement(e, 2); !w || !m {
		t.Fatal("position change in walk mode not recognized")
	}
	if w, m := r.observeMovement(e, 2.8); w || m {
		t.Fatal("walk animation did not expire")
	}
	e.MoveSerial++
	e.Motion = 1
	e.X = 11
	if w, m := r.observeMovement(e, 3); w || !m {
		t.Fatal("run motion lost")
	}
	e.MoveSerial++
	e.Motion = 0
	e.X = 11.2
	if w, m := r.observeMovement(e, 3.1); w || m {
		t.Fatal("run stop misread as walk")
	}
	e.MoveSerial++
	e.X = 11.5
	if w, m := r.observeMovement(e, 3.6); !w || !m {
		t.Fatal("walk after run not recognized")
	}
}
