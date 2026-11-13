package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"testing"
)

func TestMagicProjectileKeepsInitialSpeedAndFollowsTarget(t *testing.T) {
	p := skillProjectile{age: .5, delay: .4}
	p.move(render.V3{10, 0, 0}, .25) // Only .1 s of this frame follows the .4 s windup.
	if p.pos[0] < 4.999 || p.pos[0] > 5.001 || p.arrived {
		t.Fatalf("launch=%+v", p)
	}
	p.age = .6
	p.move(render.V3{11, 0, 0}, .1)
	if p.pos[0] < 9.999 || p.pos[0] > 10.001 || p.speed != 50 || p.arrived {
		t.Fatalf("homing changed initial speed: %+v", p)
	}
	p.age = .7
	p.move(render.V3{11, 0, 0}, .1)
	if !p.arrived || p.pos[0] != 11 {
		t.Fatalf("arrival=%+v", p)
	}
}
func TestMagicProjectileZeroDistanceArrives(t *testing.T) {
	p := skillProjectile{age: .4}
	p.move(render.V3{}, .016)
	if !p.arrived || p.pos != (render.V3{}) {
		t.Fatalf("zero distance=%+v", p)
	}
}

func TestLateMagicProjectileCatchesUpFlight(t *testing.T) {
	for _, elapsed := range []float32{.45, .5, .6, .7} {
		p := skillProjectile{age: elapsed, delay: .4}
		p.move(render.V3{10, 0, 0}, elapsed-.4)
		if elapsed < .6 && p.arrived {
			t.Fatal("premature impact", elapsed)
		}
		if elapsed >= .6 && !p.arrived {
			t.Fatal("late result replayed flight", elapsed, p)
		}
	}
}
