package render

import (
	"math"
	"testing"
)

func TestFrustumRejectsOutsideAndKeepsIntersectingSpheres(t *testing.T) {
	for _, origin := range []V3{{}, {1703, 80, 1632}} {
		for _, aspect := range []float32{.5, 1, 3.5} {
			f := ViewFrustum(LookAtLH(origin, origin.Add(V3{0, 0, 1}), V3{0, 1, 0}), PerspectiveFovLH(math.Pi/2, aspect, 1, 100))
			for _, tc := range []struct {
				name    string
				center  V3
				radius  float32
				visible bool
			}{
				{"inside", V3{0, 0, 50}, 0, true},
				{"behind", V3{0, 0, -5}, 1, false},
				{"before near", V3{0, 0, .1}, .1, false},
				{"crossing near", V3{0, 0, .5}, .6, true},
				{"past far", V3{0, 0, 110}, 1, false},
				{"crossing far", V3{0, 0, 100.5}, 1, true},
				{"above", V3{0, 100, 50}, 1, false},
				{"below", V3{0, -100, 50}, 1, false},
				{"right", V3{100 * aspect, 0, 50}, 1, false},
				{"left", V3{-100 * aspect, 0, 50}, 1, false},
				{"crossing side", V3{51 * aspect, 0, 50}, 5, true},
			} {
				if got := f.SphereVisible(origin.Add(tc.center), tc.radius); got != tc.visible {
					t.Errorf("%s origin=%v aspect=%v got=%v", tc.name, origin, aspect, got)
				}
			}
		}
	}
}

func TestClipBoundsAreConservative(t *testing.T) {
	plane := [4]float32{0, 2, 0, -4} // y >= 2
	for _, tc := range []struct {
		min, max V3
		visible  bool
	}{
		{V3{-1, -2, -1}, V3{1, 1, 1}, false},
		{V3{-1, -2, -1}, V3{1, 2, 1}, true},
		{V3{-1, 1, -1}, V3{1, 3, 1}, true},
	} {
		if got := ClipBoxVisible(plane, tc.min, tc.max); got != tc.visible {
			t.Errorf("box %v..%v got %v", tc.min, tc.max, got)
		}
	}
	if ClipSphereVisible(plane, V3{0, 0, 0}, 1) || !ClipSphereVisible(plane, V3{0, 0, 0}, 2) || !ClipSphereVisible(plane, V3{0, 0, 0}, float32(math.Inf(1))) {
		t.Fatal("sphere clipping lost a touching or unbounded sphere")
	}
	if !ClipBoxVisible([4]float32{}, V3{}, V3{}) || !ClipSphereVisible([4]float32{}, V3{}, 0) {
		t.Fatal("disabled clip plane rejected geometry")
	}
}

func TestFrustumFlatWaterBounds(t *testing.T) {
	projection := PerspectiveFovLH(math.Pi/4, 1, .1, 100)
	for _, tc := range []struct {
		name        string
		eye, target V3
		visible     bool
	}{
		{"toward surface", V3{50, 10, 10}, V3{50, 0, 50}, true},
		{"away from surface", V3{50, 10, 10}, V3{50, 20, -10}, false},
		{"below looking up", V3{50, -10, 10}, V3{50, 0, 50}, true},
		{"outside map", V3{200, 10, 10}, V3{200, 0, 50}, false},
		{"crossing edge", V3{105, 10, 10}, V3{100, 0, 50}, true},
	} {
		f := ViewFrustum(LookAtLH(tc.eye, tc.target, V3{0, 1, 0}), projection)
		if got := f.BoxVisible(V3{0, 0, 0}, V3{100, 0, 100}); got != tc.visible {
			t.Errorf("%s: got %v, want %v", tc.name, got, tc.visible)
		}
	}
}
