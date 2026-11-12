//go:build integration

package scene

import (
	"os"
	"runtime"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/go-gl/gl/v4.5-core/gl"
)

func TestFieldStreamingResidencyGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires local Linux GL context and original assets")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Title: "stream test", Width: 64, Height: 64, Hidden: true, Debug: true})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
	if err := render.Init(true); err != nil {
		t.Fatal(err)
	}
	arc, err := archive.Open(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	parent := NewAssets(arc)
	w, err := loadStreamedField(parent, "1")
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	var firstMesh *render.Mesh
	var models, textures int
	for i, pos := range []render.V3{{583, 80, 1767}, {1024, 60, 1024}, {1800, 60, 300}, {583, 80, 1767}} {
		for frame := 0; frame < 240; frame++ {
			w.UpdateVisible(pos, 500)
			if w.Ready() {
				break
			}
		}
		if !w.Ready() {
			t.Fatal("stream did not converge")
		}
		s := w.stream
		if len(s.cells) != len(s.order) || len(s.cells) >= 1024 {
			t.Fatal("residency includes entire map", len(s.cells))
		}
		if i == 0 {
			models, textures = len(s.assets.models), len(s.assets.textures)
			for _, m := range s.assets.models {
				if m == nil {
					continue
				}
				for _, p := range m.Parts {
					if p.Tex == nil {
						continue
					}
					var wrap int32
					gl.GetTextureParameteriv(p.Tex.ID, gl.TEXTURE_WRAP_S, &wrap)
					if wrap != gl.REPEAT {
						t.Fatal("static model UVs clamp instead of repeating the original texture")
					}
				}
			}
			for _, batch := range s.terrainBatches {
				firstMesh = batch.mesh
				break
			}
		}
		if i == 2 && firstMesh != nil && firstMesh.Count != 0 {
			t.Fatal("old cell GPU mesh was not deleted")
		}
		if i == 3 && (models != len(s.assets.models) || textures != len(s.assets.textures)) {
			t.Fatal("cache grew after returning to the same area", models, textures, len(s.assets.models), len(s.assets.textures))
		}
		t.Logf("position %.0f/%.0f: cells=%d models=%d textures=%d", pos[0], pos[2], len(s.cells), len(s.assets.models), len(s.assets.textures))
	}
	if len(parent.models) != 0 || len(parent.textures) != 0 {
		t.Fatal("field leaked into avatar/menu cache")
	}
	owned := w.stream.assets
	var waterIDs []uint32
	if w.water != nil {
		for _, normal := range w.water.normals {
			if normal != nil {
				waterIDs = append(waterIDs, normal.ID)
			}
		}
		for _, frame := range w.water.frames {
			if frame != nil {
				waterIDs = append(waterIDs, frame.ID)
			}
		}
	}
	w.Close()
	for _, id := range waterIDs {
		if gl.IsTexture(id) {
			t.Fatal("water frame survived world close")
		}
	}
	for _, tex := range owned.textures {
		if tex != nil && gl.IsTexture(tex.ID) {
			t.Fatal("texture survived world close")
		}
	}
	if err := gl.GetError(); err != gl.NO_ERROR {
		t.Fatalf("GL resource lifetime error: %#x", err)
	}
}
