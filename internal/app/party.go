package app

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

type partyView struct {
	inviteSerial, replySerial uint64
	raidErrorSerial           uint64
	raidHidden                bool
	expires                   float64
}

func (a *App) updatePartyInvite(in *platform.Input, w, h int) bool {
	if a.partyMessage == nil || a.state == nil {
		return false
	}
	p := &a.state.Party
	if p.InviteSerial != a.party.inviteSerial {
		a.party.inviteSerial = p.InviteSerial
		if p.InviteSerial == 0 {
			a.partyMessage.Visible = false
			return false
		}
		name := p.InviteName
		if name == "" {
			name = fmt.Sprintf("Player %d", p.InviteID)
		}
		kind := "party"
		if p.RaidInvite {
			kind = "raid"
		}
		a.partyMessage.Open(fmt.Sprintf("%s invites you to a %s. Join?", name, kind), true)
		a.party.expires = a.clock + 30 // Rapture's invite window; expires only the prompt.
	}
	if p.RaidErrorSerial != a.party.raidErrorSerial {
		a.party.raidErrorSerial = p.RaidErrorSerial
		messages := map[uint32]string{0: "Raid invitation declined.", 1: "Player is busy.", 2: "Only the raid leader or assistant can invite.", 3: "Player is already in a party.", 4: "Player is unavailable.", 5: "Player is already in this raid.", 12: "Raid not found.", 13: "Raid does not allow auto-join.", 14: "Raid is full."}
		text := messages[p.RaidError]
		if text == "" {
			text = "Raid request refused."
		}
		a.combatNotice, a.combatNoticeUntil = text, a.clock+4
	}
	if p.ReplySerial != a.party.replySerial {
		a.party.replySerial = p.ReplySerial
		messages := [...]string{"Party invitation declined.", "Player is busy.", "Only the party leader can invite.", "Player is already in a party.", "Player is unavailable.", "Player is already in this party."}
		text := "Party request refused."
		if int(p.Reply.Result) < len(messages) {
			text = messages[p.Reply.Result]
		}
		a.combatNotice, a.combatNoticeUntil = text, a.clock+4
	}
	if !a.partyMessage.Visible {
		return false
	}
	if len(p.Members) > 0 || a.clock >= a.party.expires {
		a.partyMessage.Visible = false
		return false
	}
	answer := a.partyMessage.Update(in, w, h)
	if answer != ui.MessageNone {
		packet := proto.PartyAnswerPacket(answer == ui.MessageOK, a.state.SelfID)
		if p.RaidInvite {
			packet = proto.RaidAnswerPacket(answer == ui.MessageOK, a.state.SelfID)
		}
		if err := a.world.Send(packet); err != nil {
			a.backToLogin("Unable to answer party invitation.")
		}
	}
	return true
}
func (a *App) updatePartyPointer(in *platform.Input, w, h int) bool {
	if a.partyUI == nil || len(a.state.Party.Members) == 0 {
		return false
	}
	var id uint32
	var captured bool
	if a.state.Party.Raid {
		if a.party.raidHidden {
			id, captured = a.partyUI.UpdateRaidCompact(in, w, h, a.state.SelfID, a.state.Party.RaidSlots)
			if id != 0 {
				a.state.Target = id
			}
			return captured
		}
		a.partyUI.SetRaidSettings(a.state.Party.RaidDivision, a.state.Party.RaidAutoJoin)
		var action ui.RaidAction
		id, action, captured = a.partyUI.UpdateRaid(in, w, h, a.state.Party.RaidSlots, a.state.Party.Leader == a.state.SelfID)
		var packet []byte
		switch action {
		case ui.RaidClose:
			a.party.raidHidden = true
			a.partyUI.ResetRaidSettings()
		case ui.RaidApply:
			division, auto := a.partyUI.RaidSettings()
			a.partyUI.ApplyRaidViewSettings()
			if a.state.Party.Leader == a.state.SelfID {
				if division != a.state.Party.RaidDivision {
					packet = proto.RaidDivisionPacket(division)
				}
				if auto != a.state.Party.RaidAutoJoin {
					if err := a.world.Send(proto.RaidAutoJoinPacket(auto)); err != nil {
						a.backToLogin("Unable to update raid settings.")
						return true
					}
				}
			}
			a.party.raidHidden = true
			a.partyUI.ResetRaidSettings()
		}
		if packet != nil {
			if err := a.world.Send(packet); err != nil {
				a.backToLogin("Unable to update raid settings.")
			}
		}
	} else {
		id, captured = a.partyUI.Update(in, w, h, a.state.Party.Members)
	}
	if id != 0 {
		a.state.Target = id
	}
	return captured
}
func (a *App) drawParty(w, h int) {
	if a.partyUI == nil {
		return
	}
	// Self resources are delivered separately from remote party pool updates.
	members := append([]proto.PartyMember(nil), a.state.Party.Members...)
	for i := range members {
		if members[i].ID == a.state.SelfID {
			members[i] = a.state.SelfPartyMember()
		}
	}
	if a.state.Party.Raid {
		if !a.party.raidHidden {
			a.partyUI.SetRaidSettings(a.state.Party.RaidDivision, a.state.Party.RaidAutoJoin)
			a.partyUI.DrawRaid(a.sprites, a.font, w, h, members, a.state.Party.RaidSlots, a.state.Party.Leader, a.state.Target)
		} else {
			a.partyUI.DrawRaidCompact(a.sprites, a.font, w, h, a.state.SelfID, members, a.state.Party.RaidSlots, a.state.Party.Leader, a.state.Target)
		}
	} else {
		a.partyUI.Draw(a.sprites, a.font, w, h, members, a.state.Party.Leader, a.state.Target)
	}
}

