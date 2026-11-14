package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"log"
)

// Resolution changes commit on OK; other graphics controls keep their existing
// live preview. Cancel never changes the desktop display mode.
func (a *App) applyDisplaySettings() bool {
	if a.gameMenu == nil {
		return true
	}
	mode := platform.WindowMode(a.gameMenu.Settings.DisplayMode)
	a.gameMenu.Settings.DisplayMode = mode
	if a.displayConfigured && mode == a.appliedDisplay {
		return true
	}
	available := mode == (platform.DisplayMode{})
	for _, candidate := range a.gameMenu.DisplayModes {
		available = available || candidate == mode
	}
	if !available {
		a.gameMenu.DisplayError = "Display mode unavailable."
		a.gameMenu.Settings.DisplayMode = a.appliedDisplay
		log.Printf("display mode %s is not available", mode.Label())
		return false
	}
	if a.cfg.ApplyDisplayMode != nil {
		if err := a.cfg.ApplyDisplayMode(mode); err != nil {
			log.Printf("apply display mode: %v", err)
			a.displayConfigured = false
			a.gameMenu.DisplayError = "Display change failed."
			a.gameMenu.Settings.DisplayMode = a.appliedDisplay
			return false
		}
	} else if mode != (platform.DisplayMode{}) {
		a.gameMenu.DisplayError = "Display mode unavailable."
		a.gameMenu.Settings.DisplayMode = a.appliedDisplay
		return false
	}
	a.appliedDisplay, a.displayConfigured = mode, true
	a.gameMenu.DisplayError = ""
	return true
}
