package app

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

func hasChar(in *platform.Input, want rune) bool {
	return strings.ContainsRune(strings.ToLower(string(in.Chars)), want)
}

func (a *App) openGameMenu() {
	if a.gameMenu == nil {
		return
	}
	a.menuOpen = true
	if a.menuConfirm != nil {
		a.menuConfirm.Visible = false
	}
	a.menuIntent = ui.MenuNone
	a.gameMenu.Open(ui.MenuMain)
	a.tooltip = nil
}

// Options is an ordinary window. It owns pointer gestures started inside it,
// while held movement, automatic travel and gameplay outside keep running.
func (a *App) gameMenuControls(in *platform.Input, w, h int) (platform.Input, bool) {
	controls := *in
	modal := a.menuPending || a.menuConfirm != nil && a.menuConfirm.Visible
	width, height := a.gameMenu.Size()
	x, y := a.gameMenu.Origin(w, h)
	over := pointInside(in, x, y, width, height)
	v := a.wv
	if in.Pressed && over {
		v.uiLeft = -3
	}
	if in.RightPressed && over {
		v.uiRight = true
	}
	owned := v.uiLeft == -3 || v.uiRight || a.windowDrag.position == &a.gameMenu.Position
	cameraDrag := v.mouselook && in.RightDown && !in.RightPressed && !v.uiRight
	captured := owned || over && !cameraDrag && (in.Pressed || in.Clicked || in.MouseDown || in.RightPressed || in.RightDown || in.Wheel != 0)
	menuInput := *in
	if !captured && !modal {
		menuInput.Pressed, menuInput.Clicked, menuInput.MouseDown = false, false, false
		menuInput.RightPressed, menuInput.RightDown = false, false
		menuInput.Wheel = 0
	}
	a.updateGameMenu(&menuInput, w, h)
	if modal || a.menuPending || a.menuConfirm != nil && a.menuConfirm.Visible {
		return platform.Input{MouseX: in.MouseX, MouseY: in.MouseY, SuppressAutoRun: true}, true
	}
	if in.HasKey(platform.KeyEscape) || hasChar(in, 'o') {
		controls.Keys = nil
		controls.Chars = nil
	}
	if captured {
		controls.Pressed, controls.Clicked, controls.MouseDown = false, false, false
		controls.RightPressed, controls.RightDown, controls.MiddlePressed = false, false, false
		controls.Wheel = 0
	}
	if in.Clicked || !in.MouseDown {
		if v.uiLeft == -3 {
			v.uiLeft = 0
		}
	}
	if !in.RightDown {
		v.uiRight = false
	}
	return controls, captured || in.HasKey(platform.KeyEscape) || hasChar(in, 'o')
}

func (a *App) updateGameMenu(in *platform.Input, w, h int) {
	if a.menuPending {
		if a.clock >= a.menuDeadline {
			a.backToLogin("Character screen response timed out.")
		}
		return
	}
	if a.menuConfirm.Visible {
		if a.menuConfirm.Update(in, w, h) == ui.MessageOK {
			if a.world == nil {
				return
			}
			if err := a.world.Send(proto.LogoutPacket()); err != nil {
				a.backToLogin("Unable to leave the map.")
				return
			}
			a.menuPending, a.menuDeadline = true, a.clock+10
		}
		return
	}
	// Header dragging stays above every underlying window and owns release.
	width, height := a.gameMenu.Size()
	x, y := a.gameMenu.Origin(w, h)
	if a.windowDrag.position == &a.gameMenu.Position {
		a.updateWindowDrag(in, w, h)
		return
	}
	if in.Pressed && in.MouseX >= x+8 && in.MouseX < x+width-30 && in.MouseY >= y+5 && in.MouseY < y+30 {
		a.windowDrag = windowDrag{position: &a.gameMenu.Position, offsetX: in.MouseX - x, offsetY: in.MouseY - y, width: width, height: height}
		return
	}
	if hasChar(in, 'o') {
		a.gameMenu.DiscardPage()
		a.applyClientSettings()
		a.menuOpen = false
		return
	}
	switch action := a.gameMenu.Update(in, w, h); action {
	case ui.MenuClose:
		a.menuOpen = false
	case ui.MenuCharacters, ui.MenuExit:
		a.menuIntent = action
		text := "Return to the character screen?"
		if action == ui.MenuExit {
			text = "Exit the game?"
		}
		a.menuConfirm.Open(text, true)
	case ui.MenuChanged:
		a.applyClientSettings()
	case ui.MenuSave:
		a.applyClientSettings()
		a.saveClientSettings()
	}
}

func (a *App) clientSettings() ui.ClientSettings {
	if a.gameMenu != nil {
		return a.gameMenu.Settings
	}
	return ui.DefaultClientSettings()
}

func (a *App) applyClientSettings() {
	if a.chatUI != nil {
		a.chatUI.Rows = a.clientSettings().ChatRows
	}
	if a.sound != nil {
		s := a.clientSettings()
		a.sound.SetMix(s.Music, s.Effects)
	}
}

func settingsPath() (string, error) {
	dir, err := os.UserConfigDir()
	return filepath.Join(dir, "shaiya-go", "settings.json"), err
}

func (a *App) loadClientSettings() {
	if !a.cfg.RememberLayout || a.gameMenu == nil {
		return
	}
	path, err := settingsPath()
	var data []byte
	if err == nil {
		data, err = os.ReadFile(path)
	}
	if errors.Is(err, os.ErrNotExist) {
		return
	}
	value := ui.DefaultClientSettings()
	if err == nil {
		err = json.Unmarshal(data, &value)
	}
	if err != nil {
		log.Printf("load settings: %v", err)
		return
	}
	if !value.Valid() {
		log.Print("load settings: invalid values")
		return
	}
	a.gameMenu.Settings = value
	a.applyClientSettings()
}

func (a *App) saveClientSettings() {
	if !a.cfg.RememberLayout {
		return
	}
	path, err := settingsPath()
	if err == nil {
		err = os.MkdirAll(filepath.Dir(path), 0700)
	}
	if err == nil {
		var data []byte
		value := a.clientSettings()
		if a.gameMenu != nil {
			value = a.gameMenu.SavedSettings()
		}
		data, err = json.Marshal(value)
		if err == nil {
			var file *os.File
			file, err = os.CreateTemp(filepath.Dir(path), ".settings-*")
			if err == nil {
				defer os.Remove(file.Name())
				_, err = file.Write(data)
				closeErr := file.Close()
				if err == nil {
					err = closeErr
				}
				if err == nil {
					err = os.Rename(file.Name(), path)
				}
			}
		}
	}
	if err != nil {
		log.Printf("save settings: %v", err)
	}
}
