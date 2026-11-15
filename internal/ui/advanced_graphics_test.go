package ui

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/platform"
)

func TestAdvancedGraphicsWindowEditsAndCancels(t *testing.T) {
	m, err := NewGameMenu(marketTextures{})
	if err != nil {
		t.Fatal(err)
	}
	m.Open(MenuGraphics)
	const w, h = 1600, 900
	x, y := m.Origin(w, h)
	click := func(mx, my int) {
		m.Update(&platform.Input{MouseX: mx, MouseY: my, Pressed: true, MouseDown: true}, w, h)
		m.Update(&platform.Input{MouseX: mx, MouseY: my, Clicked: true}, w, h)
	}
	click(x+advancedButtonX+10, y+advancedButtonY+10)
	ax, ay, _, _, ok := m.AdvancedRect(w, h)
	if !ok || ax <= x {
		t.Fatal("Advanced window did not open beside the page")
	}
	// Antialias is the first row; open its list and pick FXAA.
	row := 0
	click(ax+advancedArrowX+5, ay+advancedRowY+row*advancedRowStep)
	if !m.DisplayPopupOpen() {
		t.Fatal("row list did not open")
	}
	popupY := m.advancedPopupY(ay, row)
	click(ax+advancedPopupX+10, popupY+PlayerMenuTextInset+1*PlayerMenuRowHeight+5)
	if m.Settings.Antialias != AntialiasFXAA || m.DisplayPopupOpen() {
		t.Fatalf("antialias = %d, popup open %v", m.Settings.Antialias, m.DisplayPopupOpen())
	}
	// VSync is the second row: Off stores the inverted setting.
	click(ax+advancedArrowX+5, ay+advancedRowY+advancedRowStep)
	click(ax+advancedPopupX+10, m.advancedPopupY(ay, 1)+PlayerMenuTextInset+5)
	if !m.Settings.NoVSync {
		t.Fatal("VSync Off was not stored")
	}
	// Escape closes the Advanced window first, then the page.
	m.Update(&platform.Input{Keys: []platform.Key{platform.KeyEscape}}, w, h)
	if _, _, _, _, open := m.AdvancedRect(w, h); open || m.Page != MenuGraphics {
		t.Fatal("escape did not close only the Advanced window")
	}
	m.Update(&platform.Input{Keys: []platform.Key{platform.KeyEscape}}, w, h)
	if m.Page != MenuMain || m.Settings.Antialias != AntialiasOff || m.Settings.NoVSync {
		t.Fatal("leaving the page kept the unsaved advanced change")
	}
}

func TestAdvancedGraphicsOptionsRoundTrip(t *testing.T) {
	m := &GameMenu{Settings: DefaultClientSettings()}
	for _, opt := range AdvancedGraphicsOptions {
		values := opt.Values(m)
		if len(values) < 2 || opt.Label == "" {
			t.Fatalf("%q has %d values", opt.Label, len(values))
		}
		for i := range values {
			opt.Set(m, i)
			if got := opt.Get(m); got != i {
				t.Fatalf("%q set %d got %d", opt.Label, i, got)
			}
		}
	}
	if !m.Settings.Valid() {
		t.Fatal("advanced options produced invalid settings")
	}
}
