//go:build integration

package scene

import (
	"math"
	"os"
	"runtime"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

func TestWaterFieldDrawProfileGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires original assets and GL")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Title: "water field profile", Width: 256, Height: 256, Hidden: true, Debug: true})
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
	w, err := loadStreamedField(NewAssets(arc), "1")
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	r, err := render.NewScene3D()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Delete()
	eye := render.V3{600, 2, 620}
	for i := 0; i < 240 && !w.Ready(); i++ {
		w.UpdateVisible(eye, 800)
	}
	if !w.Ready() {
		t.Fatal("stream not ready")
	}
	r.View = render.LookAtLH(eye, eye.Add(render.V3{0, 0, 5}), render.V3{0, 1, 0})
	mirror := render.Identity()
	mirror[5] = -1
	r.View = mirror.Mul(r.View)
	r.Proj = render.PerspectiveFovLH(math.Pi/4, 1, .1, max(400, w.FogEnd))
	r.ClipPlane = [4]float32{0, 1, 0, 0}
	r.Begin()
	frustum := render.ViewFrustum(r.View, r.Proj)
	render.ResetDrawCalls()
	for _, key := range w.stream.order {
		if c := w.stream.cells[key]; c != nil {
			for _, p := range c.parts {
				if render.ClipBoxVisible(r.ClipPlane, p.minimum, p.maximum) && frustum.SphereVisible(p.center, p.radius) {
					drawTerrainCell(r, w.stream.terrainBatches[p.batch], p.rangeInBatch)
				}
			}
		}
	}
	terrain := render.DrawCalls()
	render.ResetDrawCalls()
	w.stream.drawWithClipCulling(r, false)
	before := render.DrawCalls()
	render.ResetDrawCalls()
	w.stream.drawWithClipCulling(r, true)
	after := render.DrawCalls()
	groupedTerrain := 0
	for _, batch := range w.stream.visibleTerrain {
		for _, layer := range batch.material.Layers {
			if layer.Mask != nil && layer.Tile != nil {
				groupedTerrain++
			}
		}
	}
	t.Logf("reflected field: terrain %d -> %d, objects=%d, clipping %d -> %d", terrain, groupedTerrain, after-groupedTerrain, before, after)
	r.End()
}
