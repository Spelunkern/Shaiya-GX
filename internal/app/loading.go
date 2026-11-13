package app

import (
	"fmt"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
)

type mapLoading struct {
	active, presented, started, drawn bool
	selection                         bool
	mapID                             uint16
	progress                          float32
}

// Present the overlay before synchronous map setup, then keep it up until
// the initial visible cells/resources are ready. Network packets still pump.
func (a *App) updateMapLoading() bool {
	if a.world == nil || a.state == nil {
		return false
	}
	phase := a.state.Phase
	if phase == game.PhaseCharSelect && a.screen == screenCharacters && !a.creating {
		fury := a.state.Faction.Faction == 1
		key := "select_a"
		if fury {
			key = "select_b"
		}
		if _, loaded := a.worlds[key]; !loaded {
			if !a.loading.active || !a.loading.selection {
				a.loading = mapLoading{active: true, selection: true}
			}
			if !a.loading.presented {
				return true
			}
			if a.selectWorld(fury) == nil {
				a.backToLogin("Unable to load character scenery.")
				return false
			}
			a.loading = mapLoading{}
			return false
		}
	}
	if phase != game.PhaseLoading && phase != game.PhaseInWorld {
		a.loading = mapLoading{}
		return false
	}
	needsWorld := phase == game.PhaseInWorld && (a.screen != screenWorld || a.wv == nil || a.wv.mapID != a.state.MapID)
	if !a.loading.active && (phase == game.PhaseLoading || needsWorld) {
		a.loading = mapLoading{active: true, mapID: a.state.MapID}
		a.menuOpen = false
	}
	// A transfer received during initial streaming supersedes the old plan.
	if a.loading.active && a.loading.started && a.wv != nil && a.wv.mapID != a.state.MapID {
		a.loading = mapLoading{active: true, mapID: a.state.MapID}
	}
	l := &a.loading
	if !l.active {
		return false
	}
	l.mapID = a.state.MapID
	if phase == game.PhaseLoading || !l.presented {
		return true
	}
	if !l.started {
		a.screen = screenWorld
		a.enterWorld()
		if a.wv == nil || a.wv.world == nil {
			a.backToLogin("Unable to load map resources.")
			return false
		}
		l.started = true
		l.progress = .1
	}
	if l.drawn && a.wv != nil && a.wv.world != nil && a.wv.world.Ready() {
		a.loading = mapLoading{}
		return false
	}
	return true
}
func (a *App) drawMapLoading(w, h int) {
	l := &a.loading
	status := "Connecting to the map..."
	if l.selection {
		status = "Loading character selection..."
	}
	if l.started && a.wv != nil && a.wv.world != nil {
		l.progress = max(l.progress, .1+.9*a.wv.world.LoadProgress())
		status = fmt.Sprintf("Loading map %d... %d%%", l.mapID, int(l.progress*100))
		l.drawn = true
	}
	a.sprites.Begin(w, h)
	a.loadingScreen.Draw(a.sprites, a.font, a.white, w, h, l.progress, status)
	a.sprites.Flush()
	l.presented = true
}
