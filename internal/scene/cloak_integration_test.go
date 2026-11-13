//go:build integration

package scene

import (
	"math"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/data/model"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/go-gl/gl/v4.5-core/gl"
)

func TestOriginalCloakTablesAndBodiesGL(t *testing.T) {
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
	s := NewAssets(arc)
	renderer, err := render.NewScene3D()
	if err != nil {
		t.Fatal(err)
	}
	defer renderer.Delete()
	skin, err := render.NewSkinner()
	if err != nil {
		t.Fatal(err)
	}
	defer skin.Delete()
	renderer.SetCameraView(render.V3{0, 1.4, 3.5}, render.V3{0, 1, 0}, render.V3{0, 1, 0})
	renderer.Proj = render.PerspectiveFovLH(math.Pi/4, 800.0/600, .1, 50)
	renderer.LightDir = render.V3{.3, -1, -.5}
	renderer.Ambient = .8
	dir := os.Getenv("SHAIYA_TEST_ARTIFACTS")
	if dir == "" {
		dir = t.TempDir()
	}
	cases := 0
	for race := uint8(0); race < 4; race++ {
		for sex := uint8(0); sex < 2; sex++ {
			for _, job := range []uint8{0, 4} {
				if race == 3 && job == 4 {
					job = 3
				}
				if (race == 1 || race == 2) && job == 0 {
					job = 2
				}
				ap := Appearance{Race: race, Gender: sex, Job: job, Helmet: -1, CloakType: 24, Height: 2}
				prefix := cloakPrefix(ap)
				for fit := uint8(0); fit < 8; fit++ {
					raw, err := arc.ReadFile("cloak/Character/" + raceFolders[race] + "/" + string(rune('1'+fit)) + "_" + prefix + ".PC")
					if err != nil {
						t.Fatal(err)
					}
					pc, err := model.ParseCloakPC(raw)
					if err != nil || len(pc.Records) != 24 {
						t.Fatalf("PC %s fit %d: %v", prefix, fit, err)
					}
					for index, record := range pc.Records {
						if (record.Mode != 0 && record.Mode != 1) || len(record.Pins) != 5 {
							t.Fatal("unsupported native record", prefix, record)
						}
						ap.CloakFit, ap.CloakModel = fit, uint8(index)
						if _, err := s.cloakGeometry(ap); err != nil {
							t.Fatalf("cloak %s fit %d model %d: %v", prefix, fit, index, err)
						}
					}
					cases += len(pc.Records)
				}
				ap.CloakFit, ap.CloakModel = 0, 0
				actor, err := BuildCharacter(s, ap, "000_normal")
				if err != nil {
					t.Fatal(err)
				}
				if actor.cloak == nil {
					t.Fatalf("missing cloak %s: %v", prefix, s.Missing)
				}
				actor.World = render.Identity()
				for frame := 0; frame < 180; frame++ {
					actor.World = render.Basis(render.V3{float32(frame) / 600, 0, 0}, render.V3{float32(math.Sin(float64(frame) / 60)), 0, float32(math.Cos(float64(frame) / 60))}, render.V3{0, 1, 0})
					actor.Update(1.0 / 60)
					gl.Viewport(0, 0, 800, 600)
					gl.ClearColor(.15, .18, .22, 1)
					gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
					renderer.Begin()
					actor.Draw(skin, renderer)
					renderer.End()
					if frame == 0 && race == 0 && sex == 0 && job == 0 {
						if err := render.SavePNG(filepath.Join(dir, "cloak-native-humf-back.png"), 800, 600); err != nil {
							t.Fatal(err)
						}
					}
					for _, v := range actor.cloak.physics.positions {
						for _, x := range v {
							if math.IsNaN(float64(x)) || math.IsInf(float64(x), 0) {
								t.Fatalf("cloth exploded: %s", prefix)
							}
						}
					}
				}
				if race == 0 && sex == 0 && job == 0 {
					if err := render.SavePNG(filepath.Join(dir, "cloak-native-humf.png"), 800, 600); err != nil {
						t.Fatal(err)
					}
				}
			}
		}
	}
	if code := gl.GetError(); code != gl.NO_ERROR {
		t.Fatalf("cloak GL error %#x", code)
	}
	t.Logf("%d native PC configurations, meshes and textures loaded; all 16 bodies simulated and rendered", cases)
}
