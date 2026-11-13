package scene

import (
	"math"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/effect"
)

func TestSEFFAlphaHasNativeRampHoldAndFade(t *testing.T) {
	d := effect.SEFFEmitter{LifetimeMS: 500, Alpha: .5}
	for _, tc := range []struct {
		age   float64
		alpha float32
	}{{0, .5}, {.1, .8}, {.5, 1}, {.75, .5}, {1, 0}, {2, 0}} {
		if got := weaponParticleAlpha(d, tc.age); math.Abs(float64(got-tc.alpha)) > 1e-5 {
			t.Fatal(tc, got)
		}
	}
}

func TestSEFFBirthIsRepeatableAndUsesNativeDimensions(t *testing.T) {
	d := effect.SEFFEmitter{Rate: 20, Shape: 3, Extent: 5, HalfSize: .1, LifetimeMS: 500, Alpha: .5, Local: true, Origin: [3]float32{0, 1, 0}}
	for seed := uint32(0); seed < 100; seed++ {
		pos, size, alpha := weaponParticle(d, 0, 0, seed)
		if math.Abs(float64(pos[0])) > .051 || math.Abs(float64(pos[2])) > .051 || pos[1] < .5 || pos[1] > 1.5 || size != .2 || alpha != .5 {
			t.Fatal(pos, size, alpha)
		}
		again, _, _ := weaponParticle(d, 0, 0, seed)
		if again != pos {
			t.Fatal("birth changed between camera passes")
		}
	}
}
