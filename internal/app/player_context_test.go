package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"testing"
)

func TestPlayerContextActionsAndPointerCapture(t *testing.T) {
	a := inputApp()
	a.state.SelfID = 1
	a.state.Target = 2
	a.state.Entities[2] = &game.Entity{ID: 2, Kind: game.KindUser, Shape: &proto.UserShape{Name: "Ally"}}
	controls := a.worldControls(&platform.Input{MouseX: 300, MouseY: 20, RightPressed: true, RightDown: true}, 1024, 768)
	if !a.playerContext.open || controls.RightPressed || controls.RightDown {
		t.Fatal("context click reached camera")
	}
	actions := a.contextActions(2)
	if len(actions) != 4 || actions[0].label != "Invite to party" || actions[1].label != "Trade" || actions[2].label != "Duel" || actions[3].label != "Whisper" {
		t.Fatal(actions)
	}
	a.state.Party.Members = []proto.PartyMember{{ID: 1}, {ID: 2}}
	a.state.Party.Leader = 1
	actions = a.contextActions(2)
	if len(actions) != 5 || actions[0].label != "Remove from party" || actions[1].label != "Make leader" {
		t.Fatal(actions)
	}
	a.worldControls(&platform.Input{Keys: []platform.Key{platform.KeyEscape}}, 1024, 768)
	if a.playerContext.open {
		t.Fatal("context did not close")
	}
}

func TestPlayerContextDoesNotActivateChangedAction(t *testing.T) {
	a := inputApp()
	a.state.SelfID = 1
	a.state.Target = 2
	a.state.Entities[2] = &game.Entity{ID: 2, Kind: game.KindUser, Shape: &proto.UserShape{Name: "Ally"}}
	a.worldControls(&platform.Input{MouseX: 300, MouseY: 20, RightPressed: true, RightDown: true}, 1024, 768)
	x, y := a.playerContext.x, a.playerContext.y
	a.worldControls(&platform.Input{MouseX: x + 10, MouseY: y + 15, Pressed: true, MouseDown: true}, 1024, 768)
	a.state.Party.Members = []proto.PartyMember{{ID: 1}, {ID: 2}}
	a.state.Party.Leader = 1
	// The first row changed from invite to kick; release must not send either.
	a.worldControls(&platform.Input{MouseX: x + 10, MouseY: y + 15, Clicked: true}, 1024, 768)
	if a.playerContext.open {
		t.Fatal("context remained open")
	}
}

func TestPlayerContextWhisperUsesSelectedNameAndKeepsMovement(t *testing.T) {
	a := inputApp()
	a.state.SelfID, a.state.Target = 1, 2
	a.state.Entities[2] = &game.Entity{ID: 2, Kind: game.KindUser, Shape: &proto.UserShape{Name: "Ally"}}
	a.msg = map[int]string{5007: "Susurrar"}
	a.worldControls(&platform.Input{MouseX: 300, MouseY: 20, RightPressed: true, RightDown: true}, 1024, 768)
	index := len(a.contextActions(2)) - 1
	x, y := a.playerContext.x+10, a.playerContext.y+10+index*contextRowHeight+5
	controls := a.worldControls(&platform.Input{MouseX: x, MouseY: y, Pressed: true, MouseDown: true, Down: "W"}, 1024, 768)
	if controls.Down != "W" || controls.Pressed {
		t.Fatal("popup consumed movement or leaked its press", controls)
	}
	a.worldControls(&platform.Input{MouseX: x, MouseY: y, Clicked: true}, 1024, 768)
	if !a.chatOpen || a.chatUI.Channel != 5 || a.chatUI.Recipient != "Ally" || a.playerContext.open {
		t.Fatal("whisper did not focus the selected recipient")
	}
	labels := a.contextLabels(a.contextActions(2))
	if labels[len(labels)-1] != "Susurrar" {
		t.Fatal("menu ignored original message table", labels)
	}
}

func TestPlayerContextRejectsNonPlayers(t *testing.T) {
	a := inputApp()
	a.state.SelfID = 1
	a.state.Entities[2] = &game.Entity{ID: 2, Kind: game.KindMob}
	if len(a.contextActions(2)) != 0 || len(a.contextActions(99)) != 0 {
		t.Fatal("player services offered on an invalid target")
	}
}

func TestPlayerContextOwnPortraitOffersConfirmedLeave(t *testing.T) {
	a := inputApp()
	a.state.SelfID = 1
	a.state.Party.Members = []proto.PartyMember{{ID: 1}, {ID: 2}}
	a.state.Party.Leader = 1
	controls := a.worldControls(&platform.Input{MouseX: 100, MouseY: 20, RightPressed: true, RightDown: true}, 1024, 768)
	actions := a.contextActions(1)
	if !a.playerContext.open || a.playerContext.id != 1 || controls.RightDown || len(actions) != 1 || actions[0].label != "Leave party" {
		t.Fatal("own portrait did not offer party leave", actions)
	}
	if len(a.state.Party.Members) != 2 {
		t.Fatal("opening context changed party membership")
	}
}

func TestPlayerContextPartyEligibility(t *testing.T) {
	a := inputApp()
	a.state.SelfID = 1
	a.state.Slots[0] = proto.CharacterSlot{ID: 1, Race: 0}
	a.state.Entities[2] = &game.Entity{ID: 2, Kind: game.KindUser, Shape: &proto.UserShape{Race: 2, Name: "Enemy"}}
	hasInvite := func() bool {
		for _, action := range a.contextActions(2) {
			if action.label == "Invite to party" || action.label == "Invite to raid" {
				return true
			}
		}
		return false
	}
	if hasInvite() {
		t.Fatal("party invite offered across factions")
	}
	a.state.Entities[2].Shape.Race = 0
	a.state.Party.Leader = 1
	for i := uint32(10); i < 17; i++ {
		a.state.Party.Members = append(a.state.Party.Members, proto.PartyMember{ID: i})
	}
	if hasInvite() {
		t.Fatal("party invite offered for a full group")
	}
	a.state.Party.Members = nil
	if !hasInvite() {
		t.Fatal("available friendly player has no invitation")
	}
}

func TestPlayerContextDoesNotOpenThroughInventory(t *testing.T) {
	a := inputApp()
	a.state.SelfID, a.state.Target = 1, 2
	a.state.Entities[2] = &game.Entity{ID: 2, Kind: game.KindUser, Shape: &proto.UserShape{Name: "Ally"}}
	a.inventory.open = true
	a.inventoryUI.Position.Move(270, 0, 1024, 768, 256, 486)
	a.worldControls(&platform.Input{MouseX: 300, MouseY: 20, RightPressed: true, RightDown: true}, 1024, 768)
	if a.playerContext.open {
		t.Fatal("target popup opened through a foreground window")
	}
}
