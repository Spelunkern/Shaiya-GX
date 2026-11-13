package scene

import (
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"math"
	"testing"
)

func TestDungeonOcclusionLeavesDoorwayOpenAndUsesCurrentCamera(t *testing.T) {
	wall := func(x0, x1 float32) [][3]render.V3 {
		a, b, c, d := render.V3{x0, -10, 10}, render.V3{x1, -10, 10}, render.V3{x1, 10, 10}, render.V3{x0, 10, 10}
		return [][3]render.V3{{a, b, c}, {a, c, d}}
	}
	w := &World{dg: []dgPart{{occluders: append(wall(-20, -2), wall(2, 20)...)}}}
	w.buildDungeonOccluders()
	r := &render.Scene3D{View: render.Identity(), Proj: render.PerspectiveFovLH(math.Pi/4, 16.0/9, .1, 100)}
	eye := render.V3{}
	w.PrepareTerrainOcclusion(r, eye)
	if r.Occlusion == nil {
		t.Fatal("dungeon depth mask missing")
	}
	if r.OccludedBox(render.V3{-.2, -.2, 20}, render.V3{.2, .2, 21}) {
		t.Fatal("doorway was sealed")
	}
	if !r.OccludedBox(render.V3{8, -.2, 20}, render.V3{8.2, .2, 21}) {
		t.Fatal("solid wall does not occlude")
	}
	first := r.Occlusion
	w.PrepareTerrainOcclusion(r, eye)
	if r.Occlusion != first {
		t.Fatal("stationary camera rebuilt mask storage")
	}
	r.View = render.LookAtLH(render.V3{0, 0, 30}, render.V3{0, 0, 40}, render.V3{0, 1, 0})
	w.PrepareTerrainOcclusion(r, render.V3{0, 0, 30})
	if r.OccludedBox(render.V3{8, -.2, 45}, render.V3{8.2, .2, 46}) {
		t.Fatal("old camera mask hid visible geometry")
	}
	r.ClipPlane = [4]float32{0, 1, 0, 0}
	w.PrepareTerrainOcclusion(r, eye)
	if r.Occlusion != nil {
		t.Fatal("main mask reused for clipped reflection")
	}
}
