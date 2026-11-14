package platform

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"path/filepath"
)

type windowState struct {
	Width, Height int
	Maximized     bool
}

func windowStatePath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "shaiya-go", "window.json"), nil
}

func loadWindowState() (windowState, error) {
	var state windowState
	path, err := windowStatePath()
	if err != nil {
		return state, err
	}
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return state, nil
	}
	if err != nil {
		return state, err
	}
	if err = json.Unmarshal(b, &state); err != nil {
		return windowState{}, err
	}
	if state.Width < 320 || state.Height < 240 || state.Width > 16384 || state.Height > 16384 {
		return windowState{}, errors.New("invalid saved window dimensions")
	}
	return state, nil
}

func (w *Window) saveSize() {
	if !w.rememberSize || w.normalSize.Width == 0 {
		return
	}
	state := w.normalSize
	state.Maximized = w.maximized
	path, err := windowStatePath()
	if err == nil {
		err = os.MkdirAll(filepath.Dir(path), 0o700)
	}
	if err == nil {
		var b []byte
		b, err = json.Marshal(state)
		if err == nil {
			err = os.WriteFile(path, b, 0o600)
		}
	}
	if err != nil {
		log.Printf("save window size: %v", err)
	}
}
