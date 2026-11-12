package game

import (
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"testing"
)

func TestRecreationAppliesOnlyCompleteSuccess(t *testing.T) {
	s := New()
	key := [2]uint8{1, 2}
	item := proto.InventoryItem{Bag: 1, Slot: 2, Type: 1, TypeID: 1, Count: 1, Craft: "00000000000000000001"}
	s.Inventory[key] = item
	packet := func(result uint8, craft string) []byte {
		w := proto.NewWriter(proto.ItemCompose)
		w.Raw([]byte{result, 1, 2})
		w.Fixed(craft, 21, true)
		return w.Bytes()
	}
	p := packet(0, "01020304050600000001")
	for n := 2; n < len(p); n++ {
		if _, err := s.Apply(p[:n]); err == nil || s.Inventory[key] != item {
			t.Fatalf("truncated recreation response %d", n)
		}
	}
	if _, err := s.Apply(packet(1, "")); err != nil || s.Inventory[key] != item {
		t.Fatal("refusal modified equipment")
	}
	if _, err := s.Apply(p); err != nil || s.Inventory[key].Craft != "01020304050600000001" || s.ComposeSerial != 2 {
		t.Fatal("confirmed recreation not applied")
	}
}
