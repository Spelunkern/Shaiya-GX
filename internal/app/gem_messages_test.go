package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/text"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"testing"
)

func TestGemLinkUsesOriginalMessages(t *testing.T) {
	for _, success := range []bool{false, true} {
		a := &App{state: game.New(), msg: text.SysMsg{55: "Linked <i> with <r>", 56: "Failed <i> with <r>"}}
		a.gemLink = gemLinkView{pending: true, itemName: "Sword", gemName: "Lapis"}
		a.finishGemLink(success)
		id, want := 56, "Failed Sword with Lapis"
		if success {
			id, want = 55, "Linked Sword with Lapis"
		}
		if len(a.state.Chat) != 1 || a.state.Chat[0].MessageID != id || a.state.Chat[0].Text != want {
			t.Fatal(a.state.Chat)
		}
	}
}
