package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

const contextWidth, contextRowHeight = 164, 24

type playerContext struct {
	id         uint32
	x, y, held int
	heldLabel  string
	open       bool
}
type contextAction struct {
	label  string
	packet []byte
}

func (a *App) contextActions(id uint32) []contextAction {
	if id == 0 || a.state == nil {
		return nil
	}
	member := a.state.Party.Member(id)
	leader := a.state.Party.Leader == a.state.SelfID
	var out []contextAction
	if member == nil && id != a.state.SelfID && (len(a.state.Party.Members) == 0 || leader || a.state.Party.Raid && a.state.Party.RaidSubLeader == a.state.SelfID) {
		if a.state.Party.Raid {
			out = append(out, contextAction{"Invite to raid", proto.RaidInvitePacket(id)})
		} else {
			out = append(out, contextAction{"Invite to party", proto.PartyInvitePacket(id)})
		}
	}
	if member != nil && leader && id != a.state.SelfID {
		if a.state.Party.Raid {
			out = append(out, contextAction{"Make leader", proto.RaidLeaderPacket(id)}, contextAction{"Make assistant", proto.RaidSubLeaderPacket(id)}, contextAction{"Remove from raid", proto.RaidExpelPacket(id)})
		} else {
			out = append(out, contextAction{"Make leader", proto.PartyLeaderPacket(id)}, contextAction{"Remove from party", proto.PartyExpelPacket(id)})
		}
	}
	if member != nil {
		label := "Leave party"
		if a.state.Party.Raid {
			label = "Leave raid"
		}
		out = append(out, contextAction{label, proto.PartyLeavePacket()})
	}
	if id != a.state.SelfID && a.state.Duel.Phase == game.DuelNone && a.state.GuildDuel.Phase == game.DuelNone {
		out = append(out, contextAction{"Trade", proto.ExchangeRequestPacket(id)})
		if a.state.Duel.Phase == game.DuelNone && !a.state.Exchange.Active {
			out = append(out, contextAction{"Duel", proto.DuelRequestPacket(id)})
		}
	}
	if id != a.state.SelfID && a.state.Guild.ID != 0 && a.state.Guild.Rank > 0 && a.state.Guild.Rank <= 4 && a.state.GuildDuel.Phase == game.DuelNone && a.state.Duel.Phase == game.DuelNone && !a.state.Exchange.Active {
		if e := a.state.Entities[id]; e != nil && e.Kind == game.KindUser && e.GuildID != 0 && e.GuildID != a.state.Guild.ID {
			out = append(out, contextAction{"Guild duel", proto.GuildDuelRequestPacket(id)})
		}
	}
	if id != a.state.SelfID && a.state.Guild.ID != 0 && a.state.Guild.Rank > 0 && a.state.Guild.Rank <= 2 {
		if _, member := a.state.Guild.Members[id]; !member {
			out = append(out, contextAction{"Invite to guild", proto.GuildInvitePacket(id)})
		}
	}
	return out
}
func (a *App) updatePlayerContext(in *platform.Input, w, h int) bool {
	c := &a.playerContext
	if c.open {
		actions := a.contextActions(c.id)
		if len(actions) == 0 || (a.state.Entities[c.id] == nil && a.state.Party.Member(c.id) == nil) {
			c.open = false
			return false
		}
		if in.HasKey(platform.KeyEscape) {
			c.open = false
			return true
		}
		row := -1
		if in.MouseX >= c.x && in.MouseX < c.x+contextWidth && in.MouseY >= c.y && in.MouseY < c.y+len(actions)*contextRowHeight {
			row = (in.MouseY - c.y) / contextRowHeight
		}
		if in.Pressed {
			c.held = row
			c.heldLabel = ""
			if row >= 0 {
				c.heldLabel = actions[row].label
			}
			if row < 0 {
				c.open = false
			}
			return true
		}
		if in.Clicked {
			if row >= 0 && row == c.held && actions[row].label == c.heldLabel {
				action := actions[row]
				if action.label == "Trade" && a.state.Exchange.Active {
					a.questMessage("Finish the current trade first.")
				} else if err := a.world.Send(action.packet); err != nil {
					a.backToLogin("Unable to send player action.")
				}
			}
			c.open = false
			return true
		}
		if in.RightPressed {
			c.open = false
			return true
		}
		return row >= 0 && (in.MouseDown || in.RightDown || in.Wheel != 0)
	}
	if !in.RightPressed || a.wv == nil || a.wv.mapView.full {
		return false
	}
	id := uint32(0)
	if in.MouseX >= 256 && in.MouseX < 512 && in.MouseY >= 0 && in.MouseY < 64 {
		if e := a.state.Entities[a.state.Target]; e != nil && e.Kind == game.KindUser {
			id = e.ID
		}
	}
	if id == 0 && a.partyUI != nil && len(a.state.Party.Members) > 0 {
		if a.state.Party.Raid && !a.party.raidHidden {
			id = a.partyUI.RaidMemberAt(w, h, in.MouseX, in.MouseY, a.state.Party.RaidSlots)
		} else if a.state.Party.Raid {
			id = a.partyUI.RaidCompactMemberAt(w, h, in.MouseX, in.MouseY, a.state.SelfID, a.state.Party.RaidSlots)
		} else if !a.state.Party.Raid {
			x, y := ui.PartyOrigin(w, h, len(a.state.Party.Members))
			if in.MouseX >= x && in.MouseX < x+ui.PartyWidth && in.MouseY >= y && in.MouseY < y+len(a.state.Party.Members)*ui.PartyRowHeight {
				id = a.state.Party.Members[(in.MouseY-y)/ui.PartyRowHeight].ID
			}
		}
	}
	actions := a.contextActions(id)
	if len(actions) == 0 {
		return false
	}
	*c = playerContext{id: id, x: max(0, min(in.MouseX, w-contextWidth)), y: max(0, min(in.MouseY, h-len(actions)*contextRowHeight)), held: -1, open: true}
	return true
}
func (a *App) drawPlayerContext(w, h int) {
	c := &a.playerContext
	if !c.open || a.state == nil {
		return
	}
	actions := a.contextActions(c.id)
	height := len(actions) * contextRowHeight
	a.sprites.Draw(a.white, float32(c.x), float32(c.y), contextWidth, float32(height), 0, 0, 1, 1, [4]float32{.06, .05, .04, .96})
	for i, action := range actions {
		a.font.Draw(a.sprites, action.label, c.x+8, c.y+5+i*contextRowHeight, white, contextWidth-16)
	}
}
