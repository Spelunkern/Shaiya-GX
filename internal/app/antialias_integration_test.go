//go:build integration

package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
	"github.com/go-gl/gl/v4.5-core/gl"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestAntialiasSceneAndMenuGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("GL and game resources")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
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
	a, e := New(Config{}, arc)
	if e != nil {
		t.Fatal(e)
	}
	defer a.Close()
	if a.clientSettings().Antialias != ui.AntialiasOff {
		t.Fatal("new installation enabled AA")
	}
	if e = a.Preview("world-water"); e != nil {
		t.Fatal(e)
	}
	slot := a.state.Slots[0]
	slot.Name = "Antialias"
	a.state.Slots[0] = slot
	a.frameDT = 1.0 / 60
	for i := 0; !a.SceneReady(); i++ {
		if i > 500 {
			t.Fatal("scene did not load")
		}
		a.Draw(960, 600)
	}
	modes := []int{ui.AntialiasOff, ui.AntialiasFXAA, ui.AntialiasTAA}
	for _, n := range a.aaSamples {
		if n == 2 {
			modes = append(modes, ui.AntialiasMSAA2)
		}
		if n == 4 {
			modes = append(modes, ui.AntialiasMSAA4)
		}
		if n == 8 {
			modes = append(modes, ui.AntialiasMSAA8)
		}
		if n == 16 {
			modes = append(modes, ui.AntialiasMSAA16)
		}
	}
	out := os.Getenv("SHAIYA_CAPTURE_DIR")
	if out == "" {
		out = t.TempDir()
	}
	if e = os.MkdirAll(out, 0700); e != nil {
		t.Fatal(e)
	}
	for _, mode := range modes {
		a.gameMenu.Settings.Antialias = mode
		for frame := 0; frame < 10; frame++ {
			a.Draw(960, 600)
		}
		if a.msaaFailed || a.fxaaFailed || a.taaFailed {
			t.Fatal("AA mode failed", mode)
		}
		if code := gl.GetError(); code != gl.NO_ERROR {
			t.Fatalf("mode%d GL error %#x", mode, code)
		}
		if mode == ui.AntialiasTAA {
			if a.taa == nil {
				t.Fatal("no TAA resources")
			}
			if e = render.SavePNG(filepath.Join(out, "taa-water.png"), 960, 600); e != nil {
				t.Fatal(e)
			}
		}
	}
	a.gameMenu.Settings.Antialias = ui.AntialiasOff
	a.Draw(960, 600)
	if a.msaaTarget != nil || a.fxaa != nil || a.taa != nil || a.scene3d.TemporalJitter != [2]float32{} {
		t.Fatal("Off retained AA resources or jitter")
	}
	a.menuOpen = true
	a.gameMenu.Open(ui.MenuGraphics)
	x, y := a.gameMenu.Origin(960, 600)
	a.gameMenu.Update(&platform.Input{MouseX: x + 274, MouseY: y + 90, Pressed: true, MouseDown: true}, 960, 600)
	a.gameMenu.Update(&platform.Input{MouseX: x + 274, MouseY: y + 90, Clicked: true}, 960, 600)
	if !a.gameMenu.DisplayPopupOpen() {
		t.Fatal("AA dropdown did not open")
	}
	a.gameMenu.Update(&platform.Input{Keys: []platform.Key{platform.KeyDown}}, 960, 600)
	a.gameMenu.Update(&platform.Input{Keys: []platform.Key{platform.KeyDown}}, 960, 600)
	if a.gameMenu.Settings.Antialias != ui.AntialiasTAA {
		t.Fatal("TAA not selectable")
	}
	a.Draw(960, 600)
	if e = render.SavePNG(filepath.Join(out, "taa-options.png"), 960, 600); e != nil {
		t.Fatal(e)
	}
	if code := gl.GetError(); code != gl.NO_ERROR {
		t.Fatalf("AA menu GL error %#x", code)
	}
	a.gameMenu.Open(ui.MenuUser)
	x, y = a.gameMenu.Origin(960, 600)
	if a.chatUI.AutoHide {
		t.Fatal("default chat auto-hides")
	}
	a.gameMenu.Update(&platform.Input{MouseX: x + 140, MouseY: y + 232, Pressed: true, MouseDown: true}, 960, 600)
	a.gameMenu.Update(&platform.Input{MouseX: x + 140, MouseY: y + 232, Clicked: true}, 960, 600)
	a.applyClientSettings()
	if !a.chatUI.AutoHide {
		t.Fatal("chat auto-hide option was not applied")
	}
	a.Draw(960, 600)
	if e = render.SavePNG(filepath.Join(out, "chat-options.png"), 960, 600); e != nil {
		t.Fatal(e)
	}
	a.gameMenu.DiscardPage()
	a.applyClientSettings()
	if a.chatUI.AutoHide {
		t.Fatal("cancelling options retained chat auto-hide")
	}
	if code := gl.GetError(); code != gl.NO_ERROR {
		t.Fatalf("chat menu GL error %#x", code)
	}
}
