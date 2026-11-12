//go:build integration

package app

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/session"
	"github.com/Spelunkern/Shaiya-GX/internal/testworld"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

func TestRaidSevenAgainstRapture(t *testing.T) {
	var accounts []struct{ User, Password string }
	if raw := os.Getenv("SHAIYA_TEST_RAID_ACCOUNTS"); raw == "" {
		t.Skip("requires seven disposable accounts")
	} else if err := json.Unmarshal([]byte(raw), &accounts); err != nil || len(accounts) != 7 {
		t.Fatal("invalid disposable raid fixture")
	}
	type player struct {
		state *game.State
		world *session.World
		app   *App
		send  func([]byte)
		wait  func(func() bool)
	}
	players := make([]player, 7)
	for i, account := range accounts {
		s, w, send, wait := testworld.EnterAs(t, account.User, account.Password)
		a := inputApp()
		a.state, a.world = s, w
		a.partyMessage, _ = ui.NewMessage(inputTextures{})
		players[i] = player{s, w, a, send, wait}
	}
	leader, assistant := &players[0], &players[1]
	leader.app.partyCommand("/invite " + assistant.state.Slots[0].Name)
	assistant.wait(func() bool { return assistant.state.Party.InviteSerial == 1 })
	assistant.send(proto.PartyAnswerPacket(true, assistant.state.SelfID))
	leader.wait(func() bool { return len(leader.state.Party.Members) == 2 })
	assistant.wait(func() bool { return len(assistant.state.Party.Members) == 2 })
	leader.app.partyCommand("/raid")
	leader.wait(func() bool { return leader.state.Party.Raid })
	assistant.wait(func() bool { return assistant.state.Party.Raid })
	leader.app.partyCommand("/raidauto on")
	leader.wait(func() bool { return leader.state.Party.RaidAutoJoin })
	assistant.wait(func() bool { return assistant.state.Party.RaidAutoJoin })
	for i := 2; i < len(players); i++ {
		joining := &players[i]
		if i == 6 {
			joining.app.partyCommand("/raidjoin " + leader.state.Slots[0].Name)
		} else {
			inviter := leader
			if i == 2 {
				inviter = assistant
			}
			inviter.app.partyCommand("/raidinvite " + joining.state.Slots[0].Name)
			joining.wait(func() bool { return joining.state.Party.InviteSerial == 1 })
			joining.app.updatePartyInvite(&platform.Input{}, 1024, 768)
			if !joining.state.Party.RaidInvite || !joining.app.partyMessage.Visible {
				t.Fatalf("member %d missing raid prompt", i)
			}
			joining.app.updatePartyInvite(&platform.Input{Keys: []platform.Key{platform.KeyEnter}}, 1024, 768)
		}
		for j := 0; j <= i; j++ {
			p := &players[j]
			p.wait(func() bool { return p.state.Party.Raid && len(p.state.Party.Members) == i+1 })
		}
	}
	for i := range players {
		p := &players[i]
		if p.state.Party.Leader != leader.state.SelfID || p.state.Party.RaidSubLeader != assistant.state.SelfID || !p.state.Party.RaidAutoJoin {
			t.Fatalf("member %d has wrong raid roles/settings", i)
		}
		for slot := uint16(0); slot < 7; slot++ {
			if p.state.Party.RaidSlots[slot] != players[slot].state.SelfID {
				t.Fatalf("member %d has incorrect slot %d", i, slot)
			}
		}
	}
	leader.app.partyUI, _ = ui.NewPartyPanel(inputTextures{})
	leader.app.party.raidHidden = true
	cx, cy := leader.app.partyUI.RaidCompactOrigin(1024, 768)
	compactClick := func(x, y int) {
		leader.app.updatePartyPointer(&platform.Input{Pressed: true, MouseDown: true, MouseX: x, MouseY: y}, 1024, 768)
		leader.app.updatePartyPointer(&platform.Input{Clicked: true, MouseX: x, MouseY: y}, 1024, 768)
	}
	compactClick(cx+49, cy+12) // second group
	compactClick(cx+8, cy+24+8)
	if leader.state.Target != players[6].state.SelfID {
		t.Fatal("compact group did not select seventh player")
	}
	leader.app.partyCommand("/kick " + players[3].state.Slots[0].Name)
	for i := range players {
		p := &players[i]
		if i == 3 {
			p.wait(func() bool { return !p.state.Party.Raid && len(p.state.Party.Members) == 0 })
		} else {
			p.wait(func() bool {
				return len(p.state.Party.Members) == 6 && p.state.Party.Member(players[3].state.SelfID) == nil
			})
		}
	}
	leader.app.partyCommand("/raiddisband")
	for i := range players {
		p := &players[i]
		p.wait(func() bool { return !p.state.Party.Raid && len(p.state.Party.Members) == 0 })
		p.send(proto.LogoutPacket())
		p.wait(func() bool { return p.state.SelfID == 0 })
	}
	t.Log("seven players: assistant invitation, auto-join, multi-chunk roster, slots, kick and disband confirmed")
}
