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
	if len(actions) != 3 || actions[0].label != "Invite to party" || actions[1].label != "Trade" || actions[2].label != "Duel" {
		t.Fatal(actions)
	}
	a.state.Party.Members = []proto.PartyMember{{ID: 1}, {ID: 2}}
	a.state.Party.Leader = 1
	actions = a.contextActions(2)
	if len(actions) != 5 || actions[0].label != "Make leader" || actions[1].label != "Remove from party" {
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
	a.worldControls(&platform.Input{MouseX: x + 5, MouseY: y + 5, Pressed: true, MouseDown: true}, 1024, 768)
	a.state.Party.Members = []proto.PartyMember{{ID: 1}, {ID: 2}}
	a.state.Party.Leader = 1
	// The first row changed from invite to promote; release must not send either.
	a.worldControls(&platform.Input{MouseX: x + 5, MouseY: y + 5, Clicked: true}, 1024, 768)
	if a.playerContext.open {
		t.Fatal("context remained open")
	}
}
