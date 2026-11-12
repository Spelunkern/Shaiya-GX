//go:build integration

package app

import (
	"os"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/testworld"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

func TestRaidAgainstRapture(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_PARTNER") == "" {
		t.Skip("requires two disposable accounts")
	}
	s1, w1, send1, wait1 := testworld.Enter(t)
	s2, w2, send2, wait2 := testworld.EnterAs(t, os.Getenv("SHAIYA_TEST_PARTNER"), os.Getenv("SHAIYA_TEST_PARTNER_PASSWORD"))
	a1, a2 := inputApp(), inputApp()
	a1.state, a1.world, a2.state, a2.world = s1, w1, s2, w2
	a1.partyUI, _ = ui.NewPartyPanel(inputTextures{})
	a2.partyMessage, _ = ui.NewMessage(inputTextures{})
	a1.partyCommand("/invite " + s2.Slots[0].Name)
	wait2(func() bool { return s2.Party.InviteSerial > 0 })
	send2(proto.PartyAnswerPacket(true, s2.SelfID))
	wait1(func() bool { return len(s1.Party.Members) == 2 })
	wait2(func() bool { return len(s2.Party.Members) == 2 })
	a1.partyCommand("/raid")
	if s1.Party.Raid {
		t.Fatal("raid changed before confirmation")
	}
	wait1(func() bool { return s1.Party.Raid })
	wait2(func() bool { return s2.Party.Raid })
	for _, s := range []struct{ leader, sub, wantLeader, wantSub uint32 }{{s1.Party.Leader, s1.Party.RaidSubLeader, s1.SelfID, s2.SelfID}, {s2.Party.Leader, s2.Party.RaidSubLeader, s1.SelfID, s2.SelfID}} {
		if s.leader != s.wantLeader || s.sub != s.wantSub {
			t.Fatalf("conversion roles %+v", s)
		}
	}
	a1.partyCommand("/raidauto on")
	wait1(func() bool { return s1.Party.RaidAutoJoin })
	wait2(func() bool { return s2.Party.RaidAutoJoin })
	x, y := a1.partyUI.RaidOrigin(1024, 768)
	click := func(dx, dy int) {
		t.Helper()
		a1.updatePartyPointer(&platform.Input{Pressed: true, MouseDown: true, MouseX: x + dx, MouseY: y + dy}, 1024, 768)
		a1.updatePartyPointer(&platform.Input{Clicked: true, MouseX: x + dx, MouseY: y + dy}, 1024, 768)
	}
	click(310, 315)
	if !s1.Party.RaidAutoJoin {
		t.Fatal("auto-join changed before confirmation")
	}
	click(146, 315)
	if s1.Party.RaidDivision != 1 {
		t.Fatal("division changed before confirmation")
	}
	click(390, 410)
	if !a1.party.raidHidden {
		t.Fatal("OK did not close raid window")
	}
	wait1(func() bool { return !s1.Party.RaidAutoJoin && s1.Party.RaidDivision == 2 })
	wait2(func() bool { return !s2.Party.RaidAutoJoin && s2.Party.RaidDivision == 2 })
	a1.partyCommand("/raidwindow")
	click(555, 10)
	if !a1.party.raidHidden || !a1.partyCommand("/raidwindow") || a1.party.raidHidden {
		t.Fatal("raid window close/reopen failed")
	}
	a1.partyCommand("/raiddivision 3")
	wait1(func() bool { return s1.Party.RaidDivision == 3 })
	wait2(func() bool { return s2.Party.RaidDivision == 3 })
	a2.partyCommand("/raidswap 0 8") // assistant may rearrange slots
	wait1(func() bool { return s1.Party.RaidSlots[8] == s1.SelfID })
	wait2(func() bool { return s2.Party.RaidSlots[8] == s1.SelfID })
	a1.partyCommand("/leader " + s2.Slots[0].Name)
	wait1(func() bool { return s1.Party.Leader == s2.SelfID })
	wait2(func() bool { return s2.Party.Leader == s2.SelfID })
	a2.partyCommand("/raiddisband")
	wait1(func() bool { return !s1.Party.Raid && len(s1.Party.Members) == 0 })
	wait2(func() bool { return !s2.Party.Raid && len(s2.Party.Members) == 0 })
	// Raid invitations use a different answer opcode even before conversion.
	a1.partyCommand("/raidinvite " + s2.Slots[0].Name)
	wait2(func() bool { return s2.Party.InviteSerial > 1 })
	a2.updatePartyInvite(&platform.Input{}, 1024, 768)
	if !s2.Party.RaidInvite || !a2.partyMessage.Visible {
		t.Fatal("raid invitation missing")
	}
	a2.updatePartyInvite(&platform.Input{Keys: []platform.Key{platform.KeyEnter}}, 1024, 768)
	wait1(func() bool { return len(s1.Party.Members) == 2 })
	wait2(func() bool { return len(s2.Party.Members) == 2 })
	a2.partyCommand("/leave")
	wait1(func() bool { return len(s1.Party.Members) == 0 })
	wait2(func() bool { return len(s2.Party.Members) == 0 })
	send2(proto.LogoutPacket())
	wait2(func() bool { return s2.SelfID == 0 })
	send1(proto.LogoutPacket())
	wait1(func() bool { return s1.SelfID == 0 })
}
