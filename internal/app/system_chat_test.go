package app

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/text"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

func TestAcknowledgedEventsUseRuntimeMessagesAndNativePalette(t *testing.T) {
	a := inputApp()
	a.state.SelfID = 1
	a.state.Entities[2] = &game.Entity{ID: 2, Kind: game.KindUser, Shape: &proto.UserShape{Name: "Opponent"}}
	a.msg = text.SysMsg{50: "XP <b>", 53: "Gold <b>", 1002: "Critical <t> <v>", 1003: "Received critical <t> <v>", 1004: "Miss <t>", 347: "No such character"}
	xp := proto.NewWriter(proto.ExpGain)
	xp.U32(1234)
	xp.U32(0)
	if _, err := a.state.Apply(xp.Bytes()); err != nil {
		t.Fatal(err)
	}
	a.confirmedSystemChat(xp.Bytes(), 0)
	if got := a.state.Chat[0]; got.Text != "XP 1234" || got.MessageID != 50 || got.Style != 3 {
		t.Fatal(got)
	}
	for _, tc := range []struct {
		source, target uint32
		result         uint8
		want           string
		rgb            uint32
	}{{1, 2, 1, "Critical Opponent 73", 0xff9b32}, {2, 1, 1, "Received critical Opponent 73", 0xff0000}, {1, 2, 3, "Miss Opponent", 0xff9b32}} {
		a.state.LastAttack = proto.AttackResult{Source: tc.source, Target: tc.target, Result: tc.result, HP: 73}
		a.confirmedSystemChat(proto.NewWriter(proto.CharacterUserAutoAttack).Bytes(), 0)
		got := chatEntry(a.msg, a.state.Chat[len(a.state.Chat)-1])
		if got.Text != tc.want || got.Color != ui.RGBColor(tc.rgb) || !got.System {
			t.Fatal(got)
		}
	}
	a.state.Details.Money = 100
	before := len(a.state.Chat)
	a.confirmedSystemChat(proto.NewWriter(proto.SetMoney).Bytes(), 100)
	a.confirmedSystemChat(proto.NewWriter(proto.SetMoney).Bytes(), 110)
	if len(a.state.Chat) != before {
		t.Fatal("balance snapshot or spending invented acquired gold")
	}
	a.confirmedSystemChat(proto.NewWriter(proto.SetMoney).Bytes(), 90)
	if a.state.Chat[len(a.state.Chat)-1].Text != "Gold 10" {
		t.Fatal(a.state.Chat)
	}
	got := chatEntry(a.msg, game.ChatLine{Op: proto.ChatError, Style: 1, MessageID: 347, Text: "hardcoded fallback"})
	if got.Text != "No such character" {
		t.Fatal("chat error ignored runtime table", got)
	}
}
