package app

import (
	"fmt"
	"math"
	"strings"

	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

type guildDuelView struct {
	requestSerial               uint64
	inviteUntil, countdownUntil float64
}

func (a *App) confirmedGuildDuel(p []byte) {
	if a.state == nil {
		return
	}
	d, v := &a.state.GuildDuel, &a.guildDuel
	switch proto.Op(p) {
	case proto.GuildDuelRequest:
		if d.Phase == game.DuelIncoming {
			v.inviteUntil = a.clock + 10
		}
	case proto.GuildDuelAnswer:
		if d.Response.Result != 1 {
			if a.guildDuelMessage != nil {
				a.guildDuelMessage.Visible = false
			}
			a.questMessage("Guild duel declined or unavailable.")
		}
	case proto.GuildDuelCountdown:
		v.countdownUntil = a.clock + 5
		if a.guildDuelMessage != nil {
			a.guildDuelMessage.Visible = false
		}
	case proto.GuildDuelStart:
		a.questMessage("Guild duel started.")
	case proto.GuildDuelLeave:
		if d.LastLeave.Player == a.state.SelfID {
			a.questMessage("You left the guild duel.")
		}
	case proto.GuildDuelResult:
		if d.Outcome == 1 {
			a.questMessage("Your guild won the duel.")
		} else if d.Outcome == 2 {
			a.questMessage("Your guild lost the duel.")
		}
	}
}

func (a *App) updateGuildDuelInvite(in *platform.Input, w, h int) bool {
	if a.state == nil || a.guildDuelMessage == nil {
		return false
	}
	d, v := &a.state.GuildDuel, &a.guildDuel
	if d.RequestSerial != v.requestSerial {
		v.requestSerial = d.RequestSerial
		if d.Phase == game.DuelIncoming {
			name := fmt.Sprintf("Player %d", d.Opponent)
			if e := a.state.Entities[d.Opponent]; e != nil && e.Shape != nil && e.Shape.Name != "" {
				name = e.Shape.Name
			}
			a.guildDuelMessage.Open(name+" challenges your guild. Accept?", true)
			if v.inviteUntil == 0 {
				v.inviteUntil = a.clock + 10
			}
		}
	}
	if !a.guildDuelMessage.Visible {
		return false
	}
	if d.Phase != game.DuelIncoming || a.clock >= v.inviteUntil {
		a.guildDuelMessage.Visible = false
		return false
	}
	if result := a.guildDuelMessage.Update(in, w, h); result != ui.MessageNone {
		accept := result == ui.MessageOK
		if accept && (a.state.Duel.Phase != game.DuelNone || a.state.Exchange.Active) {
			accept = false
			a.questMessage("Finish the current trade or duel first.")
		}
		if err := a.world.Send(proto.GuildDuelAnswerPacket(accept)); err != nil {
			a.backToLogin("Unable to answer guild duel.")
			return true
		}
		if !accept {
			d.Phase = game.DuelNone
		}
	}
	return true
}

func (a *App) guildDuelCommand(text string) bool {
	fields := strings.Fields(text)
	if len(fields) == 0 || !strings.EqualFold(fields[0], "/guildduel") {
		return false
	}
	e := a.state.Entities[a.state.Target]
	if len(fields) > 1 {
		e = nil
		name := strings.Join(fields[1:], " ")
		for _, candidate := range a.state.Entities {
			if candidate.Kind == game.KindUser && candidate.Shape != nil && strings.EqualFold(candidate.Shape.Name, name) {
				e = candidate
				break
			}
		}
	}
	if a.state.Guild.ID == 0 || a.state.Guild.Rank == 0 || a.state.Guild.Rank > 4 || a.state.GuildDuel.Phase != game.DuelNone || a.state.Duel.Phase != game.DuelNone || a.state.Exchange.Active || e == nil || e.Kind != game.KindUser || e.ID == a.state.SelfID || e.GuildID == 0 || e.GuildID == a.state.Guild.ID {
		a.questMessage("Select a member of another guild; rank 1-4 is required.")
		return true
	}
	if err := a.world.Send(proto.GuildDuelRequestPacket(e.ID)); err != nil {
		a.backToLogin("Unable to request guild duel.")
	}
	return true
}

func (a *App) drawGuildDuel(w, h int) {
	if a.state == nil || a.state.GuildDuel.Phase != game.DuelCounting || !a.state.GuildDuel.Active {
		return
	}
	seconds := max(0, int(math.Ceil(a.guildDuel.countdownUntil-a.clock)))
	a.font.Draw(a.sprites, fmt.Sprintf("Guild duel begins in %d", seconds), w/2-75, 120, white, 200)
}
