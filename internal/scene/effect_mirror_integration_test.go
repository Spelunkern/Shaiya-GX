//go:build integration

package scene

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/go-gl/gl/v4.5-core/gl"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestOriginalMirroredMapLampGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL and original assets")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Width: 800, Height: 600, Hidden: true, Debug: true})
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
	assets := NewAssets(arc)
	w, err := loadStreamedField(assets, "1")
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	f := w.mapFX
	if f == nil {
		t.Fatal("original map has no effect library")
	}
	var x *Effect
	for seq, sequence := range f.library.Sequences {
		for _, record := range sequence.Records {
			d := f.library.Effects[record.Effect]
			for _, texture := range d.Textures {
				if d.Unk29 == 0 || !strings.Contains(strings.ToLower(f.library.Textures[texture]), "lamp003") {
					continue
				}
				x, err = loadEffectLibrary(assets, f.file, seq, f.library)
				if err != nil {
					t.Fatal(err)
				}
				for _, emitter := range x.emitters {
					if emitter.definition == &f.library.Effects[record.Effect] {
						x.emitters = []effectEmitter{emitter}
						break
					}
				}
				break
			}
			if x != nil {
				break
			}
		}
		if x != nil {
			break
		}
	}
	if x == nil {
		t.Fatal("original lamp003 mirror flag missing")
	}
	s, err := render.NewScene3D()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Delete()
	s.View = render.LookAtLH(render.V3{0, 0, -5}, render.V3{}, render.V3{0, 1, 0})
	s.Proj = render.PerspectiveFovLH(math.Pi/4, 800.0/600, .1, 100)
	x.SetTransform(render.Identity())
	for range 90 {
		x.Update(1.0 / 60)
	}
	if len(x.emitters[0].particles) == 0 {
		t.Fatal("map lamp did not emit")
	}
	gl.Viewport(0, 0, 800, 600)
	gl.ClearColor(.03, .03, .03, 1)
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	x.Draw(s)
	if code := gl.GetError(); code != gl.NO_ERROR {
		t.Fatalf("mirror GL error %#x", code)
	}
	dir := os.Getenv("SHAIYA_TEST_ARTIFACTS")
	if dir == "" {
		dir = t.TempDir()
	}
	if err = render.SavePNG(filepath.Join(dir, "effects-mirror-lamp.png"), 800, 600); err != nil {
		t.Fatal(err)
	}
	t.Logf("original %s lamp003 emitter %q mirrored inward", f.file, x.emitters[0].definition.Name)
}
