//go:build integration

package app

import (
	"os"
	"runtime"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/render/gl"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

func TestAAAppliesAfterRestartGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("GL and field resources")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("APPDATA", dir)
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
	cfg := Config{RememberLayout: true, DisplayModes: win.DisplayModes, CurrentDisplayMode: win.CurrentDisplayMode, ApplyDisplayMode: win.ApplyDisplayMode}
	modes := []int{ui.AntialiasFXAA, ui.AntialiasTAA}
	for _, n := range render.SupportedMSAASamples() {
		if n == 4 {
			modes = append(modes, ui.AntialiasMSAA4)
		}
	}
	for _, mode := range modes {
		a, e := New(cfg, arc)
		if e != nil {
			t.Fatal(e)
		}
		a.menuOpen = true
		a.gameMenu.Open(ui.MenuGraphics)
		a.gameMenu.Settings.Antialias = mode
		x, y := a.gameMenu.Origin(960, 600)
		a.updateGameMenu(&platform.Input{MouseX: x + 160, MouseY: y + 498, Pressed: true, MouseDown: true}, 960, 600)
		a.updateGameMenu(&platform.Input{MouseX: x + 160, MouseY: y + 498, Clicked: true}, 960, 600)
		if a.gameMenu.Page != ui.MenuMain {
			t.Fatal("graphics OK did not save")
		}
		a.Close()
		b, e := New(cfg, arc)
		if e != nil {
			t.Fatal(e)
		}
		func() {
			defer b.Close()
			if b.clientSettings().Antialias != mode {
				t.Fatal("AA did not survive restart", mode, b.clientSettings().Antialias)
			}
			if e = b.Preview("world-water"); e != nil {
				t.Fatal(e)
			}
			for frame := 0; frame < 500 && !b.SceneReady(); frame++ {
				b.Draw(960, 600)
			}
			for frame := 0; frame < 8; frame++ {
				b.Draw(960, 600)
			}
			if b.aaMode != mode || b.msaaFailed || b.fxaaFailed || b.taaFailed {
				t.Fatal("saved mode failed to render", mode)
			}
			if mode == ui.AntialiasFXAA && b.fxaa == nil || mode == ui.AntialiasTAA && b.taa == nil || mode == ui.AntialiasMSAA4 && b.msaaTarget == nil {
				t.Fatal("saved AA was only a label; renderer did not activate", mode)
			}
			if code := gl.GetError(); code != gl.NO_ERROR {
				t.Fatalf("saved AA GL error %#x", code)
			}
		}()
	}
}

func TestDetailsSliderPreviewsWorldBeforeSavingGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("GL and field resources")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("APPDATA", t.TempDir())
	const width, height = 1024, 768
	win, err := platform.Open(platform.Config{Width: width, Height: height, Hidden: true})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
	if err = render.Init(false); err != nil {
		t.Fatal(err)
	}
	arc, err := archive.OpenDirectory(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	a, err := New(Config{}, arc)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	if err = a.Preview("world-stream"); err != nil {
		t.Fatal(err)
	}
	a.openGameMenu()
	a.gameMenu.Open(ui.MenuGraphics)
	x, y := a.gameMenu.Origin(width, height)
	a.Draw(width, height)
	high := a.scene3d.Proj

	a.gameMenuControls(&platform.Input{MouseX: x + 275, MouseY: y + 284, Pressed: true, MouseDown: true}, width, height)
	a.gameMenuControls(&platform.Input{MouseX: x + 155, MouseY: y + 284, MouseDown: true}, width, height)
	a.Draw(width, height)
	medium := a.scene3d.Proj
	if a.scene3d.GeometryReduction != 1 || medium == high {
		t.Fatal("medium detail did not render during the drag")
	}
	a.gameMenuControls(&platform.Input{MouseX: x + 35, MouseY: y + 284, MouseDown: true}, width, height)
	a.Draw(width, height)
	if a.scene3d.GeometryReduction != 2 || a.scene3d.Proj == medium || a.gameMenu.SavedSettings().Details != 2 {
		t.Fatal("low detail did not preview independently of the saved value")
	}
	a.gameMenuControls(&platform.Input{MouseX: x + 35, MouseY: y + 284, Clicked: true}, width, height)
	a.gameMenuControls(&platform.Input{Keys: []platform.Key{platform.KeyEscape}}, width, height)
	a.Draw(width, height)
	if a.scene3d.GeometryReduction != 0 || a.scene3d.Proj != high {
		t.Fatal("cancel did not restore the high-detail projection")
	}
	if code := gl.GetError(); code != gl.NO_ERROR {
		t.Fatalf("details preview GL error %#x", code)
	}
}
