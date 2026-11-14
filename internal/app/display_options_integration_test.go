//go:build integration

package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestDisplayOptionsPopupGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL and assets")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	w, err := platform.Open(platform.Config{Width: 1024, Height: 768, Hidden: true})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	if err = render.Init(false); err != nil {
		t.Fatal(err)
	}
	arc, err := archive.Open(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	var applied []platform.DisplayMode
	cfg := Config{DisplayModes: func() []platform.DisplayMode {
		return []platform.DisplayMode{{Width: 800, Height: 600}, {Width: 1024, Height: 768}, {Width: 1280, Height: 720}, {Width: 1920, Height: 1080}}
	}, ApplyDisplayMode: func(m platform.DisplayMode) error { applied = append(applied, m); return nil }}
	a, err := New(cfg, arc)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	if err = a.Preview("world-graphics"); err != nil {
		t.Fatal(err)
	}
	for range 100 {
		a.Draw(1024, 768)
	}
	x, y := a.gameMenu.Origin(1024, 768)
	click := func(px, py int) {
		a.updateGameMenu(&platform.Input{MouseX: x + px, MouseY: y + py, Pressed: true, MouseDown: true}, 1024, 768)
		a.updateGameMenu(&platform.Input{MouseX: x + px, MouseY: y + py, Clicked: true}, 1024, 768)
	}
	click(145, 90)
	if !a.gameMenu.DisplayPopupOpen() {
		t.Fatal("resolution popup did not open")
	}
	a.Draw(1024, 768)
	dir := os.Getenv("SHAIYA_TEST_ARTIFACTS")
	if dir == "" {
		dir = t.TempDir()
	}
	os.MkdirAll(dir, 0755)
	if err = render.SavePNG(filepath.Join(dir, "display-resolution-popup.png"), 1024, 768); err != nil {
		t.Fatal(err)
	}
	a.gameMenuControls(&platform.Input{Keys: []platform.Key{platform.KeyUp}}, 1024, 768)
	a.gameMenuControls(&platform.Input{Keys: []platform.Key{platform.KeyEnter}}, 1024, 768)
	if a.chatOpen || a.gameMenu.DisplayPopupOpen() || a.gameMenu.Settings.DisplayMode != (platform.DisplayMode{Width: 800, Height: 600}) || len(applied) != 0 {
		t.Fatal("pending resolution leaked input or applied early")
	}
	click(160, 500)
	if len(applied) != 1 || applied[0] != (platform.DisplayMode{Width: 800, Height: 600}) || a.gameMenu.Page != ui.MenuMain {
		t.Fatal("mode did not commit on OK", applied)
	}
	// Real options contain only numeric resolutions, all applied as windows.
	a.gameMenu.SetDisplayModes(w.DisplayModes())
	a.cfg.DisplayModes = w.DisplayModes
	a.gameMenu.SetCurrentDisplayMode(w.CurrentDisplayMode())
	a.gameMenu.Open(ui.MenuGraphics)
	click(145, 90)
	a.gameMenuControls(&platform.Input{Keys: []platform.Key{platform.KeyDown}}, 1024, 768)
	selected := a.gameMenu.Settings.DisplayMode
	for _, mode := range a.gameMenu.DisplayModes {
		if mode.Width <= 0 || mode.Label() == "Window" || mode.Label() == "Fullscreen" {
			t.Fatal("special display choice remains")
		}
	}
	a.Draw(1024, 768)
	if err = render.SavePNG(filepath.Join(dir, "window-resolution-popup.png"), 1024, 768); err != nil {
		t.Fatal(err)
	}
	a.gameMenuControls(&platform.Input{Keys: []platform.Key{platform.KeyEnter}}, 1024, 768)
	click(160, 500)
	if len(applied) != 2 || applied[1] != selected {
		t.Fatal("window resolution failed to commit", applied)
	}
}
