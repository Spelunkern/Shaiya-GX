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
	"testing"
)

func TestStealthCharacterFrontAndRearGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" || os.Getenv("SHAIYA_DATA") == "" {
		t.Skip("requires original assets and GL")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Title: "stealth surfaces", Width: 800, Height: 600, Hidden: true, Debug: true})
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
	player, err := BuildCharacter(assets, Appearance{Race: 0, Job: 0, Helmet: -1, WeaponType: 1, WeaponModel: 0}, "000_normal")
	if err != nil {
		t.Fatal(err)
	}
	s, err := render.NewScene3D()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Delete()
	skin, err := render.NewSkinner()
	if err != nil {
		t.Fatal(err)
	}
	defer skin.Delete()
	s.Proj = render.PerspectiveFovLH(math.Pi/4, 800.0/600, .1, 100)
	s.View = render.LookAtLH(render.V3{0, 1.6, 3.5}, render.V3{0, 1.2, 0}, render.V3{0, 1, 0})
	s.FogStart, s.FogEnd = 0, 0
	for _, tc := range []struct {
		name         string
		hidden, back bool
	}{{"normal-rear", false, false}, {"stealth-rear", true, false}, {"normal-front", false, true}, {"stealth-front", true, true}} {
		player.World = render.Identity()
		if tc.back {
			player.World = render.Basis(render.V3{}, render.V3{0, 0, -1}, render.V3{0, 1, 0})
		}
		player.Concealed = tc.hidden
		gl.Viewport(0, 0, 800, 600)
		gl.DepthMask(true)
		gl.ClearDepth(1)
		gl.ClearColor(.2, .25, .3, 1)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		s.Begin()
		player.Draw(skin, s)
		if dir := os.Getenv("SHAIYA_TEST_ARTIFACTS"); dir != "" {
			if err = os.MkdirAll(dir, 0755); err != nil {
				t.Fatal(err)
			}
			if err = render.SavePNG(filepath.Join(dir, tc.name+".png"), 800, 600); err != nil {
				t.Fatal(err)
			}
		}
	}
}
