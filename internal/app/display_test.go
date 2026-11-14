package app

import (
	"errors"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
	"testing"
)

func TestDisplayAppliesOnlyOnOKAndFailureRestoresSelection(t *testing.T) {
	a := inputApp()
	a.menuOpen = true
	a.gameMenu.SetDisplayModes([]platform.DisplayMode{{Width: 800, Height: 600}, {Width: 1024, Height: 768}})
	a.gameMenu.Open(ui.MenuGraphics)
	var calls []platform.DisplayMode
	a.cfg.ApplyDisplayMode = func(mode platform.DisplayMode) error { calls = append(calls, mode); return nil }
	x, y := a.gameMenu.Origin(1024, 768)
	click := func(px, py int) {
		a.updateGameMenu(&platform.Input{MouseX: x + px, MouseY: y + py, Pressed: true, MouseDown: true}, 1024, 768)
		a.updateGameMenu(&platform.Input{MouseX: x + px, MouseY: y + py, Clicked: true}, 1024, 768)
	}
	click(145, 90)
	click(80, 103+ui.PlayerMenuTextInset+5)
	if len(calls) != 0 {
		t.Fatal("pending resolution changed the display")
	}
	click(160, 500)
	if len(calls) != 1 || calls[0] != (platform.DisplayMode{Width: 800, Height: 600}) {
		t.Fatal("OK did not apply", calls)
	}
	a.gameMenu.Open(ui.MenuGraphics)
	a.gameMenu.Settings.DisplayMode = platform.DisplayMode{Width: 1024, Height: 768}
	a.cfg.ApplyDisplayMode = func(platform.DisplayMode) error { return errors.New("driver refused test mode") }
	click(160, 500)
	if a.gameMenu.Page != ui.MenuGraphics || a.gameMenu.DisplayError == "" || a.gameMenu.Settings.DisplayMode != (platform.DisplayMode{Width: 800, Height: 600}) {
		t.Fatal("failed display change did not recover")
	}
	retried := false
	a.cfg.ApplyDisplayMode = func(platform.DisplayMode) error { retried = true; return nil }
	click(160, 500)
	if !retried {
		t.Fatal("failed change left a stale applied-mode cache")
	}
}
func TestDisplayDropdownConsumesEnterBeforeChat(t *testing.T) {
	a := inputApp()
	a.menuOpen = true
	a.gameMenu.SetDisplayModes([]platform.DisplayMode{{Width: 800, Height: 600}, {Width: 1024, Height: 768}})
	a.gameMenu.Open(ui.MenuGraphics)
	x, y := a.gameMenu.Origin(1024, 768)
	for _, in := range []platform.Input{{MouseX: x + 145, MouseY: y + 90, Pressed: true, MouseDown: true}, {MouseX: x + 145, MouseY: y + 90, Clicked: true}} {
		a.gameMenu.Update(&in, 1024, 768)
	}
	controls, captured := a.gameMenuControls(&platform.Input{Keys: []platform.Key{platform.KeyEnter}, Down: "W"}, 1024, 768)
	if !captured || controls.HasKey(platform.KeyEnter) || controls.Down != "W" || a.chatOpen {
		t.Fatal("dropdown input leaked or stopped held movement", controls)
	}
}

func TestDisplaySettingsPersistAndExplicitWindowOverrides(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configDir)
	t.Setenv("APPDATA", configDir)
	mode := platform.DisplayMode{Width: 800, Height: 600}
	a := inputApp()
	a.cfg.RememberLayout = true
	a.gameMenu.Settings.DisplayMode = mode
	a.gameMenu.Open(ui.MenuGraphics)
	a.saveClientSettings()
	for _, override := range []bool{false, true} {
		b := inputApp()
		b.cfg.RememberLayout = true
		b.cfg.OverrideDisplay = override
		b.gameMenu.SetDisplayModes([]platform.DisplayMode{mode})
		var applied platform.DisplayMode
		b.cfg.ApplyDisplayMode = func(m platform.DisplayMode) error { applied = m; return nil }
		b.loadClientSettings()
		want := mode
		if override {
			want = platform.DisplayMode{}
		}
		if applied != want || b.gameMenu.Settings.DisplayMode != want || !b.displayConfigured {
			t.Fatalf("override=%v: applied=%v settings=%v", override, applied, b.gameMenu.Settings.DisplayMode)
		}
	}
}

func TestWindowedChoicesMigrateAndPersist(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configDir)
	t.Setenv("APPDATA", configDir)
	for _, mode := range []platform.DisplayMode{{Width: 1600, Height: 900, Kind: platform.DisplayWindowed}, {Width: 1280, Height: 720}} {
		a := inputApp()
		a.cfg.RememberLayout = true
		a.gameMenu.Settings.DisplayMode = mode
		a.gameMenu.Open(ui.MenuGraphics)
		a.saveClientSettings()
		b := inputApp()
		b.cfg.RememberLayout = true
		b.gameMenu.SetDisplayModes([]platform.DisplayMode{mode})
		var got platform.DisplayMode
		b.cfg.ApplyDisplayMode = func(m platform.DisplayMode) error { got = m; return nil }
		b.loadClientSettings()
		if got != platform.WindowMode(mode) || b.gameMenu.Settings.DisplayMode != platform.WindowMode(mode) {
			t.Fatal("display kind did not persist", mode, got)
		}
	}
}
