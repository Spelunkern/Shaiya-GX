package ui

import (
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"testing"
)

func TestDisplayDropdownChoosesPendingModeAndCancelRestores(t *testing.T) {
	m, err := NewGameMenu(marketTextures{})
	if err != nil {
		t.Fatal(err)
	}
	m.SetDisplayModes([]platform.DisplayMode{{Width: 800, Height: 600}, {Width: 1024, Height: 768}, {Width: 1920, Height: 1080}})
	m.Open(MenuGraphics)
	x, y := m.Origin(1024, 768)
	click := func(px, py int) int {
		m.Update(&platform.Input{MouseX: x + px, MouseY: y + py, Pressed: true, MouseDown: true}, 1024, 768)
		return m.Update(&platform.Input{MouseX: x + px, MouseY: y + py, Clicked: true}, 1024, 768)
	}
	click(145, 90)
	if !m.DisplayPopupOpen() {
		t.Fatal("resolution arrow did not open")
	}
	click(80, 103+PlayerMenuTextInset+PlayerMenuRowHeight+5)
	if m.DisplayPopupOpen() || m.Settings.DisplayMode != (platform.DisplayMode{Width: 800, Height: 600}) {
		t.Fatal("wrong pending display mode", m.Settings.DisplayMode)
	}
	click(230, 500)
	if m.Page != MenuMain || m.Settings.DisplayMode != (platform.DisplayMode{}) {
		t.Fatal("cancel kept pending resolution")
	}
	m.Open(MenuGraphics)
	click(145, 90)
	m.Update(&platform.Input{Keys: []platform.Key{platform.KeyDown}}, 1024, 768)
	if m.Settings.DisplayMode != (platform.DisplayMode{Width: 800, Height: 600}) {
		t.Fatal("keyboard selection failed")
	}
	m.Update(&platform.Input{Keys: []platform.Key{platform.KeyEscape}}, 1024, 768)
	if m.DisplayPopupOpen() || m.Page != MenuGraphics {
		t.Fatal("dropdown escape closed the whole options page")
	}
}

func TestDisplayModeScrollbarDoesNotSelectUnderlyingRow(t *testing.T) {
	m, err := NewGameMenu(marketTextures{})
	if err != nil {
		t.Fatal(err)
	}
	var modes []platform.DisplayMode
	for i := 0; i < 12; i++ {
		modes = append(modes, platform.DisplayMode{Width: 800 + i*32, Height: 600 + i*16})
	}
	m.SetDisplayModes(modes)
	m.Open(MenuGraphics)
	x, y := m.Origin(1024, 768)
	click := func(px, py int) {
		m.Update(&platform.Input{MouseX: x + px, MouseY: y + py, Pressed: true, MouseDown: true}, 1024, 768)
		m.Update(&platform.Input{MouseX: x + px, MouseY: y + py, Clicked: true}, 1024, 768)
	}
	click(145, 90)
	click(68+95, 103+8+m.displayScroll.height-19+5)
	if !m.DisplayPopupOpen() || m.displayOffset != 1 || m.Settings.DisplayMode != (platform.DisplayMode{}) {
		t.Fatal("scrollbar selected a mode or dismissed popup")
	}
	click(80, 103+PlayerMenuTextInset+5)
	if m.Settings.DisplayMode != (platform.DisplayMode{Width: 800, Height: 600}) {
		t.Fatal("scrolled row did not select the displayed mode")
	}
}
