//go:build integration

package scene

import (
	"fmt"
	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/data/world"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/go-gl/gl/v4.5-core/gl"
	"math"
	"os"
	"path"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestDungeonVisibilityPreservesPixelsGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("GL and real dungeon assets")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Width: 640, Height: 360, Hidden: true})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
	if err = render.Init(false); err != nil {
		t.Fatal(err)
	}
	arc, err := archive.OpenDirectory(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	r, err := render.NewScene3D()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Delete()
	saved := 0
	for _, id := range []string{"3", "4", "5", "6", "7", "8", "9"} {
		raw, err := arc.ReadFile("world/" + id + ".wld")
		if err != nil {
			continue
		}
		wd, err := world.ParseWLD(raw)
		if err != nil {
			t.Fatal(err)
		}
		if !wd.Dungeon {
			continue
		}
		a := NewAssets(arc)
		w, err := LoadDungeon(a, "world/"+id+".wld")
		if err != nil {
			t.Fatal(err)
		}
		if w.FogStart != 0 || w.FogEnd != 0 {
			t.Fatal("dungeon enables field fog")
		}
		raw, err = arc.ReadFile("world/dungeon/" + wd.Layout)
		if err != nil {
			t.Fatal(err)
		}
		dg, err := world.ParseDG(raw)
		if err != nil {
			t.Fatal(err)
		}
		var legacy []dgPart
		var eyes []render.V3
		dg.Root.Walk(func(n *world.DGNode) {
			for _, g := range n.Groups {
				var tex *render.Texture
				if g.Texture >= 0 && int(g.Texture) < len(dg.Textures) {
					tex = a.FindTexture(dg.Textures[g.Texture])
				}
				for _, m := range g.Meshes {
					vertices := make([]render.Vertex, len(m.Vertices))
					for i, v := range m.Vertices {
						vertices[i] = render.Vertex{Pos: render.V3(v.Pos), Normal: render.V3(v.Normal), UV: v.UV, UV2: v.UV2}
					}
					var indices []uint32
					for _, f := range m.Faces {
						if int(f[0]) >= len(vertices) || int(f[1]) >= len(vertices) || int(f[2]) >= len(vertices) {
							continue
						}
						indices = append(indices, uint32(f[0]), uint32(f[1]), uint32(f[2]))
						p, q, z := vertices[f[0]], vertices[f[1]], vertices[f[2]]
						if p.Normal[1] > .7 && q.Normal[1] > .7 && z.Normal[1] > .7 {
							eyes = append(eyes, p.Pos.Add(q.Pos).Add(z.Pos).Scale(1.0/3).Add(render.V3{0, 2, 0}))
						}
					}
					if len(indices) == 0 {
						continue
					}
					stem := strings.TrimSuffix(wd.Layout, path.Ext(wd.Layout))
					var lm *render.Texture
					if m.Lightmap >= 0 && m.Lightmap < dg.Lightmaps {
						file := fmt.Sprintf("world/dungeon/%s/%s_L%d.dds", stem, stem, m.Lightmap)
						if _, ok := arc.Lookup(file); ok {
							lm = a.worldTexture(file)
						}
					}
					legacy = append(legacy, dgPart{mesh: render.NewMesh(vertices, indices), tex: tex, lightmap: lm})
				}
			}
		})
		t.Logf("map%s DG raw=%d batches=%d objects=%d rotors=%d", id, len(legacy), len(w.dg), len(w.Objects), len(w.rotors))
		if len(eyes) == 0 {
			t.Fatal("no dungeon floor samples", id)
		}
		r.Proj = render.PerspectiveFovLH(math.Pi/4, 640.0/360, .1, max(float32(400), wd.FogEnd))
		r.FogStart, r.FogEnd = 0, 0
		var eye render.V3
		draw := func(mode int) ([]byte, int, time.Duration) {
			r.Occlusion = nil
			start := time.Now()
			if mode == 2 {
				w.PrepareTerrainOcclusion(r, eye)
			}
			gl.Viewport(0, 0, 640, 360)
			gl.DepthMask(true)
			gl.ClearDepth(1)
			gl.ClearColor(.03, .03, .03, 1)
			gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
			r.Begin()
			render.ResetDrawCalls()
			if mode == 0 {
				for _, p := range legacy {
					if p.lightmap != nil {
						r.DrawLightmapped(p.mesh, p.tex, p.lightmap)
					} else {
						r.DrawModel(p.mesh, p.tex, render.Identity(), render.NativeWorldAlphaRef(true), false)
					}
				}
				for _, rotor := range w.rotors {
					matrix := rotor.transform(w.elapsed)
					for _, p := range rotor.model.Parts {
						r.DrawModel(p.Mesh, p.Tex, matrix, render.NativeWorldAlphaRef(true), false)
					}
				}
				for _, mani := range []bool{true, false} {
					for _, object := range w.Objects {
						if object.Mani != mani {
							continue
						}
						for _, p := range object.Model.Parts {
							r.DrawModel(p.Mesh, p.Tex, object.World, render.NativeWorldAlphaRef(mani), false)
						}
					}
				}
				for _, v := range w.vanis {
					v.draw(r, w.elapsed)
				}
			} else {
				w.Draw(r)
			}
			r.End()
			elapsed := time.Since(start)
			pixels := make([]byte, 640*360*4)
			gl.ReadPixels(0, 0, 640, 360, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(pixels))
			return pixels, render.DrawCalls(), elapsed
		}
		poses := 3
		if id == "3" {
			poses = 4
		}
		for pose := 0; pose < poses; pose++ {
			if pose == 3 {
				eye = render.V3{199.53047, 21.09025, 400.599}
			} else {
				eye = eyes[(len(eyes)-1)*pose/3]
			}
			aspect := []float32{16.0 / 9, 4.0 / 3, 21.0 / 9, 16.0 / 9}[pose]
			r.Proj = render.PerspectiveFovLH(math.Pi/4, aspect, .1, max(float32(400), wd.FogEnd))
			for turn := 0; turn < 8; turn++ {
				w.Update(.113)
				angle := float64(turn) * math.Pi / 4
				r.View = render.LookAtLH(eye, eye.Add(render.V3{float32(math.Sin(angle)), -.08, float32(math.Cos(angle))}), render.V3{0, 1, 0})
				// The builder's eye is used only to select nearby occluder candidates.
				w.occlusion = nil
				before, oldCalls, oldTime := draw(0)
				batch, batchCalls, _ := draw(1)
				after, newCalls, newTime := draw(2)
				mask := w.occlusion
				changed, maxDelta := 0, 0
				for i := range before {
					d := int(before[i]) - int(after[i])
					if d < 0 {
						d = -d
					}
					maxDelta = max(maxDelta, d)
					if d > 2 {
						changed++
					}
				}
				batchChanged := 0
				for i := range before {
					d := int(before[i]) - int(batch[i])
					if d > 2 || d < -2 {
						batchChanged++
					}
				}
				t.Logf("map%s pose%d turn%d raw=%d batch/frustum=%d occlusion=%d hidden=%d cpu=%s/%s pixels=%d batchPixels=%d maxDelta=%d", id, pose, turn, oldCalls, batchCalls, newCalls, mask.Culled, oldTime, newTime, changed, batchChanged, maxDelta)
				if changed > 0 {
					t.Fatalf("dungeon visibility changed %d channels (batch=%d)", changed, batchChanged)
				}
				saved += oldCalls - newCalls
			}
		}
		for _, p := range legacy {
			p.mesh.Delete()
		}
		w.Close()
		for _, texture := range a.textures {
			if texture != nil {
				texture.Delete()
			}
		}
	}
	if saved == 0 {
		t.Fatal("no dungeon submission savings")
	}
}
