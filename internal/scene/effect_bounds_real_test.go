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

func TestRealWorldEffectParticleBounds(t *testing.T) {
	if os.Getenv("SHAIYA_DATA") == "" {
		t.Skip("requires original assets")
	}
	arc, err := archive.Open(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	checkedFiles := 0
	for _, file := range []string{"Effect/world_a1.eft", "Effect/world_b1.eft", "Effect/DA_target_recover.EFT"} {
		raw, err := arc.ReadFile(file)
		if err != nil {
			t.Logf("%s unavailable", file)
			continue
		}
		lib, err := effect.ParseEFT(raw)
		if err != nil {
			t.Fatal(err)
		}
		checked, particles := 0, 0
		for i := range lib.Effects {
			d := &lib.Effects[i]
			if !d.LinearSupported() {
				continue
			}
			bound := emitterReach(d, 1)
			x := &Effect{emitters: []effectEmitter{{definition: d}}, basis: render.Identity()}
			for tick := 0; tick < 360; tick++ {
				x.Update(1.0/60 + 1e-6)
				for _, particle := range x.emitters[0].particles {
					particles++
					position := particle.pos
					if d.Unk06 != 0 {
						position = position.Add(particle.velocity)
					}
					if distance := position.Len(); math.IsNaN(float64(distance)) || math.IsInf(float64(distance), 0) || distance > bound {
						t.Fatalf("%s emitter %d tick %d: particle %v exceeds bound %v", file, i, tick, distance, bound)
					}
				}
			}
			checked++
		}
		if checked == 0 || particles == 0 {
			t.Fatalf("%s: no simulated particles", file)
		}
		checkedFiles++
		t.Logf("%s: %d supported emitters, %d particle samples within bounds", file, checked, particles)
	}
	if checkedFiles == 0 {
		t.Fatal("no world effect libraries checked")
	}
}

func TestRealWorldEffectMeshBounds(t *testing.T) {
	if os.Getenv("SHAIYA_DATA") == "" {
		t.Skip("requires original assets")
	}
	arc, err := archive.Open(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	raw, err := arc.ReadFile("Effect/world_a1.eft")
	if err != nil {
		t.Fatal(err)
	}
	lib, err := effect.ParseEFT(raw)
	if err != nil {
		t.Fatal(err)
	}
	checked := 0
	for _, filename := range lib.Meshes {
		name := "Effect/3DE/" + path.Base(strings.ReplaceAll(filename, `\`, "/"))
		raw, err := arc.ReadFile(name)
		if err != nil {
			continue
		}
		mesh, err := effect.Parse3DE(raw)
		if err != nil {
			continue
		}
		maxRadius := float32(0)
		for _, v := range mesh.Vertices {
			maxRadius = max(maxRadius, render.V3(v.Position).Len())
		}
		for _, frame := range mesh.Frames {
			for _, v := range frame.Vertices {
				maxRadius = max(maxRadius, render.V3(v.Position).Len())
			}
		}
		geom := newEffectGeometry(mesh)
		for _, frame := range mesh.Frames {
			for _, delta := range []float32{-2, -1, 0, 1, 2} {
				key := min(max(float32(frame.Key)+delta, 0), float32(mesh.MaxKeyframe))
				for _, v := range geom.verticesAt(key) {
					if distance := v.Pos.Len(); distance > maxRadius*5+.001 {
						t.Fatalf("%s frame %v exceeds mesh bound: %v > %v", name, key, distance, maxRadius*5)
					}
				}
			}
		}
		checked++
	}
	if checked == 0 {
		t.Fatal("no original meshes checked")
	}
	t.Logf("checked interpolation bounds for %d original meshes", checked)
}

// Exercise every newly supported force emitter against the original EFTs.
func TestRealEffectForceBounds(t *testing.T) {
	if os.Getenv("SHAIYA_DATA") == "" {
		t.Skip("requires original assets")
	}
	a, err := archive.Open(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	count, samples := 0, 0
	a.Walk(func(file string, _ *archive.File) {
		if !strings.HasSuffix(strings.ToLower(file), ".eft") {
			return
		}
		raw, err := a.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		lib, err := effect.ParseEFT(raw)
		if err != nil {
			t.Fatal(err)
		}
		for i := range lib.Effects {
			d := &lib.Effects[i]
			if !d.LinearSupported() || d.Unk20 == 0 && d.Unk21 == 0 {
				continue
			}
			bound := emitterReach(d, 1)
			x := &Effect{emitters: []effectEmitter{{definition: d}}, basis: render.Identity()}
			for tick := 0; tick < 120; tick++ {
				x.Update(1.0/60 + 1e-6)
				for _, particle := range x.emitters[0].particles {
					pos := particle.pos
					if d.Unk06 != 0 {
						pos = pos.Add(particle.velocity)
					}
					distance := pos.Len()
					if math.IsNaN(float64(distance)) || math.IsInf(float64(distance), 0) || distance > bound {
						t.Fatalf("%s emitter %d tick %d: position %v exceeds bound %v", file, i, tick, pos, bound)
					}
					samples++
				}
			}
			count++
		}
	})
	if count == 0 || samples == 0 {
		t.Fatal("no force emitters or particles checked")
	}
	t.Logf("checked %d force emitters and %d samples", count, samples)
}

func TestRealStationaryZeroLifeBounds(t *testing.T) {
	if os.Getenv("SHAIYA_DATA") == "" {
		t.Skip("requires original assets")
	}
	a, err := archive.Open(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	count, samples := 0, 0
	a.Walk(func(file string, _ *archive.File) {
		if !strings.HasSuffix(strings.ToLower(file), ".eft") {
			return
		}
		raw, err := a.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		lib, err := effect.ParseEFT(raw)
		if err != nil {
			t.Fatal(err)
		}
		for i := range lib.Effects {
			d := &lib.Effects[i]
			p := d.Parameters()
			if p.LifeMin != 0 || p.LifeMax != 0 || !d.LinearSupported() || d.Spread1 != [3]float32{} || d.Spread2 != [3]float32{} {
				continue
			}
			bound := emitterReach(d, 1)
			if math.IsInf(float64(bound), 0) || math.IsNaN(float64(bound)) {
				t.Fatalf("%s[%d] nonfinite bound", file, i)
			}
			e := effectEmitter{definition: d}
			if d.UsesMesh() {
				e.geometry = &effectGeometry{source: &effect.Mesh3DE{}}
			}
			x := &Effect{emitters: []effectEmitter{e}, basis: render.Identity()}
			for tick := 0; tick < 120; tick++ {
				x.Update(1.0/60 + .000001)
				for _, v := range x.emitters[0].particles {
					position := v.pos
					if d.Unk06 != 0 {
						position = position.Add(v.velocity)
					}
					if position.Len() > bound {
						t.Fatalf("%s[%d] position outside bound", file, i)
					}
					samples++
				}
			}
			x.Stop()
			x.Update(1.0/60 + .000001)
			if !x.Done() {
				t.Fatalf("%s[%d] did not stop", file, i)
			}
			count++
		}
	})
	if count == 0 || samples == 0 {
		t.Fatal("no supported stationary zero-life particles")
	}
	t.Logf("checked %d emitters and %d positions", count, samples)
}

func TestRealMovingZeroLifeEffects(t *testing.T) {
	if os.Getenv("SHAIYA_DATA") == "" {
		t.Skip("requires original assets")
	}
	a, err := archive.Open(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	raw, err := a.ReadFile("Effect/ev_worldcup.eft")
	if err != nil {
		t.Fatal(err)
	}
	lib, err := effect.ParseEFT(raw)
	if err != nil {
		t.Fatal(err)
	}
	for _, index := range []int{72, 73, 74} {
		d := &lib.Effects[index]
		if !d.LinearSupported() || !d.UsesMesh() || !math.IsInf(float64(emitterReach(d, 1)), 1) {
			t.Fatalf("emitter %d not safely supported", index)
		}
		e := effectEmitter{definition: d, geometry: &effectGeometry{source: &effect.Mesh3DE{}}}
		x := &Effect{emitters: []effectEmitter{e}, basis: render.Identity()}
		x.Update(.2)
		if len(x.emitters[0].particles) != 1 {
			t.Fatalf("emitter %d did not spawn", index)
		}
		p := x.emitters[0].particles[0].pos
		x.Update(.2)
		if x.emitters[0].particles[0].pos == p {
			t.Fatalf("emitter %d did not move", index)
		}
		x.Stop()
		x.Update(.02)
		if !x.Done() {
			t.Fatalf("emitter %d did not stop", index)
		}
	}
}
