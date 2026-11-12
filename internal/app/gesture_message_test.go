package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/text"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"testing"
)

func TestGestureMessagesUseLocalizedTemplateAndKnownName(t *testing.T) {
	a := inputApp()
	a.msg = text.SysMsg{5233: "<p> says hello."}
	a.state.SelfID = 1
	a.state.Slots[0] = proto.CharacterSlot{ID: 1, Name: "Alice"}
	a.state.LastMotion = proto.CharacterMotionResult{ID: 1, Motion: 121}
	a.confirmedGesture(proto.CharacterMotion)
	if len(a.state.Chat) != 1 || a.state.Chat[0].Text != "Alice says hello." {
		t.Fatal(a.state.Chat)
	}
	a.state.Entities[2] = &game.Entity{ID: 2, Kind: game.KindUser, Shape: &proto.UserShape{Name: "Bob"}}
	a.state.LastMotion.ID = 2
	a.confirmedGesture(proto.CharacterMotion)
	if len(a.state.Chat) != 2 || a.state.Chat[1].Text != "Bob says hello." {
		t.Fatal(a.state.Chat)
	}
	a.state.LastMotion.ID = 3
	a.confirmedGesture(proto.CharacterMotion)
	if len(a.state.Chat) != 2 {
		t.Fatal("unknown actor produced a message")
	}
	a.state.LastMotion.Motion = 10
	a.confirmedGesture(proto.CharacterMotion)
	if len(a.state.Chat) != 2 {
		t.Fatal("sit produced a gesture message")
	}
}
