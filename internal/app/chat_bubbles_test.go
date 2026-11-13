package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"testing"
)

func TestOrdinaryChatBubbleOnlyConfirmedEchoAndReplacement(t *testing.T) {
	a := inputApp()
	a.clock = 10
	a.state.AddChat(game.ChatLine{Op: proto.ChatParty, SenderID: 7, Text: "group"})
	a.confirmedChatBubble(proto.ChatParty)
	if len(a.chatBubbles) != 0 {
		t.Fatal("party produced ordinary bubble")
	}
	a.state.AddChat(game.ChatLine{Op: proto.ChatNormal, SenderID: 7, Text: "first"})
	a.confirmedChatBubble(proto.ChatNormal)
	a.clock = 12
	a.state.AddChat(game.ChatLine{Op: proto.ChatNormal, SenderID: 7, Text: "replaced 😀"})
	a.confirmedChatBubble(proto.ChatNormal)
	if len(a.chatBubbles) != 1 || a.chatBubbles[0].text != "replaced 😀" || a.chatBubbles[0].until != 17 {
		t.Fatal("bubble did not replace/reset", a.chatBubbles)
	}
}
