package scene

import (
	"math"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/effect"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

func TestMapEffectRangeIncludesFarFrustumCorners(t *testing.T) {
	projection := render.PerspectiveFovLH(math.Pi/4, 16.0/9, .1, 600)
	if !mapEffectInRange(projection, 500*500, 128) {
		t.Fatal("effect inside the far plane was culled at the old 300-unit limit")
	}
	if !mapEffectInRange(projection, 700*700, 128) {
		t.Fatal("visible wide-angle frustum corner was culled")
	}
	if mapEffectInRange(projection, 2000*2000, 128) {
		t.Fatal("distant effect remained in range")
	}
	if !mapEffectInRange(render.Identity(), 1e10, 128) {
		t.Fatal("unknown projection incorrectly culled an effect")
	}
}

func TestMapEffectPrecomputedReachMatchesOriginal(t *testing.T) {
	projections := []render.M4{render.Identity(), render.PerspectiveFovLH(math.Pi/4, 16.0/9, .1, 600), render.PerspectiveFovLH(math.Pi/2, 1, .5, 1000)}
	for _, p := range projections {
		for _, radius := range []float32{0, 2, 200, float32(math.Inf(1))} {
			for _, distance := range []float32{0, 100, 300000, 1000000, 1e10} {
				denominator := 1 - p[10]
				want := true
				if denominator < 0 && p[14] < 0 && p[0] != 0 && p[5] != 0 {
					far := p[14] / denominator
					if far > 0 && !math.IsInf(float64(far), 0) && !math.IsNaN(float64(far)) {
						x, y := float64(1/p[0]), float64(1/p[5])
						reach := far*float32(math.Sqrt(1+x*x+y*y)) + radius
						want = distance <= reach*reach
					}
				}
				reach, valid := mapEffectCameraReach(p)
				if got := mapEffectWithinReach(reach, valid, distance, radius); got != want {
					t.Fatalf("projection=%v radius=%v distance²=%v: got %v want %v", p, radius, distance, got, want)
				}
			}
		}
	}
}

func BenchmarkMapEffectRange(b *testing.B) {
	p := render.PerspectiveFovLH(math.Pi/4, 16.0/9, .1, 600)
	for _, precomputed := range []bool{false, true} {
		name := "per effect"
		if precomputed {
			name = "per camera"
		}
		b.Run(name, func(b *testing.B) {
			count := 0
			for range b.N {
				reach, valid := mapEffectCameraReach(p)
				for i := range 1000 {
					var inRange bool
					if precomputed {
						inRange = mapEffectWithinReach(reach, valid, float32(i*i), 5)
					} else {
						inRange = mapEffectInRange(p, float32(i*i), 5)
					}
					if inRange {
						count++
					}
				}
			}
			if count == 0 {
				b.Fatal("invalid benchmark")
			}
		})
	}
}

func TestEffectReachBoundsLinearMotionAndUncertainData(t *testing.T) {
	d := &effect.Emitter{InitialDelay: 2, Unk13: 2, Position: [3]float32{3, 0, 0}, OffsetFrame: [3]float32{2, 0, 0}, Spread1: [3]float32{4, 0, 0}, Spread2: [3]float32{4, 0, 0}}
	if reach := emitterReach(d, 1); reach < 14 {
		t.Fatalf("effect movement or extent was excluded: %v", reach)
	}
	d.Opacity = []effect.OpacityFrame{{Time: 1}, {Time: 1}}
	if !math.IsInf(float64(emitterReach(d, 1)), 1) {
		t.Fatal("invalid curve used a finite bound")
	}
	d.Opacity = nil
	if !math.IsInf(float64(emitterReach(d, math.Inf(1))), 1) {
		t.Fatal("unknown mesh used a finite bound")
	}
}
