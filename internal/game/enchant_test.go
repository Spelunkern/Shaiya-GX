package game

import (
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"testing"
)

func TestEnchantResultAppliesAtomically(t *testing.T) {
	s := New()
	l, m, i := [2]uint8{1, 1}, [2]uint8{1, 2}, [2]uint8{1, 3}
	s.Inventory[l] = proto.InventoryItem{Bag: 1, Slot: 1, Type: 95, TypeID: 1, Count: 2}
	s.Inventory[m] = proto.InventoryItem{Bag: 1, Slot: 2, Type: 25, TypeID: 1, Count: 4}
	s.Inventory[i] = proto.InventoryItem{Bag: 1, Slot: 3, Type: 1, TypeID: 1, Count: 1}
	w := proto.NewWriter(proto.EnchantAdd)
	w.Raw([]byte{1, 1, 1, 1, 1, 2, 2, 1, 3})
	w.U32(500)
	w.Fixed("00000000000000000001", 21, true)
	p := w.Bytes()
	for n := 2; n < len(p); n++ {
		if _, err := s.Apply(p[:n]); err == nil || s.Inventory[l].Count != 2 || s.Inventory[m].Count != 4 {
			t.Fatalf("truncated packet %d changed state", n)
		}
	}
	if _, err := s.Apply(p); err != nil {
		t.Fatal(err)
	}
	if s.Inventory[l].Count != 1 || s.Inventory[m].Count != 2 || s.Inventory[i].Craft != "00000000000000000001" || s.Details.Money != 500 || s.EnchantSerial != 1 {
		t.Fatal("confirmed enchant not applied")
	}
}
