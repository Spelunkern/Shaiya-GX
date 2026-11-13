package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"math"
	"testing"
)

func TestCastIndicatorFollowsWorldProjection(t *testing.T) {
	view := render.LookAtLH(render.V3{0, 0, -10}, render.V3{}, render.V3{0, 1, 0})
	proj := render.PerspectiveFovLH(math.Pi/2, 1, .1, 100)
	x, y, ok := castBarOrigin(render.V3{}, view, proj, 100, 100)
	if !ok || x != 52 || y != 50 {
		t.Fatal("native half-unit offset/nearest-even rounding", x, y, ok)
	}
	x, y, ok = castBarOrigin(render.V3{1, 0, 0}, view, proj, 100, 100)
	if !ok || x != 58 || y != 50 {
		t.Fatal("indicator detached from player", x, y, ok)
	}
	if _, _, ok := castBarOrigin(render.V3{0, 0, -11}, view, proj, 100, 100); ok {
		t.Fatal("indicator drawn behind camera")
	}
}
