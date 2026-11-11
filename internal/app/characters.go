package app

import (
	"fmt"

	"github.com/Spelunkern/shaiya-go-client/internal/game"
	"github.com/Spelunkern/shaiya-go-client/internal/platform"
	"github.com/Spelunkern/shaiya-go-client/internal/proto"
	"github.com/Spelunkern/shaiya-go-client/internal/ui"
)

var (
	white = [4]float32{1, 1, 1, 1}
	gray  = [4]float32{.8, .8, .8, 1}
)

// charNotice is a provisional message line until the original dialogs
// (message boxes) are ported.
type charNotice struct {
	text      string
	until     float64
	deleteArm float64
}

func (a *App) updateCharacters(in *platform.Input, w, h int, dt float64) {
	if a.state == nil {
		return
	}
	if a.world == nil { // offscreen preview
		a.chars.Update(in, w, h, dt)
		return
	}
	a.clock += dt
	switch a.state.Phase {
	case game.PhaseCharSelect:
		a.screen = screenCharacters
		a.chars.Light = a.state.Faction.Faction == 0
		a.chars.SetChars(a.state.Slots)
		a.chars.Busy = false
		switch a.chars.Update(in, w, h, dt) {
		case ui.CharStart:
			if ch := a.chars.Chars[a.chars.Selected]; ch != nil {
				a.chars.Busy = true
				a.world.Send(proto.SelectCharacterPacket(ch.ID))
			}
		case ui.CharDelete:
			ch := a.chars.Chars[a.chars.Selected]
			if ch == nil {
				break
			}
			if a.clock < a.notice.deleteArm {
				a.world.Send(proto.DeleteCharacterPacket(ch.ID))
				a.notice = charNotice{text: "Deleting " + ch.Name + "...", until: a.clock + 3}
			} else {
				a.notice = charNotice{text: "Press Delete again to delete " + ch.Name + ".", until: a.clock + 3, deleteArm: a.clock + 3}
			}
		case ui.CharCreate:
			a.notice = charNotice{text: "Character creation is not available yet.", until: a.clock + 3}
		case ui.CharExit:
			a.backToLogin("")
		}
	case game.PhaseFaction:
		// PROVISIONAL: the faction screen is not ported.
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
		if a.screen != screenWorld || a.wv == nil || a.wv.mapID != a.state.MapID {
			a.screen = screenWorld
			a.enterWorld()
		}
		a.updateWorld(in, dt)
		if in.HasKey(platform.KeyEscape) {
			a.world.Send(proto.LogoutPacket())
			a.wv = nil
		}
	}
}

func (a *App) drawCharacters(w, h int) {
	if a.state == nil {
		return
	}
	y := 40
	line := func(s string, c [4]float32) {
		a.font.Draw(a.sprites, s, 40, y, c, 0)
		y += 18
	}
	switch a.state.Phase {
	case game.PhaseFaction:
		line("Choose a faction (provisional): L = Light, F = Fury", white)
	case game.PhaseCharSelect, game.PhaseLoading:
		a.chars.Draw(a.sprites, a.font, w, h)
		if a.state.Phase == game.PhaseLoading {
			line("Loading...", white)
		}
		if a.clock < a.notice.until {
			tw := a.font.Width(a.notice.text)
			a.font.Draw(a.sprites, a.notice.text, (w-tw)/2, h/2, white, 0)
		}
	case game.PhaseInWorld:
		s := a.state
		line(fmt.Sprintf("map %d  (%.1f, %.1f, %.1f)  entities %d  [provisional HUD; Esc = character select]", s.MapID, s.Self.X, s.Self.Y, s.Self.Z, len(s.Entities)), white)
		for _, c := range s.Chat[max(0, len(s.Chat)-10):] {
			line(fmt.Sprintf("[%d%s] %s", c.SenderID, c.Sender, c.Text), gray)
		}
	}
}
