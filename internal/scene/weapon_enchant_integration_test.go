//go:build integration

package scene

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
	"github.com/go-gl/gl/v4.5-core/gl"
)

func TestOriginalWeaponEnchantCatalogueGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires original assets and GL")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Title: "Weapon.Seff", Width: 1200, Height: 840, Hidden: true, Debug: true})
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
	a := NewAssets(arc)
	s, err := render.NewScene3D()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Delete()
	k, err := render.NewSkinner()
	if err != nil {
		t.Fatal(err)
	}
	defer k.Delete()
	f, err := ui.LoadFont(13)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Delete()
	sprites, err := render.NewSprites(1024)
	if err != nil {
		t.Fatal(err)
	}
	var actors []*Character
	layers := 0
	for family := uint8(1); family <= 15; family++ {
		for level := uint8(5); level <= 10; level++ {
			c, err := BuildCharacter(a, Appearance{Race: 0, Helmet: -1, ShowHelmet: true, WeaponType: family, WeaponEnchant: level}, "000_normal")
			if err != nil {
				t.Fatal(family, level, err)
			}
			want := 1
			if family == 5 || family == 15 {
				want = 2
			}
			records := c.WeaponEnchantRecords()
			if len(records) != want {
				t.Fatalf("family%d +%d: attached recipes %v, want%d; missing %v", family, level, records, want, a.Missing)
			}
			for _, record := range records {
				if record != int32(6*(family-1)+level-4) {
					t.Fatal("wrong family/tier", records)
				}
			}
			layers += len(c.attachments[0].enchant.recipe.layers)
			c.Update(1.2)
			actors = append(actors, c)
		}
	}
	if layers != 186 || len(a.weaponSEFF.Records) != 162 || len(a.weaponRecipes) != 90 {
		t.Fatal("original catalogue coverage", layers, len(a.weaponRecipes))
	}
	gl.Viewport(0, 0, 1200, 840)
	gl.ClearColor(.025, .025, .035, 1)
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	for i, c := range actors {
		family, rank := i/6, i%6
		render.InViewport(family*80, rank*140, 80, 124, 840, func() {
			s.SetCameraView(render.V3{0, 1.2, 4.5}, render.V3{0, 1.0, 0}, render.V3{0, 1, 0})
			s.Proj = render.PerspectiveFovLH(math.Pi/4, 80.0/124, .1, 100)
			s.Begin()
			c.World = render.Identity()
			c.Draw(k, s)
			s.End()
		})
	}
	sprites.Begin(1200, 840)
	for i := range actors {
		f.Draw(sprites, fmt.Sprintf("%02d +%d", i/6+1, i%6+5), i/6*80+12, i%6*140+124, [4]float32{1, 1, 1, 1}, 78)
	}
	sprites.End()
	dir := os.Getenv("SHAIYA_TEST_ARTIFACTS")
	if dir == "" {
		dir = t.TempDir()
	}
	if err = render.SavePNG(filepath.Join(dir, "weapon-enchants-90.png"), 1200, 840); err != nil {
		t.Fatal(err)
	}
	for _, level := range []uint8{0, 4, 50, 255} {
		c, err := BuildCharacter(a, Appearance{Helmet: -1, WeaponType: 1, WeaponEnchant: level, ShieldType: 19, OffhandEnchant: 10}, "000_normal")
		if err != nil || len(c.WeaponEnchantRecords()) != 0 {
			t.Fatal("low tier, armor encoding or shield emitted weapon glow", level, err)
		}
	}
	// Compare only weapon effects against the same clear background; the
	// actual bone/extra matrices still come from a real animated character.
	c := actors[5]
	viewport := func(hidden bool) []byte {
		gl.Viewport(0, 0, 1200, 840)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		s.SetCameraView(render.V3{0, 1, -3}, render.V3{0, 1, 0}, render.V3{0, 1, 0})
		s.Proj = render.PerspectiveFovLH(math.Pi/4, 1200.0/840, .1, 100)
		s.HideWeaponEffects = hidden
		s.Begin()
		for i := range c.attachments {
			p := &c.attachments[i]
			c.drawWeaponEnchant(s, p, p.extra.Mul(c.preparePose()[p.bone]).Mul(c.World))
		}
		s.End()
		out := make([]byte, 1200*840*4)
		gl.ReadPixels(0, 0, 1200, 840, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&out[0]))
		return out
	}
	without, with := viewport(true), viewport(false)
	changed := 0
	for i := 0; i < len(with); i += 4 {
		if with[i] != without[i] || with[i+1] != without[i+1] || with[i+2] != without[i+2] {
			changed++
		}
	}
	if changed < 100 {
		t.Fatal("weapon enchant did not reach framebuffer", changed)
	}
	render.ResetDrawCalls()
	start := time.Now()
	s.Begin()
	for range 80 {
		for i := range c.attachments {
			p := &c.attachments[i]
			c.drawWeaponEnchant(s, p, p.extra.Mul(c.preparePose()[p.bone]).Mul(c.World))
		}
	}
	s.End()
	t.Logf("90 recipes /186 layers; visible pixels%d; 80 +10 weapons CPU%s, draws%d", changed, time.Since(start), render.DrawCalls())
	if code := gl.GetError(); code != gl.NO_ERROR {
		t.Fatalf("weapon SEFF GL error %#x", code)
	}
}
