package app

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

func TestRaidSettingsCancelAndReopen(t *testing.T) {
	a := inputApp()
	a.partyUI, _ = ui.NewPartyPanel(inputTextures{})
	a.state.SelfID = 1
	a.state.Party.Raid, a.state.Party.Leader, a.state.Party.RaidDivision = true, 1, 1
	a.state.Party.Members = []proto.PartyMember{{ID: 1}, {ID: 2}}
	a.state.Party.RaidSlots = map[uint16]uint32{0: 1, 1: 2}
	x, y := a.partyUI.RaidOrigin(1024, 768)
	click := func(dx, dy int) {
		a.updatePartyPointer(&platform.Input{Pressed: true, MouseDown: true, MouseX: x + dx, MouseY: y + dy}, 1024, 768)
		a.updatePartyPointer(&platform.Input{Clicked: true, MouseX: x + dx, MouseY: y + dy}, 1024, 768)
	}
	click(35, 347) // master loot: native wire value 4
	if division, _ := a.partyUI.RaidSettings(); division != 4 || a.state.Party.RaidDivision != 1 {
		t.Fatal("draft was not isolated from confirmed state")
	}
	click(475, 410) // Cancel
	if !a.party.raidHidden {
		t.Fatal("cancel did not close")
	}
	if !a.partyCommand("/raidwindow") || a.party.raidHidden {
		t.Fatal("window did not reopen")
	}
	a.updatePartyPointer(&platform.Input{}, 1024, 768)
	if division, _ := a.partyUI.RaidSettings(); division != 1 {
		t.Fatal("cancel retained draft")
	}
}

func TestCompactRaidCapturesSelectionAndContext(t *testing.T) {
	a := inputApp()
	a.partyUI, _ = ui.NewPartyPanel(inputTextures{})
	a.state.SelfID = 1
	a.state.Party.Raid, a.state.Party.Leader = true, 1
	a.state.Party.Members = []proto.PartyMember{{ID: 1}, {ID: 2}}
	a.state.Party.RaidSlots = map[uint16]uint32{0: 1, 1: 2}
	a.party.raidHidden = true
	x, y := a.partyUI.RaidCompactOrigin(1024, 768)
	pointX, pointY := x+8, y+24+ui.PartyRowHeight+8
	pressed := a.worldControls(&platform.Input{Pressed: true, MouseDown: true, MouseX: pointX, MouseY: pointY}, 1024, 768)
	released := a.worldControls(&platform.Input{Clicked: true, MouseX: pointX, MouseY: pointY}, 1024, 768)
	if pressed.Pressed || released.Clicked || a.state.Target != 2 {
		t.Fatalf("compact selection leaked to world: target %d", a.state.Target)
	}
	context := a.worldControls(&platform.Input{RightPressed: true, RightDown: true, MouseX: pointX, MouseY: pointY}, 1024, 768)
	if context.RightPressed || !a.playerContext.open || a.playerContext.id != 2 {
		t.Fatal("compact row context menu failed")
	}
}
