//go:build integration

package scene

import (
	"bytes"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/go-gl/gl/v4.5-core/gl"
)

func TestNativeTerrainRealMapGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL and real field assets")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, e := platform.Open(platform.Config{Width: 960, Height: 600, Hidden: true})
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
	out := os.Getenv("SHAIYA_CAPTURE_DIR")
	if out == "" {
		out = t.TempDir()
	}
	if e = os.MkdirAll(out, 0755); e != nil {
		t.Fatal(e)
	}
	for _, tc := range []struct {
		mapID    string
		x, z     float32
		expected float32
	}{{"1", 1938, 1936, 149.265}, {"2", 966, 58, 95.13}, {"19", 500, 2, 112.835}} {
		w, e := loadStreamedField(NewAssets(arc), tc.mapID)
		if e != nil {
			t.Fatal(e)
		}
		func() {
			defer w.Close()
			if h := w.Height(tc.x, tc.z); math.Abs(float64(h-tc.expected)) > .001 {
				t.Fatal("wrong native height", tc.mapID, h)
			}
			// Keep original resources and the camera identical for the before/after.
			legacy := *w
			legacy.terrainFan = false
			legacy.heights = make([]float32, len(w.heights))
			for i, h := range w.WLD.Heights {
				legacy.heights[i] = (float32(h) - 10000) / 50
			}
			x0, z0 := max(0, int(tc.x/8)*4-16), max(0, int(tc.z/8)*4-16)
			x0, z0 = min(x0, w.side-33), min(z0, w.side-33)
			cells := (w.side - 1) / w.sectionsPerSide()
			sx, sz := x0/cells, z0/cells
			section := w.stream.loadMaterial(sx, sz)
			target := render.V3{tc.x, (w.Height(tc.x, tc.z) + legacy.Height(tc.x, tc.z)) * .5, tc.z}
			eye := target.Add(render.V3{-85, 65, -85})
			r.Proj = render.PerspectiveFovLH(math.Pi/4, 960.0/600, .1, 500)
			r.View = render.LookAtLH(eye, target, render.V3{0, 1, 0})
			r.FogStart, r.FogEnd = 450, 500
			var before []byte
			for i, source := range []*World{&legacy, w} {
				v, idx := outTerrain(source, x0, z0, 32, 32, sx, sz)
				if i == 1 {
					checkMeshHeights(t, w, v, idx)
				}
				m := render.NewMesh(v, idx)
				section.Mesh = m
				gl.Viewport(0, 0, 960, 600)
				gl.DepthMask(true)
				gl.ClearColor(.15, .2, .25, 1)
				gl.ClearDepth(1)
				gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
				r.Begin()
				drawSection(r, section)
				r.End()
				m.Delete()
				pixels := make([]byte, 960*600*4)
				gl.ReadPixels(0, 0, 960, 600, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&pixels[0]))
				suffix := "before"
				if i == 0 {
					before = pixels
				} else {
					suffix = "native"
					if bytes.Equal(before, pixels) {
						t.Fatal("real terrain correction did not change the image")
					}
				}
				if e = render.SavePNG(filepath.Join(out, "terrain-"+tc.mapID+"-"+suffix+".png"), 960, 600); e != nil {
					t.Fatal(e)
				}
				if code := gl.GetError(); code != gl.NO_ERROR {
					t.Fatalf("terrain GL error %#x", code)
				}
				t.Logf("map%s %s vertices=%d triangles=%d", tc.mapID, suffix, len(v), len(idx)/3)
			}
		}()
	}
}
