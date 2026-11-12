//go:build integration

package app

import (
	"os"
	"runtime"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
)

func TestSwimmingMotionRealMap(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL and original assets")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Title: "swimming motion", Width: 64, Height: 64, Hidden: true})
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
	w, err := scene.LoadMap(scene.NewAssets(arc), 1)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	var deep, dry render.V3
	found := false
	// Search a real shore, not a synthetic height map.
	for x := float32(10); x < float32(w.WLD.MapSize)-10 && !found; x += 5 {
		for z := float32(10); z < float32(w.WLD.MapSize)-10; z += 5 {
			if _, ok := w.WaterFloor(x, z); !ok {
				continue
			}
			for _, d := range []render.V3{{5, 0, 0}, {-5, 0, 0}, {0, 0, 5}, {0, 0, -5}} {
				nx, nz := x+d[0], z+d[2]
				if w.Height(nx, nz) > 0 {
					deep = render.V3{x, scene.SwimEntryY, z}
					dry = render.V3{nx, w.Height(nx, nz), nz}
					found = true
					break
				}
			}
			if found {
				break
			}
		}
	}
	if !found {
		t.Fatal("no shoreline found")
	}
	if _, ok := w.UnderwaterFogAt(render.V3{deep[0], -.1, deep[2]}); !ok {
		t.Fatal("camera beneath the surface should receive underwater fog")
	}
	if _, ok := w.UnderwaterFogAt(render.V3{deep[0], .1, deep[2]}); ok {
		t.Fatal("camera above the surface should retain the map fog")
	}
	if _, ok := w.UnderwaterFogAt(render.V3{dry[0], -.1, dry[2]}); ok {
		t.Fatal("underground camera on dry land should not receive water fog")
	}
	v := &worldView{world: w, pos: dry, eye: dry.Add(render.V3{0, 2, -5})}
	offset := v.eye.Sub(v.pos)
	v.step(deep.Sub(dry))
	if v.pos[1] != dry[1] {
		t.Fatalf("walking off a shore teleported vertically: %v", v.pos)
	}
	for i := 0; i < 100 && v.pos[1] > scene.SwimEntryY; i++ {
		v.settleIntoWater(.1)
	}
	if v.pos[1] != scene.SwimEntryY || !w.SwimmingAt(v.pos[0], v.pos[1], v.pos[2]) {
		t.Fatalf("water entry: %v", v.pos)
	}
	if !v.floatAtSurface(.2) || v.pos[1] != scene.SwimSurfaceY {
		t.Fatalf("surface flotation: %v", v.pos)
	}
	if v.floatAtSurface(.2) {
		t.Fatal("surface continues to rise")
	}
	v.step(render.V3{0, -.5, 0})
	below := v.pos[1]
	if v.floatAtSurface(.2) || v.pos[1] != below {
		t.Fatal("deep position should be preserved")
	}
	if !v.riseInWater(1) || v.pos[1] != scene.SwimSurfaceY || v.riseInWater(1) {
		t.Fatal("manual ascent should stop at the surface", v.pos)
	}
	v.step(dry.Sub(v.pos))
	if v.pos[1] != dry[1] || w.SwimmingAt(v.pos[0], v.pos[1], v.pos[2]) {
		t.Fatalf("shore exit: %v", v.pos)
	}
	if v.eye.Sub(v.pos).Sub(offset).Len() > .001 {
		t.Fatal("camera offset changed")
	}
}
