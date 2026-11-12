package game

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

func TestGemAddOnlyAppliesConfirmedResult(t *testing.T) {
	s := New()
	gemKey, itemKey := [2]uint8{1, 1}, [2]uint8{1, 2}
	s.Inventory[gemKey] = proto.InventoryItem{Bag: 1, Slot: 1, Type: 30, TypeID: 4, Count: 2}
	s.Inventory[itemKey] = proto.InventoryItem{Bag: 1, Slot: 2, Type: 1, TypeID: 3, Count: 1}
	w := proto.NewWriter(proto.GemAdd)
	w.Raw([]byte{1, 1, 1, 1, 1, 2, 0, 4})
	w.U32(0)
	w.U32(875)
	p := w.Bytes()
	for i := 2; i < len(p); i++ {
		if _, err := s.Apply(p[:i]); err == nil || s.Inventory[gemKey].Count != 2 || s.Inventory[itemKey].Gems[0] != 0 {
			t.Fatalf("partial gem packet %d", i)
		}
	}
	if _, err := s.Apply(p); err != nil {
		t.Fatal(err)
	}
	if s.Inventory[gemKey].Count != 1 || s.Inventory[itemKey].Gems[0] != 4 || s.Details.Money != 875 || s.GemAddSerial != 1 {
		t.Fatal("gem confirmation not applied")
	}
	w = proto.NewWriter(proto.GemAdd)
	w.Raw([]byte{0, 1, 1, 0, 1, 2, 0, 0})
	w.U32(0)
	w.U32(750)
	if _, err := s.Apply(w.Bytes()); err != nil {
		t.Fatal(err)
	}
	if _, ok := s.Inventory[gemKey]; ok || s.Inventory[itemKey].Gems[0] != 4 || s.Details.Money != 750 {
		t.Fatal("failed attempt applied wrong state")
	}
}

func TestGemRemoveConfirmedAndFailureAmbiguous(t *testing.T) {
	s := New()
	key := [2]uint8{1, 2}
	item := proto.InventoryItem{Bag: 1, Slot: 2, Type: 1, TypeID: 3, Count: 1, Gems: [6]uint8{4, 5}}
	s.Inventory[key] = item
	packet := func(result byte, removed [6]byte) []byte {
		w := proto.NewWriter(proto.GemRemove)
		w.Raw([]byte{result, key[0], key[1]})
		w.Raw(removed[:])
		w.Raw([]byte{4, 5, 0, 0, 0, 0})
		w.Raw([]byte{1, 1, 0, 0, 0, 0})
		w.Raw([]byte{3, 4, 0, 0, 0, 0})
		w.U32(900)
		return w.Bytes()
	}
	p := packet(1, [6]byte{1, 0})
	for i := 2; i < len(p); i++ {
		if _, err := s.Apply(p[:i]); err == nil || s.Inventory[key] != item {
			t.Fatalf("partial response %d changed state", i)
		}
	}
	if _, err := s.Apply(p); err != nil {
		t.Fatal(err)
	}
	if s.Inventory[key].Gems != [6]uint8{0, 5} || s.Details.Money != 900 || s.GemRemoveSerial != 1 {
		t.Fatal("confirmed removal not applied")
	}
	before := s.Inventory[key]
	if _, err := s.Apply(packet(0, [6]byte{})); err != nil {
		t.Fatal(err)
	}
	if s.Inventory[key] != before {
		t.Fatal("ambiguous failure changed equipment without confirmation")
	}
}
