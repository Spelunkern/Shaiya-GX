//go:build integration

package app

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/testworld"
)

func TestChatAgainstRapture(t *testing.T) {
	st, world, send, wait := testworld.Enter(t)
	a := inputApp()
	a.state, a.world = st, world
	const message = "N2 chat input and sender identity"
	a.updateChat(&platform.Input{Keys: []platform.Key{platform.KeyEnter}})
	a.updateChat(&platform.Input{Chars: []rune(message)})
	a.updateChat(&platform.Input{Keys: []platform.Key{platform.KeyEnter}})
	if a.chatOpen || a.state == nil {
		t.Fatal("chat submit did not return to world controls")
	}
	var received *game.ChatLine
	wait(func() bool {
		for i := range st.Chat {
			line := &st.Chat[i]
			if line.Text == message && line.SenderID == st.SelfID {
				received = line
				return true
			}
		}
		return false
	})
	if received.Sender != st.Slots[0].Name {
		t.Fatal("received chat lost the roster name", received.Sender)
	}
	t.Log("typed message echoed by Rapture with the known character name")
	send(proto.LogoutPacket())
	wait(func() bool { return st.Phase == game.PhaseCharSelect })
}
