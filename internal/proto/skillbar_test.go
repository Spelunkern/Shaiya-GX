package proto

import (
	"bytes"
	"reflect"
	"testing"
)

func TestSkillBarWireAndValidation(t *testing.T) {
	slots := []QuickSlot{{Bar: 0, Slot: 9, Bag: 100, Number: 7}, {Bar: 4, Slot: 0, Bag: 3, Number: 8}}
	p, err := SkillBarPacket(slots)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(p, []byte{0x0b, 1, 2, 0, 9, 100, 7, 0, 4, 0, 3, 8, 0}) {
		t.Fatalf("wire=%x", p)
	}
	got, err := DecodeQuickSlots(p)
	if err != nil || !reflect.DeepEqual(got, slots) {
		t.Fatalf("decode=%v %v", got, err)
	}
	for _, q := range []QuickSlot{{Bar: 5}, {Slot: 10}, {Bag: 6}, {Bag: 100}, {Bag: 1, Number: 24}} {
		if _, err := SkillBarPacket([]QuickSlot{q}); err == nil {
			t.Fatalf("accepted %+v", q)
		}
	}
	if _, err := SkillBarPacket(make([]QuickSlot, 129)); err == nil {
		t.Fatal("accepted oversized bar")
	}
}
