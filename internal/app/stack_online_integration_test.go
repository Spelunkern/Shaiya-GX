//go:build integration

package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/testworld"
	"testing"
)

func TestGMStacksAndClearAgainstIsolatedGX(t *testing.T) {
	st, _, send, wait := testworld.Enter(t)
	wait(func() bool { return st.Self.IsGM })
	if st.GMTurbo {
		t.Fatal("turbo enabled by default")
	}
	equipped := map[[2]uint8]proto.InventoryItem{}
	for key, it := range st.Inventory {
		if key[0] == 0 {
			equipped[key] = it
		}
	}
	command := func(text string) {
		before := len(st.Chat)
		send(proto.ChatPacket(proto.ChannelNormal, false, text))
		wait(func() bool { return len(st.Chat) > before })
	}
	command("/iclear")
	for key, it := range st.Inventory {
		if key[0] != 0 {
			t.Fatal("carried item survived clear")
		}
		if equipped[key] != it {
			t.Fatal("clear changed equipment")
		}
	}
	for key, it := range equipped {
		if st.Inventory[key] != it {
			t.Fatal("clear removed equipment")
		}
	}
	command("/getitem 25 1 255")
	var source [2]uint8
	for key, it := range st.Inventory {
		if key[0] > 0 && it.Type == 25 && it.TypeID == 1 {
			source = key
			if it.Count != 255 {
				t.Fatal("grant truncated")
			}
		}
	}
	if source[0] == 0 {
		t.Fatal("missing granted apples")
	}
	// Exercises the wide opcode and request count through actual encryption,
	// dispatch, database persistence and client inventory acknowledgement.
	send(proto.ShopSellPacket(source[0], source[1], 255))
	wait(func() bool { return st.Inventory[source].Count == 0 })
	t.Log("default turbo off, equipped items preserved, 255 grant and 255 sale confirmed")
}
