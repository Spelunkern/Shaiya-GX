//go:build integration

package scene

import (
	"bytes"
	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/go-gl/gl/v4.5-core/gl"
	"math"
	"os"
	"runtime"
	"testing"
	"time"
)

func TestTerrainOcclusionKeepsPixelsAndReducesSubmissionGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("needs assets and GPU")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, e := platform.Open(platform.Config{Width: 640, Height: 360, Hidden: true})
	if e != nil {
		t.Fatal(e)
	}
	defer win.Close()
	if e = render.Init(false); e != nil {
		t.Fatal(e)
	}
	arc, e := archive.Open(os.Getenv("SHAIYA_DATA"))
	if e != nil {
		t.Fatal(e)
	}
	defer arc.Close()
	r, e := render.NewScene3D()
	if e != nil {
		t.Fatal(e)
	}
	defer r.Delete()
	totalCulled, saved := 0, 0
	for _, tc := range []struct {
		mapID string
		x, z  float32
	}{{"1", 583, 1767}, {"19", 1703, 1632}, {"1", 950, 950}} {
		w, e := loadStreamedField(NewAssets(arc), tc.mapID)
		if e != nil {
			t.Fatal(e)
		}
		func() {
			defer w.Close()
			eye := render.V3{tc.x, w.Height(tc.x, tc.z) + 8, tc.z}
			for i := 0; i < 500; i++ {
				w.UpdateVisible(eye, 900)
				if w.Ready() {
					break
				}
			}
			if !w.Ready() {
				t.Fatal("stream not ready")
			}
			r.Proj = render.PerspectiveFovLH(math.Pi/4, 640.0/360, .1, 500)
			r.FogColor, r.FogStart, r.FogEnd = w.FogColor, w.FogStart, w.FogEnd
			for turn := 0; turn < 8; turn++ {
				a := float64(turn) * math.Pi / 4
				r.View = render.LookAtLH(eye, eye.Add(render.V3{float32(math.Sin(a)), -.1, float32(math.Cos(a))}), render.V3{0, 1, 0})
				draw := func(occlude bool) ([]byte, int, time.Duration) {
					r.Occlusion = nil
					start := time.Now()
					if occlude {
						w.PrepareTerrainOcclusion(r, eye)
					}
					gl.Viewport(0, 0, 640, 360)
					gl.DepthMask(true)
					gl.ClearColor(.13, .17, .2, 1)
					gl.ClearDepth(1)
					gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
					r.Begin()
					render.ResetDrawCalls()
					w.Draw(r)
					r.End()
					elapsed := time.Since(start)
					p := make([]byte, 640*360*4)
					gl.ReadPixels(0, 0, 640, 360, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&p[0]))
					return p, render.DrawCalls(), elapsed
				}
				before, oldCalls, oldTime := draw(false)
				after, newCalls, newTime := draw(true)
				culled := 0
				if r.Occlusion != nil {
					culled = r.Occlusion.Culled
				}
				totalCulled += culled
				saved += oldCalls - newCalls
				t.Logf("map=%s turn=%d culls=%d draws=%d->%d CPU=%s->%s", tc.mapID, turn, culled, oldCalls, newCalls, oldTime, newTime)
				if !bytes.Equal(before, after) {
					n := 0
					for i := range before {
						if before[i] != after[i] {
							n++
						}
					}
					t.Errorf("visible pixels changed: %d channels", n)
				}
				if newCalls > oldCalls {
					t.Error("occlusion increased draw submissions")
				}
			}
		}()
	}
	if totalCulled == 0 || saved <= 0 {
		t.Fatalf("no measured culling benefit: culls=%d draws saved=%d", totalCulled, saved)
	}
}
