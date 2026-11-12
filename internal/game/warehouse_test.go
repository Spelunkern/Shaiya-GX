package game

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

func TestWarehouseAuthoritativeMovesAndMalformedWithdrawal(t *testing.T) {
	s := New()
	list := proto.NewWriter(proto.WarehouseItemList)
	list.U32(0)
	list.U8(1)
	list.U8(239)
	list.U8(25)
	list.U8(4)
	list.U16(100)
	list.Raw(make([]byte, 6))
	list.U8(10)
	list.Fixed("", 21, true)
	if _, err := s.Apply(list.Bytes()); err != nil {
		t.Fatal(err)
	}
	if !s.WarehouseReady || s.Warehouse[239].Count != 10 {
		t.Fatal("snapshot not applied", s.Warehouse)
	}
	move := proto.NewWriter(proto.ItemBagToBag)
	for _, v := range []struct{ bag, slot, typ, id, count uint8 }{{100, 239, 0, 0, 0}, {1, 23, 25, 4, 10}} {
		move.U8(v.bag)
		move.U8(v.slot)
		move.U8(v.typ)
		move.U8(v.id)
		move.U8(v.count)
		move.U16(100)
		move.Raw(make([]byte, 6))
		move.Fixed("", 21, true)
	}
	move.U32(99750)
	p := move.Bytes()
	if _, err := s.Apply(p[:len(p)-1]); err == nil {
		t.Fatal("accepted incomplete withdrawal")
	}
	if s.Warehouse[239].Count != 10 || len(s.Inventory) != 0 {
		t.Fatal("malformed packet changed inventory")
	}
	if _, err := s.Apply(p); err != nil {
		t.Fatal(err)
	}
	if len(s.Warehouse) != 0 || s.Inventory[[2]uint8{1, 23}].Count != 10 || s.Details.Money != 99750 {
		t.Fatal("withdrawal state incorrect", s.Warehouse, s.Inventory, s.Details.Money)
	}
}
