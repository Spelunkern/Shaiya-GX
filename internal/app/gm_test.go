package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"math"
	"testing"
)

func TestGMTurboChangesMovementOnlyWithServerRole(t *testing.T) {
	for _, tc := range []struct {
		gm, turbo bool
		index     uint8
		distance  float32
	}{
		{true, true, 2, 6}, {true, false, 2, .5}, {false, true, 2, .5}, {true, true, 255, 0},
	} {
		a := inputApp()
		a.state.Self.IsGM, a.state.GMTurbo = tc.gm, tc.turbo
		a.state.Self.Speed = &proto.CharacterSpeed{Move: tc.index}
		moving := tc.index != 255
		motion := uint8(0)
		if moving {
			motion = 1
		}
		a.wv = &worldView{pos: render.V3{100, 0, 100}, eye: render.V3{100, 2, 95}, facing: render.V3{0, 0, 1}, moving: moving, motion: motion, sendT: 100}
		a.updateWorld(&platform.Input{Down: "W"}, .1)
		if got := a.wv.pos[2] - 100; math.Abs(float64(got-tc.distance)) > .001 {
			t.Fatalf("%+v moved %v", tc, got)
		}
	}
}
