package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

func TestAmbientSettingsMigrateAndPersist(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("APPDATA", dir)
	p, err := settingsPath()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(`{"Music":0.5,"Effects":0.25}`), 0600); err != nil {
		t.Fatal(err)
	}
	a := inputApp()
	a.cfg.RememberLayout = true
	a.loadClientSettings()
	if a.gameMenu.Settings.Ambient != 1 || a.gameMenu.Settings.Music != .5 || a.gameMenu.Settings.Effects != .25 {
		t.Fatal("old settings did not retain a usable ambient default")
	}
	a.gameMenu.Settings.Ambient = .5
	a.gameMenu.Open(ui.MenuSound)
	a.saveClientSettings()
	b := inputApp()
	b.cfg.RememberLayout = true
	b.loadClientSettings()
	if b.gameMenu.Settings.Ambient != .5 {
		t.Fatal("ambient volume did not survive restart")
	}
}
