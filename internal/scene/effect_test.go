package scene

import (
	"math"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/effect"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

func TestEffectTextureClockLoopsOrClamps(t *testing.T) {
	d := &effect.Emitter{FrameDelay: .1}
	e := effectEmitter{definition: d, textures: make([]*render.Texture, 3)}
	e.advanceTexture(.1)
	if e.textureFrame != 0 {
		t.Fatal("advanced at the exact boundary")
	}
	e.advanceTexture(.01)
	if e.textureFrame != 1 {
		t.Fatal("did not advance after the boundary")
	}
	e.advanceTexture(.11)
	e.advanceTexture(.11)
	if e.textureFrame != 2 {
		t.Fatal("non-looping animation did not hold the last frame")
	}
	d.Unk08 = 1
	e.advanceTexture(.11)
	if e.textureFrame != 0 {
		t.Fatal("looping animation did not return to its first frame")
	}
}

func TestEffectStopDrainsAndBirthsRetainWorldPosition(t *testing.T) {
	d := &effect.Emitter{Loop: 1, InitialDelay: 1, Unk13: 1, Unk12: 60, Unk14: 60}
	x := &Effect{emitters: []effectEmitter{{definition: d}}, basis: render.Identity(), Origin: render.V3{1, 2, 3}}
	x.Update(1.0/60 + .001)
	if len(x.emitters[0].particles) != 1 {
		t.Fatal("expected one birth")
	}
	x.Origin = render.V3{5, 2, 3}
	x.Update(1.0 / 60)
	p := x.emitters[0].particles
	if len(p) != 2 || p[0].pos[0] != 1 || p[1].pos[0] != 5 {
		t.Fatalf("existing cloud followed moving anchor: %+v", p)
	}
	x.Stop()
	for i := 0; i < 5; i++ {
		x.Update(.25)
	}
	if !x.Done() {
		t.Fatal("stopped cast retained particles or emitted again")
	}
}

func TestEffectParticleLifetimeAndCapacity(t *testing.T) {
	d := &effect.Emitter{InitialDelay: .5, Unk13: .5, Unk12: 60, Unk14: 60, Unk16: .1}
	x := &Effect{emitters: []effectEmitter{{definition: d}}}
	x.Update(.25)
	if n := len(x.emitters[0].particles); n < 1 || n > 7 {
		t.Fatalf("finite emission count=%d", n)
	}
	x.Update(.25)
	x.Update(.25)
	if len(x.emitters[0].particles) != 0 {
		t.Fatal("expired particles retained")
	}
	d.Loop = 1
	d.Unk12, d.Unk14 = 1e30, 1e30
	x.Update(.25)
	if n := len(x.emitters[0].particles); n != 200 {
		t.Fatalf("native capacity=%d", n)
	}
	d.Loop = 0
	d.Unk12, d.Unk14, d.Unk16 = 1, 1, .05
	x = &Effect{emitters: []effectEmitter{{definition: d}}}
	x.Update(.1)
	if n := len(x.emitters[0].particles); n != 1 {
		t.Fatalf("one-shot emitted %d particles", n)
	}
}

func TestEffectCurvesUseAbsoluteParticleAge(t *testing.T) {
	d := &effect.Emitter{Colors: []effect.ColorFrame{{RGBA: [4]float32{0, 0, 1, 1}, Time: 0}, {RGBA: [4]float32{1, 1, 1, 0}, Time: 2}}, Sub3: []effect.Sub3Frame{{Unk1: 1, Unk2: 1, Time: 0}, {Unk1: 3, Unk2: 3, Time: 2}}}
	if c := particleColor(d, 1, 8); c != [4]float32{.5, .5, 1, .5} {
		t.Fatalf("color=%v", c)
	}
	if s := particleSize(d, 1, .3); s != 2 {
		t.Fatalf("size=%g", s)
	}
}

func TestEffectConstrainedFacing(t *testing.T) {
	close := func(a, b float32) bool { return math.Abs(float64(a-b)) < 1e-5 }
	view := render.LookAtLH(render.V3{0, 4, -4}, render.V3{0, 0, 0}, render.V3{0, 1, 0})
	horizontal := effectBaseBasis(1, view, render.Identity())
	if !close(horizontal[5], 0) || !close(horizontal[6], 1) || !close(horizontal[9], -1) {
		t.Fatal("horizontal effect basis", horizontal)
	}
	for _, f := range []render.V3{{0, 0, 1}, {1, 0, 0}, {0, 0, -1}, {-1, 0, 0}} {
		view = render.LookAtLH(render.V3{}, f.Add(render.V3{0, .5, 0}), render.V3{0, 1, 0})
		upright := effectBaseBasis(2, view, render.Identity())
		if !close(upright[8], f[0]) || !close(upright[10], f[2]) || !close(upright[5], 1) {
			t.Fatal("upright effect basis", f, upright)
		}
	}
}

func TestOrbitMotionUsesOffsetAndNativeAxes(t *testing.T) {
	near := func(a, b float32) bool { return math.Abs(float64(a-b)) < 1e-5 }
	for _, tc := range []struct {
		mode     int32
		in, want render.V3
	}{
		{1, render.V3{1, 0, 0}, render.V3{0, 0, 1}},
		{2, render.V3{1, 0, 0}, render.V3{0, -1, 0}},
		{3, render.V3{0, 1, 0}, render.V3{0, 0, -1}},
	} {
		got := orbitStep(tc.in, tc.mode, 1, 0)
		for i := range got {
			if !near(got[i], tc.want[i]) {
				t.Fatalf("mode %d: %v, want %v", tc.mode, got, tc.want)
			}
		}
	}
	definition := &effect.Emitter{Unk06: 1, Unk17: math.Pi * 30, InitialDelay: 2, Unk13: 2, Unk12: 60, Unk14: 60, Unk16: 1, Spread1: [3]float32{1, 0, 0}, Spread2: [3]float32{1, 0, 0}}
	x := &Effect{emitters: []effectEmitter{{definition: definition}}, basis: render.Identity()}
	x.Update(1.0 / 60)
	x.Update(1.0 / 60)
	p := x.emitters[0].particles[0]
	if !near(p.pos.Len(), 0) || !near(p.velocity[2], 1) || !near(p.velocity[0], 1.0/60) {
		t.Fatal("orbit must rotate its offset and add drift without moving its base", p)
	}
	if reach := emitterReach(definition, 1); math.IsInf(float64(reach), 0) || reach < p.velocity.Len() {
		t.Fatal("orbit outside conservative bounds", reach, p.velocity)
	}
}

func TestEffectForcesApplyAfterMotion(t *testing.T) {
	d := &effect.Emitter{Unk20: 1, Unk21: 1, Trembling: [3]float32{0, -6, 0}, UnkVec6: [3]float32{0, 0, 4}, RotationMin: 12, InitialDelay: 3, Unk13: 3, Unk12: 60, Unk14: 60, Unk16: 1}
	basis := effectRotation(2, math.Pi/2)
	x := &Effect{emitters: []effectEmitter{{definition: d}}, basis: basis, Origin: render.V3{100, 0, 200}}
	x.Update(1.0 / 60)
	x.Update(1.0 / 60)
	p := x.emitters[0].particles[0]
	if math.Abs(float64(p.velocity[0]-.2)) > 1e-5 || math.Abs(float64(p.velocity[1]+.1)) > 1e-5 || math.Abs(float64(p.velocity[2])) > 1e-5 {
		t.Fatal("attraction or constant force used incorrect axis/order", p.velocity)
	}
	if p.pos != x.Origin {
		t.Fatal("force moved position in the same step", p.pos)
	}
	x.Update(1.0 / 60)
	p = x.emitters[0].particles[0]
	if p.pos[0] <= x.Origin[0] || p.pos[1] >= x.Origin[1] {
		t.Fatal("force did not move particle on next step", p.pos)
	}
	if reach := emitterReach(d, 1); math.IsInf(float64(reach), 0) || reach < p.pos.Sub(x.Origin).Len() {
		t.Fatal("force exceeded culling bound", reach)
	}
}

func TestStationaryZeroLifeFollowsBillboardAndStops(t *testing.T) {
	d := &effect.Emitter{Loop: 1, Unk12: 1, Unk14: 1, Unk16: .02, Colors: []effect.ColorFrame{{RGBA: [4]float32{0, 0, 0, .2}}, {Time: 1, RGBA: [4]float32{1, 1, 1, .6}}}}
	x := &Effect{emitters: []effectEmitter{{definition: d}}, basis: render.Identity(), Origin: render.V3{1, 2, 3}}
	x.Update(.05)
	if len(x.emitters[0].particles) != 1 {
		t.Fatal("static particle was not emitted")
	}
	x.Origin = render.V3{4, 2, 3}
	x.Update(1.0/60 + .0001)
	p := x.emitters[0].particles[0]
	if p.pos != x.Origin || p.age != 0 || p.staticTime <= 0 {
		t.Fatalf("zero-life position/clock: %+v", p)
	}
	c := staticParticleColor(d, 2)
	if c[0] != 2 || math.Abs(float64(c[3]-1)) > 1e-6 {
		t.Fatal("native color slope was clamped", c)
	}
	x.Stop()
	x.Update(1.0/60 + .0001)
	if !x.Done() {
		t.Fatal("static particles did not stop")
	}
}

func TestStationaryZeroLifeSizeDoesNotAdvancePastSecondKey(t *testing.T) {
	d := &effect.Emitter{Sub3: []effect.Sub3Frame{{Unk1: 1, Unk2: 3}, {Time: 1, Unk1: 4, Unk2: 8}, {Time: 2, Unk1: 100, Unk2: 100}}}
	if v := staticParticleSize(d, 0, .5, .25); v != 2 {
		t.Fatalf("initial size %v", v)
	}
	if v := staticParticleSize(d, .5, .5, .25); v != 3.5 {
		t.Fatalf("intermediate size %v", v)
	}
	if v := staticParticleSize(d, 10, .5, .25); v != 5 {
		t.Fatalf("target size %v", v)
	}
}

func TestZeroLifeWithoutColorsBecomesWhiteAfterFirstUpdate(t *testing.T) {
	d := &effect.Emitter{}
	if got := staticParticleColor(d, 0); got != ([4]float32{}) {
		t.Fatalf("birth color %v", got)
	}
	if got := staticParticleColor(d, 1.0/60); got != ([4]float32{1, 1, 1, 1}) {
		t.Fatalf("updated color %v", got)
	}
}

func TestMovingZeroLifeParticlesRespectDifferentAnchors(t *testing.T) {
	for _, mesh := range []bool{false, true} {
		d := &effect.Emitter{Mesh: -1, Unk12: 1, Unk14: 1, Unk16: .01, Spread1: [3]float32{1, 0, 0}, Spread2: [3]float32{1, 0, 0}, Textures: []int32{0}}
		e := effectEmitter{definition: d}
		if mesh {
			d.Mesh = 0
			e.geometry = &effectGeometry{source: &effect.Mesh3DE{}}
		}
		x := &Effect{emitters: []effectEmitter{e}, basis: render.Identity()}
		x.Update(.2)
		if len(x.emitters[0].particles) != 1 {
			t.Fatalf("mesh=%v: no particle", mesh)
		}
		p := x.emitters[0].particles[0]
		reach := emitterReach(d, 1)
		if mesh {
			if p.pos[0] < .15 || !math.IsInf(float64(reach), 1) {
				t.Fatalf("mesh moved %v, bound %v", p.pos, reach)
			}
		} else if p.pos[0] > .017 || math.IsInf(float64(reach), 0) || p.pos.Len() > reach {
			t.Fatalf("billboard pos %v bound %v", p.pos, reach)
		}
		x.Stop()
		x.Update(.02)
		if !x.Done() {
			t.Fatalf("mesh=%v: particle retained after stop", mesh)
		}
	}
}

func TestOneShotEmitterDoesNotMissFloatBoundaries(t *testing.T) {
	for _, duration := range []float32{.05, .1, .2} {
		d := &effect.Emitter{InitialDelay: 2, Unk13: 2, Unk12: 1, Unk14: 1, Unk16: duration}
		x := &Effect{emitters: []effectEmitter{{definition: d}}, basis: render.Identity()}
		x.Update(.25)
		if got := len(x.emitters[0].particles); got != 1 {
			t.Fatalf("duration %v emitted %d particles", duration, got)
		}
	}
}
