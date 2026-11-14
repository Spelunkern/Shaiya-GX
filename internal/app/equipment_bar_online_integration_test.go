//go:build integration

package app

import (
	"encoding/json"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/testworld"
	"os"
	"testing"
)

func TestEquipmentBarSwitchAgainstGX(t *testing.T) {
	var accounts [][2]string
	raw := os.Getenv("SHAIYA_TEST_VISIBILITY_ACCOUNTS")
	if raw == "" {
		t.Skip("needs disposable fixtures")
	}
	if err := json.Unmarshal([]byte(raw), &accounts); err != nil || len(accounts) < 3 {
		t.Fatal("invalid fixtures")
	}
	st, _, send, wait := testworld.EnterAs(t, accounts[2][0], accounts[2][1])
	for _, cmd := range []string{"/getitem 1 2 1", "/getitem 1 1 1"} {
		n := len(st.Chat)
		send(proto.ChatPacket(proto.ChannelNormal, false, cmd))
		wait(func() bool { return len(st.Chat) > n })
		if st.Chat[len(st.Chat)-1].Style != 4 {
			t.Fatal("GM command response is not green")
		}
	}
	carry, equip := [2]uint8{1, 0}, [2]uint8{0, 5}
	send(proto.ItemMovePacket(1, 1, 0, 5))
	wait(func() bool { return st.Inventory[equip].TypeID == 1 })
	slots := []proto.QuickSlot{{Bar: 0, Slot: 0, Bag: carry[0], Number: uint16(carry[1])}}
	packet, err := proto.SkillBarPacket(slots)
	if err != nil {
		t.Fatal(err)
	}
	send(packet)
	st.QuickSlots, st.SkillBarReady = slots, true
	// A response barrier drains the previous action-bar save.
	n := len(st.Chat)
	send(proto.ChatPacket(proto.ChannelNormal, false, "/attack invalid"))
	wait(func() bool { return len(st.Chat) > n })
	a := &App{state: st}
	for i := 0; i < 4; i++ {
		beforeCarry, beforeEquip := st.Inventory[carry], st.Inventory[equip]
		p := a.actionPacket(0)
		if proto.Op(p) != proto.ItemBagToBag {
			t.Fatal("shortcut did not request equipment swap")
		}
		send(p)
		wait(func() bool { return st.Inventory[equip].TypeID == beforeCarry.TypeID })
		if st.Inventory[carry].TypeID != beforeEquip.TypeID || len(st.QuickSlots) != 1 || st.QuickSlots[0].Bag != 1 || st.QuickSlots[0].Number != 0 {
			t.Fatal("previous equipment did not remain on the same shortcut")
		}
	}
	t.Log("four authoritative equipment swaps preserve carried position and shortcut; GM responses green")
}
