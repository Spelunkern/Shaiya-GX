package scene

import (
	"math"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/data/effect"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

func TestVelocityCurveNativeKeyBoundariesAndTail(t *testing.T) {
	keys := []effect.OpacityFrame{{Opacity: 0, Time: 0}, {Opacity: 2, Time: 1}, {Opacity: 0, Time: 2}}
	var c velocityCurve
	for _, tick := range []struct{ age, dt, want float32 }{
		{0, .25, 0}, {.25, .25, .5}, {.5, .25, 1}, {1, .5, 2},
		{1.25, .25, 1.5}, {2, .75, 0}, {2.25, .25, -.5},
	} {
		if got := c.step(keys, tick.age, tick.dt); got != tick.want {
			t.Fatalf("age %g: acceleration=%g, want %g", tick.age, got, tick.want)
		}
	}
	// A single native key has no segment and leaves the zero initial state.
	c = velocityCurve{}
	if got := c.step([]effect.OpacityFrame{{Opacity: 5}}, 1, .25); got != 0 {
		t.Fatal("single key invented acceleration", got)
	}
	// Even when the clock passes several keys, the original visits one per tick.
	c = velocityCurve{}
	c.step(keys, 10, .25)
	if c.key != 1 {
		t.Fatal("skipped native segment progression", c.key)
	}
}

func TestParticleCurveChangesVelocityAfterMovementAndKeepsAlpha(t *testing.T) {
	d := &effect.Emitter{
		Loop: 1, InitialDelay: 1, Unk13: 1, Unk12: 60, Unk14: 60,
		Spread1: [3]float32{1, 0, 0}, Spread2: [3]float32{1, 0, 0},
		Opacity: []effect.OpacityFrame{{Opacity: 1}, {Opacity: 1, Time: 2}},
		Colors:  []effect.ColorFrame{{RGBA: [4]float32{1, .5, .25, .75}}},
	}
	x := &Effect{emitters: []effectEmitter{{definition: d}}, basis: render.Identity()}
	x.Update(1.0/60 + .001)
	x.Update(1.0 / 60)
	p := x.emitters[0].particles[0]
	if math.Abs(float64(p.pos[0]-1.0/60)) > 1e-6 || math.Abs(float64(p.velocity[0]-(1+1.0/60))) > 1e-6 {
		t.Fatal("curve changed position before the native movement step", p.pos, p.velocity)
	}
	if color := particleColor(d, p.age, p.life); color != d.Colors[0].RGBA {
		t.Fatal("velocity curve changed particle alpha", color)
	}
	if x.emitters[0].particles[1].curve != (velocityCurve{}) {
		t.Fatal("new particle inherited another particle's curve state")
	}
}

func TestRealVelocityCurveSimulation(t *testing.T) {
	if os.Getenv("SHAIYA_DATA") == "" {
		t.Skip("SHAIYA_DATA not set")
	}
	a, err := archive.Open(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	count := 0
	meshes := map[string]bool{}
	a.Walk(func(file string, _ *archive.File) {
		if !strings.HasSuffix(strings.ToLower(file), ".eft") {
			return
		}
		raw, err := a.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		l, err := effect.ParseEFT(raw)
		if err != nil {
			t.Fatal(file, err)
		}
		for i := range l.Effects {
			d := &l.Effects[i]
			if len(d.Opacity) == 0 || !d.LinearSupported() {
				continue
			}
			if id := d.Textures[0]; id < 0 || int(id) >= len(l.Textures) {
				t.Fatalf("%s emitter %d: invalid texture index", file, i)
			} else if name := "Effect/DDS/" + path.Base(strings.ReplaceAll(l.Textures[id], `\`, "/")); !hasArchiveFile(a, name) {
				t.Fatal("missing velocity-curve texture", name)
			}
			if d.UsesMesh() {
				if int(d.Mesh) >= len(l.Meshes) {
					t.Fatalf("%s emitter %d: invalid mesh index", file, i)
				}
				name := "Effect/3DE/" + path.Base(strings.ReplaceAll(l.Meshes[d.Mesh], `\`, "/"))
				if !meshes[name] {
					raw, err := a.ReadFile(name)
					if err != nil {
						t.Fatal(err)
					}
					if _, err := effect.Parse3DE(raw); err != nil {
						t.Fatal(name, err)
					}
					meshes[name] = true
				}
			}
			count++
			x := &Effect{emitters: []effectEmitter{{definition: d}}, basis: render.Identity()}
			for tick := 0; tick < 120; tick++ {
				x.Update(1.0/60 + 1e-6)
				for _, p := range x.emitters[0].particles {
					for _, value := range []render.V3{p.pos, p.velocity} {
						for _, v := range value {
							if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
								t.Fatalf("%s emitter %d: non-finite curve simulation", file, i)
							}
						}
					}
				}
			}
		}
	})
	if count == 0 {
		t.Fatal("no original velocity-curve emitters simulated")
	}
	t.Logf("%d newly supported emitters simulated for two seconds", count)
}

func hasArchiveFile(a *archive.Archive, name string) bool {
	_, ok := a.Lookup(name)
	return ok
}
