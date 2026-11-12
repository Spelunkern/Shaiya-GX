package ui

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/platform"
)

func TestNativeOptionsMenuRoutesAllEightButtons(t *testing.T) {
	m, err := NewGameMenu(marketTextures{})
	if err != nil {
		t.Fatal(err)
	}
	for index := 0; index < 8; index++ {
		m.Open(MenuMain)
		x, y := m.Origin(1024, 768)
		in := platform.Input{MouseX: x + 70, MouseY: y + 80 + 32*index, Pressed: true, MouseDown: true}
		m.Update(&in, 1024, 768)
		in.Pressed, in.MouseDown, in.Clicked = false, false, true
		action := m.Update(&in, 1024, 768)
		if index < 6 && m.Page != MenuPage(index+1) {
			t.Fatal("wrong subpage", index, m.Page)
		}
		if index == 6 && action != MenuCharacters || index == 7 && action != MenuExit {
			t.Fatal("wrong session action", index, action)
		}
	}
}

func TestMenuSliderKeepsReleaseAndDoesNotUseWheel(t *testing.T) {
	m, err := NewGameMenu(marketTextures{})
	if err != nil {
		t.Fatal(err)
	}
	m.Open(MenuSound)
	x, y := m.Origin(1024, 768)
	before := m.Settings
	m.Update(&platform.Input{MouseX: x + 100, MouseY: y + 106, Wheel: -120}, 1024, 768)
	if m.Settings != before {
		t.Fatal("wheel changed volume")
	}
	m.Update(&platform.Input{MouseX: x + 34, MouseY: y + 106, Pressed: true, MouseDown: true}, 1024, 768)
	m.Update(&platform.Input{MouseX: x + 500, MouseY: y + 300, MouseDown: true}, 1024, 768)
	m.Update(&platform.Input{MouseX: x - 50, MouseY: y + 300, Clicked: true}, 1024, 768)
	if m.Settings.Music != 0 || m.dragSlider != -1 {
		t.Fatal("slider lost drag/release", m.Settings, m.dragSlider)
	}
}

func TestOptionsCancelRestoresPreviewAndOKCommits(t *testing.T) {
	m, err := NewGameMenu(marketTextures{})
	if err != nil {
		t.Fatal(err)
	}
	before := m.Settings
	m.Open(MenuSound)
	m.Settings.Music = 0
	if m.SavedSettings() != before {
		t.Fatal("draft leaked into saved settings")
	}
	x, y := m.Origin(1024, 768)
	m.Update(&platform.Input{MouseX: x + 180, MouseY: y + 367, Pressed: true, MouseDown: true}, 1024, 768)
	action := m.Update(&platform.Input{MouseX: x + 180, MouseY: y + 367, Clicked: true}, 1024, 768)
	if action != MenuChanged || m.Page != MenuMain || m.Settings != before {
		t.Fatal("cancel did not restore preview", action, m.Settings)
	}
	m.Open(MenuSound)
	m.Settings.Music = 0
	x, y = m.Origin(1024, 768)
	m.Update(&platform.Input{MouseX: x + 110, MouseY: y + 367, Pressed: true, MouseDown: true}, 1024, 768)
	action = m.Update(&platform.Input{MouseX: x + 110, MouseY: y + 367, Clicked: true}, 1024, 768)
	if action != MenuSave || m.Page != MenuMain || m.SavedSettings().Music != 0 {
		t.Fatal("OK did not commit", action, m.Settings)
	}
}
