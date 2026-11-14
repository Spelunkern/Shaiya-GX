//go:build integration

package app

import (
	"os"
	"runtime"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
	"github.com/go-gl/gl/v4.5-core/gl"
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
		a.updateGameMenu(&platform.Input{MouseX: x + 160, MouseY: y + 500, Pressed: true, MouseDown: true}, 960, 600)
		a.updateGameMenu(&platform.Input{MouseX: x + 160, MouseY: y + 500, Clicked: true}, 960, 600)
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