// Slash commands share the same native requests as the player-context menu.
// They never add/remove members before server confirmation.
func (a *App) partyCommand(text string) bool {
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return false
	}
	command := strings.ToLower(fields[0])
	if strings.HasPrefix(command, "/raid") {
		return a.raidCommand(fields)
	}
	if command != "/invite" && command != "/leave" && command != "/kick" && command != "/leader" {
		return false
	}
	name := ""
	if len(fields) > 1 {
		name = strings.Join(fields[1:], " ")
	}
	var packet []byte
	switch command {
	case "/invite":
		if a.state.Party.Raid && name != "" {
			packet = proto.RaidNamePacket(proto.RaidInviteName, name)
		} else if a.state.Party.Raid && a.state.Target != 0 {
			packet = proto.RaidInvitePacket(a.state.Target)
		} else if name != "" {
			packet = proto.PartyInviteNamePacket(name)
		} else if a.state.Target != 0 {
			packet = proto.PartyInvitePacket(a.state.Target)
		}
	case "/leave":
		packet = proto.PartyLeavePacket()
	case "/kick", "/leader":
		id := a.state.Target
		if name != "" {
			id = 0
			for _, m := range a.state.Party.Members {
				if strings.EqualFold(m.Name, name) {
					id = m.ID
					break
				}
			}
		}
		if a.state.Party.Leader != a.state.SelfID || id == 0 || id == a.state.SelfID || a.state.Party.Member(id) == nil {
			a.combatNotice, a.combatNoticeUntil = "Select a party member; only the leader can do this.", a.clock+4
			return true
		}
		if a.state.Party.Raid && command == "/kick" {
			packet = proto.RaidExpelPacket(id)
		} else if a.state.Party.Raid {
			packet = proto.RaidLeaderPacket(id)
		} else if command == "/kick" {
			packet = proto.PartyExpelPacket(id)
		} else {
			packet = proto.PartyLeaderPacket(id)
		}
	}
	if packet == nil {
		a.combatNotice, a.combatNoticeUntil = "Use /invite character name, or select a player.", a.clock+4
		return true
	}
	if err := a.world.Send(packet); err != nil {
		a.backToLogin("Unable to send party request.")
	}
	return true
}

// Raid commands only send requests. Membership and settings change on ACK.
func (a *App) raidCommand(fields []string) bool {
	p := &a.state.Party
	command := strings.ToLower(fields[0])
	arg := strings.Join(fields[1:], " ")
	var packet []byte
	switch command {
	case "/raidwindow":
		if p.Raid {
			a.party.raidHidden = !a.party.raidHidden
			return true
		}
	case "/raid":
		if len(p.Members) > 1 && !p.Raid && p.Leader == a.state.SelfID {
			packet = proto.RaidConvertPacket(1, false, 1)
		}
	case "/raidinvite":
		if arg != "" {
			packet = proto.RaidNamePacket(proto.RaidInviteName, arg)
		} else if a.state.Target != 0 {
			packet = proto.RaidInvitePacket(a.state.Target)
		}
	case "/raidjoin":
		if arg != "" {
			packet = proto.RaidNamePacket(proto.RaidAutoJoinName, arg)
		}
	case "/raidauto":
		if p.Raid && p.Leader == a.state.SelfID && (arg == "on" || arg == "off") {
			packet = proto.RaidAutoJoinPacket(arg == "on")
		}
	case "/raiddivision":
		if n, err := strconv.ParseUint(arg, 10, 32); err == nil && p.Raid && p.Leader == a.state.SelfID {
			packet = proto.RaidDivisionPacket(uint32(n))
		}
	case "/raidsub":
		id := a.state.Target
		if arg != "" {
			id = 0
			for _, m := range p.Members {
				if strings.EqualFold(m.Name, arg) {
					id = m.ID
					break
				}
			}
		}
		if p.Raid && p.Leader == a.state.SelfID && id != a.state.SelfID && p.Member(id) != nil {
			packet = proto.RaidSubLeaderPacket(id)
		}
	case "/raidswap":
		if len(fields) == 3 && p.Raid && (p.Leader == a.state.SelfID || p.RaidSubLeader == a.state.SelfID) {
			first, e1 := strconv.ParseUint(fields[1], 10, 8)
			second, e2 := strconv.ParseUint(fields[2], 10, 8)
			if e1 == nil && e2 == nil && first < 30 && second < 30 {
				packet = proto.RaidSwapPacket(uint32(first), uint32(second))
			}
		}
	case "/raiddisband":
		if p.Raid && p.Leader == a.state.SelfID {
			packet = proto.RaidDisbandPacket()
		}
	default:
		return false
	}
	if packet == nil {
		a.combatNotice, a.combatNoticeUntil = "Raid command unavailable or invalid.", a.clock+4
		return true
	}
	if err := a.world.Send(packet); err != nil {
		a.backToLogin("Unable to send raid request.")
	}
	return true
}
