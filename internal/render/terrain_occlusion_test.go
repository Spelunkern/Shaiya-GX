package render

import (
	"math"
	"testing"
)

func TestTerrainOcclusionRequiresWholeBounds(t *testing.T) {
	o := NewTerrainOcclusion(160, 90)
	view := Identity()
	proj := PerspectiveFovLH(math.Pi/2, 16.0/9, .1, 100)
	o.Reset(view, proj)
	o.Triangle(V3{-20, -20, 5}, V3{20, -20, 5}, V3{0, 20, 5})
	if !o.Box(V3{-.2, -.2, 10}, V3{.2, .2, 11}) {
		t.Fatal("fully hidden object not culled")
	}
	for _, box := range [][2]V3{{{-.2, -.2, 2}, {.2, .2, 3}}, {{-.2, -.2, .05}, {.2, .2, 11}}, {{-100, -.2, 10}, {.2, .2, 11}}} {
		if o.Box(box[0], box[1]) {
			t.Fatal("visible/near-clipped/partially covered bounds culled", box)
		}
	}
	s := &Scene3D{View: view, Proj: proj, Occlusion: o}
	s.ClipPlane = [4]float32{0, 1, 0, 0}
	if s.OccludedBox(V3{-.2, -.2, 10}, V3{.2, .2, 11}) {
		t.Fatal("reflection inherited main-camera mask")
	}
	o.Reset(view, proj)
	if o.Box(V3{-.2, -.2, 10}, V3{.2, .2, 11}) {
		t.Fatal("stale visibility survived reset")
	}
}
func TestSteppedTerrainHidesBelowHillTop(t *testing.T) {
	o := NewTerrainOcclusion(160, 90)
	view := LookAtLH(V3{32, 5, -20}, V3{32, 4, 60}, V3{0, 1, 0})
	proj := PerspectiveFovLH(math.Pi/2, 16.0/9, .1, 200)
	o.Reset(view, proj)
	o.FlatQuad(V3{0, 10, 0}, V3{64, 10, 64}, 10)
	lo, hi := V3{31, 1, 70}, V3{33, 3, 72}
	if o.Box(lo, hi) {
		t.Fatal("top above the ray falsely occluded")
	}
	o.PlanarQuad(V3{0, 0, 0}, V3{64, 0, 0}, V3{64, 10, 0}, V3{0, 10, 0})
	if !o.Box(lo, hi) {
		t.Fatal("hill front failed to occlude")
	}
}

func TestLabelOcclusionRequiresCompleteRectangle(t *testing.T) {
	o := NewTerrainOcclusion(160, 90)
	view, proj := Identity(), PerspectiveFovLH(math.Pi/2, 16.0/9, .1, 100)
	o.Reset(view, proj)
	o.PlanarQuad(V3{-20, -20, 5}, V3{20, -20, 5}, V3{20, 20, 5}, V3{-20, 20, 5})
	s := &Scene3D{View: view, Proj: proj, Occlusion: o}
	if !s.OccludedLabel(V3{0, 0, 10}, 600, 340, 80, 13, 1280, 720) {
		t.Fatal("hidden label submitted")
	}
	if s.OccludedLabel(V3{0, 0, 10}, 0, 0, 80, 13, 1280, 720) {
		t.Fatal("partially offscreen label culled")
	}
	if s.OccludedLabel(V3{0, 0, 2}, 600, 340, 80, 13, 1280, 720) {
		t.Fatal("foreground label culled")
	}
}
