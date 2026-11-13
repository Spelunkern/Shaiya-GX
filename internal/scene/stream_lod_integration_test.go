//go:build integration

package scene

import (
	"bytes"
	"fmt"
	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/go-gl/gl/v4.5-core/gl"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"testing"
	"time"
)

func TestAsyncSectorsAndDistantLODGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("needs GL/assets")
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
	r.Proj = render.PerspectiveFovLH(math.Pi/4, 640.0/360, .1, 500)
	r.ViewportW, r.ViewportH = 640, 360
	var durations []time.Duration
	wait := func(w *World, eye render.V3) {
		t.Helper()
		deadline := time.Now().Add(20 * time.Second)
		for {
			start := time.Now()
			w.UpdateVisible(eye, 600)
			if w.stream.loader != nil {
				e := time.Since(start)
				durations = append(durations, e)
				if e > 8*time.Millisecond {
					t.Logf("slow stream prepare=%s terrain=%s trim=%s", w.stream.lastPrepare, w.stream.lastTerrain, w.stream.lastTrim)
				}
			}
			if w.Ready() {
				return
			}
			if time.Now().After(deadline) {
				t.Fatalf("stream stalled pending=%d outstanding=%d", len(w.stream.pending), w.stream.loader.outstanding)
			}
			time.Sleep(time.Millisecond)
		}
	}
	trianglesSaved, lodCount, skipped := 0, 0, 0
	for _, tc := range []struct {
		mapID string
		x, z  float32
	}{{"1", 583, 1767}, {"19", 1703, 1632}, {"1", 950, 950}, {"2", 1024, 1024}} {
		full, err := loadStreamedField(NewAssets(arc), tc.mapID)
		if err != nil {
			t.Fatal(err)
		}
		async, err := loadStreamedField(NewAssets(arc), tc.mapID)
		if err != nil {
			full.Close()
			t.Fatal(err)
		}
		func() {
			defer full.Close()
			defer async.Close()
			async.EnableAsyncStreaming()
			eye := render.V3{tc.x, full.Height(tc.x, tc.z) + 5, tc.z}
			wait(full, eye)
			wait(async, eye)
			t.Logf("slowest upload %s %s", async.stream.loader.slowestFile, async.stream.loader.slowestUpload)
			variants := 0
			for _, m := range async.stream.assets.models {
				if m != nil && m.lodAvailable != [2]bool{} {
					variants++
				}
			}
			t.Logf("map=%s resident=%d sectors=%d models with LOD=%d", tc.mapID, len(async.stream.drawEntries), len(async.stream.drawSectors), variants)
			r.FogColor, r.FogStart, r.FogEnd = full.FogColor, full.FogStart, full.FogEnd
			for turn := 0; turn < 8; turn++ {
				angle := float64(turn) * math.Pi / 4
				r.View = render.LookAtLH(eye, eye.Add(render.V3{float32(math.Sin(angle)), -.1, float32(math.Cos(angle))}), render.V3{0, 1, 0})
				draw := func(w *World, lod, sector bool) ([]byte, int, int, int) {
					r.StaticObjectLOD = lod
					w.stream.disableSectors = !sector
					gl.Viewport(0, 0, 640, 360)
					gl.DepthMask(true)
					gl.ClearColor(.13, .17, .2, 1)
					gl.ClearDepth(1)
					gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
					r.Begin()
					render.ResetDrawCalls()
					w.Draw(r)
					r.End()
					p := make([]byte, 640*360*4)
					gl.ReadPixels(0, 0, 640, 360, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&p[0]))
					return p, w.stream.submittedTriangles, w.stream.objectTests, render.DrawCalls()
				}
				before, _, _, _ := draw(full, false, false)
				exact, tri0, tests, calls0 := draw(async, false, true)
				dir := os.Getenv("SHAIYA_TEST_ARTIFACTS")
				if turn == 4 && dir != "" {
					os.MkdirAll(dir, 0755)
					if err := render.SavePNG(filepath.Join(dir, fmt.Sprintf("map%s-%.0f-lod-off.png", tc.mapID, tc.x)), 640, 360); err != nil {
						t.Fatal(err)
					}
				}
				if !bytes.Equal(before, exact) {
					t.Fatal("async/sector changed full-detail pixels", tc.mapID, turn)
				}
				skipped += len(async.stream.drawEntries) - tests
				reduced, tri1, _, calls1 := draw(async, true, true)
				if turn == 4 && dir != "" {
					if err := render.SavePNG(filepath.Join(dir, fmt.Sprintf("map%s-%.0f-lod-on.png", tc.mapID, tc.x)), 640, 360); err != nil {
						t.Fatal(err)
					}
				}
				changed, totalError := 0, 0
				for i := 0; i < len(exact); i += 4 {
					different := false
					for c := 0; c < 3; c++ {
						d := int(exact[i+c]) - int(reduced[i+c])
						if d < 0 {
							d = -d
						}
						totalError += d
						different = different || d != 0
					}
					if different {
						changed++
					}
				}
				fraction := float64(changed) / (640 * 360)
				t.Logf("map=%s turn=%d tests=%d/%d LOD=%d tris=%d->%d draws=%d->%d changed=%.4f%% meanRGBerr=%.5f", tc.mapID, turn, tests, len(async.stream.drawEntries), async.stream.lodInstances, tri0, tri1, calls0, calls1, fraction*100, float64(totalError)/(640*360*3))
				if fraction > .005 {
					t.Errorf("LOD visible footprint too large %.3f%%", fraction*100)
				}
				if tri1 > tri0 {
					t.Error("LOD increased triangle submission")
				}
				trianglesSaved += tri0 - tri1
				lodCount += async.stream.lodInstances
			}
			// Cancellation/generation changes while decoding; then return to settled residency.
			for _, offset := range []float32{64, 128, 192, 128, 0} {
				wait(async, eye.Add(render.V3{offset, 0, 0}))
			}
		}()
	}
	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
	t.Logf("updates=%d p95GLupdate=%s max=%s savedTriangles=%d LODinstances=%d skippedFineTests=%d", len(durations), durations[len(durations)*95/100], durations[len(durations)-1], trianglesSaved, lodCount, skipped)
	if skipped == 0 {
		t.Fatal("sector culling had no measured benefit")
	}
	if trianglesSaved == 0 {
		t.Log("LOD experiment: no triangle reduction in sampled views")
	}
}
