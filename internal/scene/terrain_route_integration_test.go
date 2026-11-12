//go:build integration

package scene

import (
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

// Measures streaming work on the GL thread while crossing and returning over
// real cell boundaries. Times exclude rendering and driver completion waits.
func TestTerrainStreamingRouteGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires original assets and GL")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Title: "terrain route", Width: 64, Height: 64, Hidden: true, Debug: true})
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
	baseline := render.GPUBytes()
	w, err := loadStreamedField(NewAssets(arc), "1")
	if err != nil {
		t.Fatal(err)
	}
	start := render.V3{583, 80, 1767}
	for n := 0; n < 240 && !w.Ready(); n++ {
		w.UpdateVisible(start, 500)
	}
	if !w.Ready() {
		t.Fatal("initial residency did not converge")
	}
	initial := render.GPUBytes()
	peak := initial
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	var longest, total time.Duration
	updates := 0
	for step := 0; step <= 64; step++ {
		x := step
		if x > 32 {
			x = 64 - x
		}
		pos := start.Add(render.V3{float32(x) * 20, 0, 0})
		for n := 0; n < 240; n++ {
			begin := time.Now()
			w.UpdateVisible(pos, 500)
			elapsed := time.Since(begin)
			longest = max(longest, elapsed)
			total += elapsed
			updates++
			peak = max(peak, render.GPUBytes())
			if w.Ready() {
				break
			}
			if n == 239 {
				t.Fatal("residency did not converge", step)
			}
		}
	}
	runtime.ReadMemStats(&after)
	returned := render.GPUBytes()
	if returned != initial {
		t.Fatalf("GPU residency changed after returning: %d -> %d", initial, returned)
	}
	w.Close()
	if left := render.GPUBytes(); left != baseline {
		t.Fatalf("GPU storage not released: baseline=%d remaining=%d", baseline, left)
	}
	t.Logf("route updates=%d mean=%s max=%s allocated=%.1f MiB, GPU initial=%.1f peak=%.1f MiB", updates, total/time.Duration(updates), longest, float64(after.TotalAlloc-before.TotalAlloc)/(1<<20), float64(initial-baseline)/(1<<20), float64(peak-baseline)/(1<<20))
}
