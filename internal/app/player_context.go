package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

const contextRowHeight = ui.PlayerMenuRowHeight

type playerContext struct {
	id            uint32
	x, y, held    int
	width, height int
	heldLabel     string
	open          bool
}
type contextAction struct {
	label  string
	packet []byte
}

func (a *App) contextActions(id uint32) []contextAction {
	if id == 0 || a.state == nil {
		return nil
	}
	if id != a.state.SelfID && a.state.Party.Member(id) == nil {
		if e := a.state.Entities[id]; e == nil || e.Kind != game.KindUser {
			return nil
		}
	}
	member := a.state.Party.Member(id)
	leader := a.state.Party.Leader == a.state.SelfID
	sameFaction := true
	if e := a.state.Entities[id]; e != nil && e.Shape != nil {
		if own, ok := a.selfSlot(); ok {
			sameFaction = own.Race/2 == e.Shape.Race/2
		}
	}
	var out []contextAction
	room := a.state.Party.Raid || len(a.state.Party.Members) < 7
	if sameFaction && room && member == nil && id != a.state.SelfID && (len(a.state.Party.Members) == 0 || leader || a.state.Party.Raid && a.state.Party.RaidSubLeader == a.state.SelfID) {
		if a.state.Party.Raid {
			out = append(out, contextAction{"Invite to raid", proto.RaidInvitePacket(id)})
		} else {
			out = append(out, contextAction{"Invite to party", proto.PartyInvitePacket(id)})
		}
	}
	if member != nil && leader && id != a.state.SelfID {
		if a.state.Party.Raid {
			out = append(out, contextAction{"Remove from raid", proto.RaidExpelPacket(id)}, contextAction{"Make leader", proto.RaidLeaderPacket(id)}, contextAction{"Make assistant", proto.RaidSubLeaderPacket(id)})
		} else {
			out = append(out, contextAction{"Remove from party", proto.PartyExpelPacket(id)}, contextAction{"Make leader", proto.PartyLeaderPacket(id)})
		}
	}
	if member != nil && id == a.state.SelfID {
		label := "Leave party"
		if a.state.Party.Raid {
			label = "Leave raid"
		}
		out = append(out, contextAction{label, proto.PartyLeavePacket()})
	}
	if id != a.state.SelfID && !a.state.Exchange.Active && a.state.Duel.Phase == game.DuelNone && a.state.GuildDuel.Phase == game.DuelNone {
		out = append(out, contextAction{"Trade", proto.ExchangeRequestPacket(id)}, contextAction{"Duel", proto.DuelRequestPacket(id)})
	}
	if id != a.state.SelfID && a.state.Guild.ID != 0 && a.state.Guild.Rank > 0 && a.state.Guild.Rank <= 4 && a.state.GuildDuel.Phase == game.DuelNone && a.state.Duel.Phase == game.DuelNone && !a.state.Exchange.Active {
		if e := a.state.Entities[id]; e != nil && e.Kind == game.KindUser && e.GuildID != 0 && e.GuildID != a.state.Guild.ID {
			out = append(out, contextAction{"Guild duel", proto.GuildDuelRequestPacket(id)})
		}
	}
	if id != a.state.SelfID && a.state.Guild.ID != 0 && a.state.Guild.Rank > 0 && a.state.Guild.Rank <= 2 {
		if e := a.state.Entities[id]; e != nil && e.GuildID == 0 {
			if _, member := a.state.Guild.Members[id]; !member {
				out = append(out, contextAction{"Invite to guild", proto.GuildInvitePacket(id)})
			}
		}
	}
	if id != a.state.SelfID && a.chatEntityName(id) != "" {
		out = append(out, contextAction{"Whisper", nil})
	}
	return out
}

func (a *App) contextLabels(actions []contextAction) []string {
	labels := make([]string, len(actions))
	for i, action := range actions {
		id := 0
		switch action.label {
		case "Invite to party", "Invite to raid":
			id = 5002
		case "Remove from party":
			id = 5003
		case "Remove from raid":
			id = 6509
		case "Make leader":
			id = 5004
			if a.state.Party.Raid {
				id = 6507
			}
		case "Make assistant":
			id = 6508
		case "Leave party", "Leave raid":
			id = 5005
		case "Trade":
			id = 5091
		case "Duel":
			id = 5008
		case "Guild duel":
			id = 5339
		case "Whisper":
			id = 5007
		}
		labels[i] = a.msg.Get(id)
		if labels[i] == "" {
			labels[i] = action.label
		}
	}
	return labels
}

