//go:build integration

package scene

import (
	"bytes"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/go-gl/gl/v4.5-core/gl"
)

func TestReducedDetailsLowersRealSceneSubmissionGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL and field assets")
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
	arc, e := archive.OpenDirectory(os.Getenv("SHAIYA_DATA"))
	if e != nil {
		t.Fatal(e)
	}
	defer arc.Close()
	r, e := render.NewScene3D()
	if e != nil {
		t.Fatal(e)
	}
	defer r.Delete()
	r.Proj = render.PerspectiveFovLH(math.Pi/4, 640.0/360, .1, 500)
	r.ViewportW, r.ViewportH = 640, 360
	r.StaticObjectLOD = true
	var tris, calls [3]int
	var gpu [3]uint64
	views := 0
	out := os.Getenv("SHAIYA_CAPTURE_DIR")
	if out != "" {
		if e = os.MkdirAll(out, 0755); e != nil {
			t.Fatal(e)
		}
	}
	var query uint32
	gl.GenQueries(1, &query)
	defer gl.DeleteQueries(1, &query)
	for _, tc := range []struct {
		mapID string
		x, z  float32
	}{{"1", 583, 1767}, {"1", 950, 950}, {"19", 1703, 1632}} {
		w, e := loadStreamedField(NewAssets(arc), tc.mapID)
		if e != nil {
			t.Fatal(e)
		}
		func() {
			defer w.Close()
			eye := render.V3{tc.x, w.Height(tc.x, tc.z) + 8, tc.z}
			for i := 0; i < 500 && !w.Ready(); i++ {
				w.UpdateVisible(eye, 650)
			}
			if !w.Ready() {
				t.Fatal("stream not ready")
			}
			r.FogColor, r.FogStart, r.FogEnd = w.FogColor, w.FogStart, w.FogEnd
			for turn := 0; turn < 4; turn++ {
				angle := float64(turn) * math.Pi / 2
				r.View = render.LookAtLH(eye, eye.Add(render.V3{float32(math.Sin(angle)), -.1, float32(math.Cos(angle))}), render.V3{0, 1, 0})
				draw := func(bias int) ([]byte, int, int, int, uint64) {
					r.DetailBias = float32(bias)
					// Warm this preset before timing, keeping uploads and first shader use out.
					gl.Viewport(0, 0, 640, 360)
					gl.DepthMask(true)
					gl.ClearColor(.13, .17, .2, 1)
					gl.ClearDepth(1)
					gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
					r.Begin()
					w.Draw(r)
					r.End()
					gl.Finish()
					gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
					render.ResetDrawCalls()
					start := time.Now()
					gl.BeginQuery(gl.TIME_ELAPSED, query)
					r.Begin()
					w.Draw(r)
					r.End()
					gl.EndQuery(gl.TIME_ELAPSED)
					cpu := time.Since(start)
					var ns uint64
					gl.GetQueryObjectui64v(query, gl.QUERY_RESULT, &ns)
					p := make([]byte, 640*360*4)
					gl.ReadPixels(0, 0, 640, 360, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&p[0]))
					t.Logf("map%s turn%d Details%d tris=%d draws=%d culled=%d CPU=%s GPU=%s", tc.mapID, turn, 2-bias, w.stream.submittedTriangles, render.DrawCalls(), w.stream.detailCulled, cpu, time.Duration(ns))
					return p, w.stream.submittedTriangles, render.DrawCalls(), w.stream.detailCulled, ns
				}
				var original []byte
				for bias := 0; bias < 3; bias++ {
					pixels, n, c, culled, ns := draw(bias)
					tris[bias] += n
					calls[bias] += c
					gpu[bias] += ns
					if bias == 0 {
						original = pixels
						if culled != 0 {
							t.Fatal("default detail culled decoration")
						}
					}
					if out != "" && turn == 0 && tc.x == 583 {
						if e = render.SavePNG(filepath.Join(out, []string{"details-high.png", "details-medium.png", "details-low.png"}[bias]), 640, 360); e != nil {
							t.Fatal(e)
						}
					}
				}
				restored, _, _, _, _ := draw(0)
				if !bytes.Equal(original, restored) {
					t.Fatal("restoring default Details changed pixels")
				}
				if code := gl.GetError(); code != gl.NO_ERROR {
					t.Fatalf("details GL error %#x", code)
				}
				views++
			}
		}()
	}
	t.Logf("TOTAL %d views: tris=%v draws=%v GPU=%v", views, tris, calls, gpu)
	if tris[1] >= tris[0] || tris[2] >= tris[1] || calls[2] >= calls[0] {
		t.Fatal("lower Details did not reduce submitted work", tris, calls)
	}
}
