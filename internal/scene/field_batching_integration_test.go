//go:build integration

package scene

import (
	"math"
	"os"
	"runtime"
	"sort"
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/go-gl/gl/v4.5-core/gl"
)

// Compare the complete old draw traversal against visibility/instancing using
// actual map geometry, textures, lighting and fog, including camera turns.
func TestFieldBatchingPreservesVisiblePixelsGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires original assets and GL")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Title: "field batching", Width: 640, Height: 360, Hidden: true, Debug: true})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
	if err = render.Init(true); err != nil {
		t.Fatal(err)
	}
	arc, err := archive.Open(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	r, err := render.NewScene3D()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Delete()
	for _, tc := range []struct {
		stem string
		x, z float32
	}{{"19", 1703, 1632}, {"1", 583, 1767}} {
		w, err := loadStreamedField(NewAssets(arc), tc.stem)
		if err != nil {
			t.Fatal(err)
		}
		func() {
			defer w.Close()
			origin := render.V3{tc.x, w.Height(tc.x, tc.z) + 2, tc.z}
			for frame := 0; frame < 500; frame++ {
				w.UpdateVisible(origin, 900)
				if w.Ready() {
					break
				}
			}
			if !w.Ready() {
				t.Fatal("stream not ready")
			}
			// Reference traversal follows native passes and stable material priority,
			// independently of instancing and camera visibility.
			nativeOrder := append([]int(nil), w.stream.drawObjects...)
			sort.SliceStable(nativeOrder, func(i, j int) bool {
				a, b := w.stream.objects[nativeOrder[i]], w.stream.objects[nativeOrder[j]]
				if a.mani != b.mani {
					return a.mani
				}
				return a.file < b.file
			})
			r.Proj = render.PerspectiveFovLH(math.Pi/4, 640.0/360, .1, max(float32(400), w.FogEnd))
			r.FogColor, r.FogStart, r.FogEnd = w.FogColor, w.FogStart, w.FogEnd
			for turn := 0; turn < 8; turn++ {
				angle := float64(turn) * math.Pi / 4
				r.View = render.LookAtLH(origin, origin.Add(render.V3{float32(math.Sin(angle)), -.12, float32(math.Cos(angle))}), render.V3{0, 1, 0})
				draw := func(optimized, clipCulling bool) ([]byte, int, time.Duration) {
					gl.Viewport(0, 0, 640, 360)
					gl.DepthMask(true)
					gl.ClearColor(.13, .17, .2, 1)
					gl.ClearDepth(1)
					gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
					r.Begin()
					render.ResetDrawCalls()
					start := time.Now()
					if optimized {
						w.stream.drawWithClipCulling(r, clipCulling)
					} else {
						s := w.stream
						for _, key := range s.order {
							if c := s.cells[key]; c != nil {
								for _, p := range c.parts {
									drawTerrainCell(r, s.terrainBatches[p.batch], p.rangeInBatch)
								}
							}
						}
						for _, id := range nativeOrder {
							o := s.objects[id]
							m := s.assets.models[o.file]
							if m != nil {
								for _, p := range m.Parts {
									r.DrawModel(p.Mesh, p.Tex, o.world, render.NativeWorldAlphaRef(o.mani), false)
								}
							}
						}
						for _, asset := range s.vanis {
							if asset != nil {
								asset.worlds = asset.worlds[:0]
							}
						}
						for _, id := range nativeOrder {
							o := s.objects[id]
							if o.vani && s.vanis[o.file] != nil {
								asset := s.vanis[o.file]
								asset.worlds = append(asset.worlds, o.world)
							}
						}
						for _, asset := range s.vanis {
							if asset != nil {
								asset.draw(r, w.elapsed)
							}
						}
					}
					elapsed := time.Since(start)
					count := render.DrawCalls()
					pixels := make([]byte, 640*360*4)
					gl.ReadPixels(0, 0, 640, 360, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&pixels[0]))
					return pixels, count, elapsed
				}
				before, oldCalls, oldCPU := draw(false, false)
				after, newCalls, newCPU := draw(true, true)
				different, largest := 0, 0
				for i, v := range before {
					d := int(v) - int(after[i])
					if d < 0 {
						d = -d
					}
					if d != 0 {
						different++
					}
					largest = max(largest, d)
				}
				t.Logf("map=%s turn=%d draws=%d -> %d CPU submission=%s -> %s differing channels=%d max=%d", tc.stem, turn, oldCalls, newCalls, oldCPU, newCPU, different, largest)
				if different != 0 {
					t.Errorf("map %s turn %d changed %d channels (max %d)", tc.stem, turn, different, largest)
				}
				if newCalls >= oldCalls {
					t.Error("draw count did not improve")
				}
				if tc.stem == "1" && turn == 0 {
					r.ClipPlane = [4]float32{0, 1, 0, 0}
					beforeClip, beforeCalls, _ := draw(true, false)
					afterClip, afterCalls, _ := draw(true, true)
					for i := range beforeClip {
						if beforeClip[i] != afterClip[i] {
							t.Fatalf("clip culling changed field pixel channel %d", i)
						}
					}
					t.Logf("plane culling draw calls %d -> %d, pixels identical", beforeCalls, afterCalls)
					r.ClipPlane = [4]float32{}
				}
			}
		}()
	}
	if code := gl.GetError(); code != gl.NO_ERROR {
		t.Fatalf("GL error %#x", code)
	}
}
