package app

import (
	"encoding/json"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

func TestNoVSyncSettingsAndApplyCallback(t *testing.T) {
	settings := ui.DefaultClientSettings()
	settings.NoVSync = true
	raw, err := json.Marshal(settings)
	if err != nil {
		t.Fatal(err)
	}
	var decoded ui.ClientSettings
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	if !decoded.NoVSync {
		t.Fatal("NoVSync lost during JSON round-trip")
	}

	calls := 0
	gotVSync := true
	a := &App{
		cfg: Config{
			ApplyVSync: func(enabled bool) {
				calls++
				gotVSync = enabled
			},
		},
		gameMenu: &ui.GameMenu{Settings: decoded},
	}
	a.applyClientSettings()
	if calls != 1 || gotVSync {
		t.Fatalf("ApplyVSync called %d times with %v, want 1 call with false", calls, gotVSync)
	}

	a.cfg.OverrideVSync = true
	a.gameMenu.Settings.NoVSync = false
	a.applyClientSettings()
	if calls != 1 {
		t.Fatalf("OverrideVSync did not suppress ApplyVSync (calls=%d)", calls)
	}
}
