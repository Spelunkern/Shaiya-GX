package app

import (
	"fmt"
	"math"

	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

type deathView struct {
	initialized, rune, party, ultimate, choosingLocation, waiting, pending bool
	waitUntil                                                              float64
}

func (a *App) hasResurrectionRune() bool {
	for key, item := range a.state.Inventory {
		if key[0] == 0 || key[1] >= 24 || item.Count == 0 {
			continue
		}
		// Rapture's GetItemRType groups types 25, 44 and 100 as family 26.
		if item.Type != 25 && item.Type != 44 && item.Type != 100 {
			continue
		}
		if d := a.itemsItem(item.Type, item.TypeID); d != nil && d.Special == 37 {
			return true
		}
	}
	return false
}
func (a *App) initDeath() {
	d := &a.wv.death
	if d.initialized {
		return
	}
	d.initialized = true
	a.deathUI.Reset()
	if own, ok := a.selfSlot(); ok {
		d.ultimate = own.Mode == 3
	}
	// Rapture marks duel defeats as loss-free and ignores resurrection runes.
	// Its duel result precedes the character death packet.
	duelDeath := a.state.DeathLossFree
	if duelDeath {
		d.ultimate = false
	}
	d.party = a.state.Party.Leader != 0 && a.state.Party.Leader != a.state.SelfID
	// Original 0x4bef30 skips rune choice on the two special PvP maps.
	if !duelDeath && a.state.MapID != 105 && a.state.MapID != 108 {
		d.rune = a.hasResurrectionRune()
	}
}
func (a *App) deathText(id int, fallback string) string {
	if s := a.msg[id]; s != "" {
		return s
	}
	return fallback
}
func (a *App) configureDeath() {
	d := &a.wv.death
	kind := ui.DeathSingle
	msg := a.deathText(10067, "You will return to the nearest town automatically.")
	if d.waiting {
		kind = ui.DeathWaiting
		msg = fmt.Sprintf("%s (%d)", a.deathText(10068, "Resurrection at the party leader's location."), max(0, int(math.Ceil(d.waitUntil-a.clock))))
	} else if d.pending {
		kind = ui.DeathWaiting
		msg = "Waiting for resurrection..."
	} else if d.choosingLocation {
		kind = ui.DeathLocation
		msg = a.deathText(809, "Where would you like to resurrect?")
	} else if d.rune {
		msg = a.deathText(800, "Resurrect using an item.")
		if d.party && !d.ultimate {
			kind = ui.DeathParty
		}
	} else if d.ultimate {
		kind = ui.DeathUltimate
		msg = a.deathText(10066, "Your character may be deleted if resurrected without protection.")
	} else if d.party {
		kind = ui.DeathParty
	}
	a.deathUI.Set(kind, msg)
	a.deathUI.LeftLabel = a.deathText(810, "Bind point")
	a.deathUI.RightLabel = a.deathText(811, "Here")
}
func (a *App) requestDeathRevival(mode proto.ReviveMode) {
	if a.world == nil || a.clock < a.wv.reviveAt {
		return
	}
	if err := a.world.Send(proto.ReviveChoicePacket(mode)); err != nil {
		a.backToLogin("Unable to request resurrection.")
		return
	}
	a.wv.reviveAt = a.clock + 3
	if mode == proto.ReviveParty {
		a.wv.death.waiting = true
		a.wv.death.waitUntil = a.clock + 30
	} else {
		a.wv.death.pending = true
	}
}

// deathChoice mirrors the two-stage native dialog; the server decides the
// actual resurrection outcome and whether a rune is available.
func deathChoice(d *deathView, choice int) (proto.ReviveMode, bool) {
	if choice == 0 || d.waiting || d.pending {
		return 0, false
	}
	if d.choosingLocation {
		if choice == 1 {
			return proto.ReviveRuneBind, true
		}
		if choice == 2 {
			return proto.ReviveRuneHere, true
		}
		return 0, false
	}
	if choice == 2 && d.party && !d.ultimate {
		return proto.ReviveParty, true
	}
	if choice != 1 {
		return 0, false
	}
	if d.rune {
		if d.ultimate {
			return proto.ReviveRuneBind, true
		}
		d.choosingLocation = true
		return 0, false
	}
	return proto.ReviveBind, true
}
func (a *App) updateDeath(in *platform.Input, w, h int) {
	if a.wv == nil || a.state == nil || a.deathUI == nil {
		return
	}
	a.initDeath()
	if a.wv.death.pending && a.clock >= a.wv.reviveAt {
		a.wv.death.pending = false
	}
	a.configureDeath()
	d := &a.wv.death
	choice := a.deathUI.Update(in, w, h)
	if !d.waiting && !d.pending && !d.ultimate && !d.rune {
		for _, r := range in.Chars {
			if r == 'r' || r == 'R' {
				a.requestDeathRevival(proto.ReviveBind)
				return
			}
		}
	}
	if mode, send := deathChoice(d, choice); send {
		a.requestDeathRevival(mode)
	}
}
func (a *App) drawDeath(w, h int) {
	if a.wv == nil || a.deathUI == nil {
		return
	}
	a.initDeath()
	a.configureDeath()
	a.deathUI.Draw(a.sprites, a.font, w, h)
}
