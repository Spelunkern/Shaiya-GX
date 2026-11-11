package app

import (
	"fmt"
	"sort"

	"github.com/Spelunkern/shaiya-go-client/internal/game"
	"github.com/Spelunkern/shaiya-go-client/internal/platform"
	"github.com/Spelunkern/shaiya-go-client/internal/proto"
)

// PROVISIONAL: character select (0x46ce30) and the world are not ported
// yet; this screen lists the characters and plays the protocol flow.
var gray = [4]float32{.8, .8, .8, 1}

func (a *App) slots() []proto.CharacterSlot {
	var out []proto.CharacterSlot
	if a.state == nil {
		return out
	}
	for _, c := range a.state.Slots {
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Slot < out[j].Slot })
	return out
}

func (a *App) updateCharacters(in *platform.Input) {
	if a.state == nil || a.world == nil {
		return
	}
	switch a.state.Phase {
	case game.PhaseCharSelect:
		a.screen = screenCharacters
		list := a.slots()
		for _, r := range in.Chars {
			if r >= '1' && r <= '5' && int(r-'1') < len(list) {
				a.world.Send(proto.SelectCharacterPacket(list[r-'1'].ID))
			}
		}
	case game.PhaseFaction:
		for _, r := range in.Chars {
			if r == 'l' || r == 'f' {
				f := uint8(0)
				if r == 'f' {
					f = 1
				}
				a.world.Send(proto.FactionPacket(f))
			}
		}
	case game.PhaseInWorld:
		a.screen = screenWorld
		if in.HasKey(platform.KeyEscape) {
			a.world.Send(proto.LogoutPacket())
		}
	}
}

func (a *App) drawCharacters(w, h int) {
	y := 40
	line := func(s string, c [4]float32) {
		a.font.Draw(a.sprites, s, 40, y, c, 0)
		y += 18
	}
	if a.state == nil {
		return
	}
	switch a.state.Phase {
	case game.PhaseFaction:
		line("Choose a faction: L = Light, F = Fury", white)
	case game.PhaseCharSelect:
		line("Character select (provisional): press 1-5", white)
		for i, c := range a.slots() {
			line(fmt.Sprintf("%d. %s  Lv %d", i+1, c.Name, c.Level), gray)
		}
	case game.PhaseLoading:
		line("Loading...", white)
	case game.PhaseInWorld:
		s := a.state
		line(fmt.Sprintf("In world (provisional): map %d  (%.1f, %.1f, %.1f)", s.MapID, s.Self.X, s.Self.Y, s.Self.Z), white)
		line(fmt.Sprintf("Entities in view: %d   Esc = back to character select", len(s.Entities)), gray)
		for _, c := range s.Chat[max(0, len(s.Chat)-10):] {
			line(fmt.Sprintf("[%d%s] %s", c.SenderID, c.Sender, c.Text), gray)
		}
	}
}

var white = [4]float32{1, 1, 1, 1}
