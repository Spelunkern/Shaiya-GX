//go:build integration

package scene

import (
	"math"
	"os"
	"reflect"
	"runtime"
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

func TestObjectShadowCostInRealFieldGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL and original assets")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Title: "shadow cost", Width: 128, Height: 128, Hidden: true})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
	if err := render.Init(false); err != nil {
		t.Fatal(err)
	}
	arc, err := archive.Open(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	w, err := loadStreamedField(NewAssets(arc), "1")
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	s, err := render.NewScene3D()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Delete()
	shadow, err := render.NewShadowMap()
	if err != nil {
		t.Fatal(err)
	}
	defer shadow.Delete()
	s.Proj = render.PerspectiveFovLH(math.Pi/4, 1, .1, 600)
	for _, p := range [][2]float32{{583, 1767}, {1024, 1024}, {1800, 300}} {
		pos := render.V3{p[0], w.Height(p[0], p[1]), p[1]}
		baselineV, baselineI := w.shadowReceiver(pos[0]-3, pos[2]-3, pos[0]+3, pos[2]+3, pos[1])
		for _, vertex := range baselineV {
			if want := w.Height(vertex.Pos[0], vertex.Pos[2]); vertex.Pos[1] != want {
				t.Fatalf("terrain receiver height at %v: got %v, want %v", vertex.Pos, vertex.Pos[1], want)
			}
		}
		reusedV, reusedI := w.shadowReceiverInto(pos[0]-3, pos[2]-3, pos[0]+3, pos[2]+3, pos[1], baselineV, baselineI, nil)
		freshV, freshI := w.shadowReceiver(pos[0]-3, pos[2]-3, pos[0]+3, pos[2]+3, pos[1])
		if !reflect.DeepEqual(reusedV, freshV) || !reflect.DeepEqual(reusedI, freshI) {
			t.Fatal("reused terrain receiver changed geometry")
		}
		freshAllocs := testing.AllocsPerRun(100, func() { w.shadowReceiver(pos[0]-3, pos[2]-3, pos[0]+3, pos[2]+3, pos[1]) })
		reusedAllocs := testing.AllocsPerRun(100, func() { w.shadowReceiverInto(pos[0]-3, pos[2]-3, pos[0]+3, pos[2]+3, pos[1], reusedV, reusedI, nil) })
		t.Logf("terrain receiver allocations: fresh %.1f, reused %.1f", freshAllocs, reusedAllocs)
		if reusedAllocs >= freshAllocs {
			t.Fatal("terrain receiver buffers were not reused")
		}
		for i := 0; i < 500; i++ {
			w.UpdateVisible(pos, 500)
			if w.Ready() {
				break
			}
		}
		if !w.Ready() {
			t.Fatal("stream did not converge")
		}
		s.View = render.LookAtLH(pos.Add(render.V3{0, 8, -12}), pos, render.V3{0, 1, 0})
		s.Begin()
		render.ResetDrawCalls()
		start := time.Now()
		const iterations = 50
		for i := 0; i < iterations; i++ {
			w.projectObjectShadows(shadow, s, render.Identity(), pos[0]-3, pos[2]-3, pos[0]+3, pos[2]+3, pos[1])
		}
		elapsed := time.Since(start)
		t.Logf("map 1 %.0f/%.0f: %d nearby placements resident, %d object shadow calls / %d queries, CPU submission %s", p[0], p[1], len(w.stream.drawObjects), render.DrawCalls(), iterations, elapsed)
		allocs := testing.AllocsPerRun(50, func() {
			w.projectObjectShadows(shadow, s, render.Identity(), pos[0]-3, pos[2]-3, pos[0]+3, pos[2]+3, pos[1])
		})
		t.Logf("object receiver allocations per query: %.1f", allocs)
	}
}
