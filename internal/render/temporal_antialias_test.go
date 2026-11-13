package render

import (
	"math"
	"testing"
)

func TestTemporalJitterAndHistoryCuts(t *testing.T) {
	for i := uint64(0); i < 16; i++ {
		j := TemporalJitter(i, 640, 360)
		if math.Abs(float64(j[0]*640)) > 1 || math.Abs(float64(j[1]*360)) > 1 {
			t.Fatal("jitter exceeds half a pixel")
		}
		if j != TemporalJitter(i+8, 640, 360) {
			t.Fatal("sequence does not repeat")
		}
	}
	p := PerspectiveFovLH(math.Pi/4, 16.0/9, .1, 500)
	view := Identity()
	moved := LookAtLH(V3{0, 0, 1}, V3{0, 0, 2}, V3{0, 1, 0})
	if !temporalHistoryCompatible(moved, p, view, p) {
		t.Fatal("normal movement resets history")
	}
	moved = LookAtLH(V3{0, 0, 30}, V3{0, 0, 31}, V3{0, 1, 0})
	if temporalHistoryCompatible(moved, p, view, p) {
		t.Fatal("teleport keeps history")
	}
	turn := LookAtLH(V3{}, V3{0, 0, -1}, V3{0, 1, 0})
	if temporalHistoryCompatible(turn, p, view, p) {
		t.Fatal("camera cut keeps history")
	}
	other := p
	other[0] *= 2
	if temporalHistoryCompatible(view, other, view, p) {
		t.Fatal("FOV change keeps history")
	}
	s := &Scene3D{Proj: p, TemporalJitter: TemporalJitter(1, 640, 360), ReverseDepth: true}
	if s.Proj != p || s.VisibilityProjection() == p {
		t.Fatal("jitter is missing or changed UI projection")
	}
	s.Shadow = true
	if s.VisibilityProjection() != p {
		t.Fatal("shadow capture jittered")
	}
}
