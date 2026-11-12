package app

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/session"
)

func TestGuildBankRejectsUnsafeMovesLocally(t *testing.T) {
	a := inputApp()
	a.state.Guild.ID = 42
	a.state.Guild.Rank = 9
	a.items = map[sdata.ItemKey]*sdata.Item{{Type: 25, TypeID: 4}: {Type: 25, TypeID: 4, SellPrice: 100}}
	a.state.Inventory[[2]uint8{1, 0}] = proto.InventoryItem{Bag: 1, Slot: 0, Type: 25, TypeID: 4, Count: 2}
	bag, bank := [2]uint8{1, 0}, [2]uint8{proto.GuildBankBag, 0}
	if a.guildBankAllowed(bag, bank) {
		t.Fatal("rank-nine deposit allowed")
	}
	a.state.Guild.Rank = 1
	if !a.guildBankAllowed(bag, bank) {
		t.Fatal("master deposit rejected")
	}
	a.items[sdata.ItemKey{Type: 25, TypeID: 4}].ReqOG = 1
	if a.guildBankAllowed(bag, bank) {
		t.Fatal("restricted item deposit allowed")
	}
	a.items[sdata.ItemKey{Type: 25, TypeID: 4}].ReqOG = 0
	a.state.Guild.Bank = map[uint8]proto.InventoryItem{0: {Bag: proto.GuildBankBag, Slot: 0, Type: 25, TypeID: 4, Count: 2}}
	a.state.Details.Money = 9
	if a.guildBankAllowed(bank, bag) {
		t.Fatal("unaffordable withdrawal allowed")
	}
	a.state.Details.Money = 10
	if !a.guildBankAllowed(bank, bag) {
		t.Fatal("affordable withdrawal rejected")
	}
}

func TestGuildBankPendingIgnoresOtherMembersMoves(t *testing.T) {
	a := inputApp()
	packets := make(chan []byte, 2)
	a.world = &session.World{Packets: packets}
	a.inventory.pending = true
	a.bank.guildPending = true
	a.bank.sourceMove = [2]uint8{1, 2}
	a.bank.destMove = [2]uint8{proto.GuildBankBag, 0}
	move := func(src, dst uint8) []byte {
		w := proto.NewWriter(proto.ItemBagToBag)
		for _, slot := range []uint8{src, dst} {
			w.Raw([]byte{proto.GuildBankBag, slot, 0, 0, 0})
			w.U16(0)
			w.Raw(make([]byte, 6))
			w.Fixed("", 21, true)
		}
		return w.Bytes()
	}
	packets <- move(3, 4)
	a.pumpWorld()
	if !a.bank.guildPending || !a.inventory.pending {
		t.Fatal("unrelated broadcast cleared pending transfer")
	}
	result := proto.NewWriter(proto.GuildBankResult)
	result.U8(1)
	result.U8(30)
	packets <- result.Bytes()
	a.pumpWorld()
	if a.bank.guildPending || a.inventory.pending || len(a.state.Chat) == 0 {
		t.Fatal("refusal did not release pending transfer")
	}
}