func (a *App) performContextAction(id uint32, action contextAction) {
	if action.label == "Whisper" {
		name := a.chatEntityName(id)
		if name == "" || a.chatUI == nil {
			return
		}
		a.chatUI.Channel, a.chatUI.Recipient = 5, name
		a.chatUI.ModeOpen, a.chatUI.Typing = false, true
		a.chatOpen = true
		a.chatInput = ui.TextField{Max: 120}
		a.chatHistoryIndex, a.chatDraft = len(a.chatHistory), ""
		return
	}
	if action.label == "Trade" && a.state.Exchange.Active {
		a.questMessage("Finish the current trade first.")
	} else if a.world != nil && action.packet != nil {
		if err := a.world.Send(action.packet); err != nil {
			a.backToLogin("Unable to send player action.")
		}
	}
}

func (a *App) updatePlayerContext(in *platform.Input, w, h int) bool {
	c := &a.playerContext
	if c.open {
		actions := a.contextActions(c.id)
		if len(actions) == 0 || (a.state.Entities[c.id] == nil && a.state.Party.Member(c.id) == nil) {
			c.open = false
			return false
		}
		c.width, c.height = ui.PlayerMenuSize(a.font, a.contextLabels(actions))
		c.x, c.y = max(0, min(c.x, w-c.width)), max(0, min(c.y, h-c.height))
		if in.HasKey(platform.KeyEscape) {
			c.open = false
			return true
		}
		row := ui.PlayerMenuRowAt(in.MouseX, in.MouseY, c.x, c.y, c.width, len(actions))
		if in.Pressed {
			c.held = row
			c.heldLabel = ""
			if row >= 0 {
				c.heldLabel = actions[row].label
			}
			if !pointInside(in, c.x, c.y, c.width, c.height) {
				c.open = false
			}
			return true
		}
		if in.Clicked {
			if row >= 0 && row == c.held && actions[row].label == c.heldLabel {
				a.performContextAction(c.id, actions[row])
			}
			c.open = false
			return true
		}
		if in.RightPressed {
			c.open = false
			return true
		}
		return panelPointer(in, c.x, c.y, c.width, c.height)
	}
	if !in.RightPressed || a.wv == nil || a.wv.mapView.full {
		return false
	}
	if a.barCovered(in, w, h) || a.chatUI != nil && a.chatUI.Contains(in.MouseX, in.MouseY, h) {
		return false
	}
	if a.actionBar != nil && a.actionBar.SlotAt(in.MouseX, in.MouseY, w, h) >= 0 {
		return false
	}
	if a.minimap != nil {
		x, y := a.minimap.Origin(w, h)
		if pointInside(in, x, y, 196, 197) {
			return false
		}
	}
	id := uint32(0)
	hx, hy := 8, 8
	if a.hud != nil {
		hx, hy = a.hud.Origin(w, h)
	}
	if pointInside(in, hx, hy, 213, 114) && a.state.Party.Member(a.state.SelfID) != nil {
		id = a.state.SelfID
	}
	tx, ty := 256, 0
	if a.hud != nil {
		tx, ty = a.hud.TargetOrigin(w, h)
	}
	if pointInside(in, tx, ty, 256, 64) {
		if a.state.Target == a.state.SelfID {
			id = a.state.SelfID
		} else if e := a.state.Entities[a.state.Target]; e != nil && e.Kind == game.KindUser {
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
	if id == 0 && !a.barCovered(in, w, h) && in.MouseY < h-64 {
		// Use the same model/depth picking as left-click selection. World
		// right-click must not target a player through another actor or wall.
		pointer := *in
		pointer.Clicked = true
		picked := a.pickMob(&pointer, w, h)
		if e := a.state.Entities[picked]; e != nil && e.Kind == game.KindUser {
			id = picked
		}
	}
	actions := a.contextActions(id)
	if len(actions) == 0 {
		return false
	}
	width, height := ui.PlayerMenuSize(a.font, a.contextLabels(actions))
	*c = playerContext{id: id, x: max(0, min(in.MouseX, w-width)), y: max(0, min(in.MouseY, h-height)), width: width, height: height, held: -1, open: true}
	return true
}
func (a *App) drawPlayerContext(w, h int) {
	c := &a.playerContext
	if !c.open || a.state == nil || a.playerMenuUI == nil {
		return
	}
	actions := a.contextActions(c.id)
	labels := a.contextLabels(actions)
	c.width, c.height = ui.PlayerMenuSize(a.font, labels)
	c.x, c.y = max(0, min(c.x, w-c.width)), max(0, min(c.y, h-c.height))
	hover := ui.PlayerMenuRowAt(a.lastMouseX, a.lastMouseY, c.x, c.y, c.width, len(actions))
	a.clearTooltipUnderPanel(c.x, c.y, c.width, c.height)
	a.playerMenuUI.Draw(a.sprites, a.font, a.white, c.x, c.y, c.width, labels, hover)
}
