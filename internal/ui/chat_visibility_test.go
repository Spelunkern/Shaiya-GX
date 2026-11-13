package ui

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/platform"
)

func TestChatAutoHideIsOptionalAndDoesNotCaptureHiddenInput(t *testing.T) {
	now := time.Unix(100, 0)
	previous := chatNow
	chatNow = func() time.Time { return now }
	t.Cleanup(func() { chatNow = previous })
	c := &ChatPanel{Rows: [2]int{5, 5}}
	if c.paneAlpha(0, now) != 1 || !c.Contains(100, 690, 768) {
		t.Fatal("default chat must remain visible and clickable")
	}
	c.AutoHide = true
	if c.paneAlpha(0, now) != 0 || c.Contains(100, 690, 768) {
		t.Fatal("hidden chat captured the world")
	}
	c.ShowHistory()
	now = now.Add(chatHold + chatFade/2)
	if c.paneAlpha(0, now) != .5 || c.paneAlpha(1, now) != .5 {
		t.Fatal("both histories must fade together")
	}
	c.Update(&platform.Input{MouseX: 100, MouseY: 430}, 768)
	if c.paneAlpha(0, now) != 1 {
		t.Fatal("hover did not hold visible history")
	}
	now = now.Add(chatHold + chatFade)
	if c.paneAlpha(0, now) != 0 {
		t.Fatal("history did not hide after inactivity")
	}
	c.Typing = true
	if c.paneAlpha(0, now) != 1 || !c.inputShown() {
		t.Fatal("typing did not reveal chat")
	}
}

func TestChatAutoHideOptionPreviewCancelAndPersistence(t *testing.T) {
	m, err := NewGameMenu(marketTextures{})
	if err != nil {
		t.Fatal(err)
	}
	if m.Settings.AutoHideChat {
		t.Fatal("auto-hide must default to off")
	}
	m.Open(MenuUser)
	x, y := m.Origin(1024, 768)
	m.Update(&platform.Input{MouseX: x + 140, MouseY: y + 232, Pressed: true, MouseDown: true}, 1024, 768)
	if m.Update(&platform.Input{MouseX: x + 140, MouseY: y + 232, Clicked: true}, 1024, 768) != MenuChanged || !m.Settings.AutoHideChat {
		t.Fatal("option did not enable auto-hide")
	}
	m.DiscardPage()
	if m.Settings.AutoHideChat {
		t.Fatal("cancel retained preview")
	}
	m.Settings.AutoHideChat = true
	raw, err := json.Marshal(m.Settings)
	if err != nil {
		t.Fatal(err)
	}
	var restored ClientSettings
	if err = json.Unmarshal(raw, &restored); err != nil || !restored.AutoHideChat {
		t.Fatal("auto-hide preference did not persist", err)
	}
	restored = DefaultClientSettings()
	if err = json.Unmarshal([]byte(`{"ChatRows":[5,5]}`), &restored); err != nil || restored.AutoHideChat {
		t.Fatal("legacy settings enabled auto-hide", err)
	}
}
