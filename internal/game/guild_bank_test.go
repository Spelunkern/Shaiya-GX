package game

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

func writeGuildBankRecord(w *proto.Writer, slot uint8) {
	w.U8(slot)
	w.U8(25)
	w.U8(4)
	w.U16(12)
	w.Raw([]byte{1, 2, 3, 4, 5, 6})
	w.U8(3)
	w.Fixed("", 21, true)
}

func writeMoveRecord(w *proto.Writer, bag, slot, typ, count uint8) {
	w.Raw([]byte{bag, slot, typ, 4, count})
	w.U16(12)
	w.Raw([]byte{1, 2, 3, 4, 5, 6})
	w.Fixed("", 21, true)
}

func TestGuildBankSnapshotsUpdatesAndWithdrawalMoney(t *testing.T) {
	s := New()
	list := proto.NewWriter(proto.GuildBankList)
	list.U8(1)
	writeGuildBankRecord(list, 3)
	packet := list.Bytes()
	for i := 2; i < len(packet); i++ {
		if _, err := s.Apply(packet[:i]); err == nil || len(s.Guild.Bank) != 0 {
			t.Fatalf("partial snapshot %d", i)
		}
	}
	if _, err := s.Apply(packet); err != nil || s.Guild.Bank[3].Count != 3 {
		t.Fatal("bank list", err)
	}
	move := proto.NewWriter(proto.ItemBagToBag)
	writeMoveRecord(move, proto.GuildBankBag, 3, 0, 0)
	writeMoveRecord(move, 1, 5, 25, 3)
	move.U32(987)
	packet = move.Bytes()
	for i := 2; i < len(packet); i++ {
		if _, err := s.Apply(packet[:i]); err == nil || s.Guild.Bank[3].Count != 3 || s.Inventory[[2]uint8{1, 5}].Count != 0 {
			t.Fatalf("partial withdrawal %d", i)
		}
	}
	if _, err := s.Apply(packet); err != nil || len(s.Guild.Bank) != 0 || s.Inventory[[2]uint8{1, 5}].Count != 3 || s.Details.Money != 987 {
		t.Fatal("withdrawal", err)
	}
	if _, ok := s.Inventory[[2]uint8{proto.GuildBankBag, 3}]; ok {
		t.Fatal("guild bank leaked into inventory")
	}
	placed := proto.NewWriter(proto.GuildBankPlaced)
	writeGuildBankRecord(placed, 7)
	placed.U32(99)
	if _, err := s.Apply(placed.Bytes()); err != nil || s.Guild.Bank[7].Type != 25 {
		t.Fatal("other player deposit", err)
	}
	taken := proto.NewWriter(proto.GuildBankTaken)
	writeGuildBankRecord(taken, 7)
	taken.U32(99)
	if _, err := s.Apply(taken.Bytes()); err != nil || len(s.Guild.Bank) != 0 {
		t.Fatal("other player withdrawal", err)
	}
}
