package game

import (
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"testing"
)

func TestExchangeServerOwnedLifecycle(t *testing.T) {
	s := New()
	source := [2]uint8{1, 2}
	s.Inventory[source] = proto.InventoryItem{Bag: 1, Slot: 2, Type: 25, TypeID: 4, Count: 7, Quality: 100}
	invite := proto.NewWriter(proto.ExchangeRequest)
	invite.U32(42)
	if _, err := s.Apply(invite.Bytes()); err != nil || s.Exchange.InviteID != 42 {
		t.Fatal("invite", err)
	}
	start := proto.NewWriter(proto.ExchangeStart)
	start.U32(42)
	start.U8(0)
	if _, err := s.Apply(start.Bytes()); err != nil || !s.Exchange.Active {
		t.Fatal("start", err)
	}
	add := proto.NewWriter(proto.ExchangeAdd)
	add.Raw([]byte{1, 2, 3, 0})
	p := add.Bytes()
	for i := 2; i < len(p); i++ {
		if _, err := s.Apply(p[:i]); err == nil {
			t.Fatalf("accepted truncated offer %d", i)
		}
		if s.Exchange.Own[0].Item.Count != 0 {
			t.Fatal("truncated offer changed state")
		}
	}
	if _, err := s.Apply(p); err != nil || s.Exchange.Own[0].Item.Count != 3 || s.Inventory[source].Count != 7 {
		t.Fatal("offer", err)
	}
	other := proto.NewWriter(proto.ExchangeAddDest)
	other.U8(1)
	other.U8(12)
	other.U8(3)
	other.U8(2)
	other.U16(50)
	other.Raw(make([]byte, 6))
	other.Fixed("", 21, true)
	if _, err := s.Apply(other.Bytes()); err != nil || s.Exchange.Other[1].Count != 2 {
		t.Fatal("other offer", err)
	}
	ready := proto.NewWriter(proto.ExchangeReady)
	ready.Raw([]byte{3, 0})
	if _, err := s.Apply(ready.Bytes()); err != nil || !s.Exchange.OwnReady || !s.Exchange.OtherReady {
		t.Fatal("ready", err)
	}
	confirm := proto.NewWriter(proto.ExchangeConfirm)
	confirm.Raw([]byte{2, 1})
	if _, err := s.Apply(confirm.Bytes()); err != nil || !s.Exchange.OtherConfirmed {
		t.Fatal("confirm", err)
	}
	reset := proto.NewWriter(proto.ExchangeReady)
	reset.Raw([]byte{3, 1})
	if _, err := s.Apply(reset.Bytes()); err != nil || s.Exchange.OwnReady || s.Exchange.OtherReady {
		t.Fatal("unready", err)
	}
	s.MarkExchangeRemove(0)
	if s.Exchange.Own[0].Item.Count != 3 {
		t.Fatal("removal changed offer before ack")
	}
	remove := proto.NewWriter(proto.ExchangeAddDest)
	remove.U8(0)
	remove.Raw(make([]byte, 32))
	if _, err := s.Apply(remove.Bytes()); err != nil || s.Exchange.Own[0].Item.Count != 0 || s.Exchange.Other[1].Count != 2 {
		t.Fatal("remove", err)
	}
	end := proto.NewWriter(proto.ExchangeEnd)
	end.U8(1)
	if _, err := s.Apply(end.Bytes()); err != nil || s.Exchange.Active || s.Exchange.EndResult != 1 || s.Inventory[source].Count != 7 {
		t.Fatal("end", err)
	}
}

func TestExchangeAmbiguousRemovalPreservesBothOffers(t *testing.T) {
	s := New()
	s.Exchange.Active = true
	s.Exchange.Own[0].Item = proto.ExchangeItem{Type: 25, Count: 1}
	s.MarkExchangeRemove(0)
	add := proto.NewWriter(proto.ExchangeAddDest)
	add.U8(0)
	add.U8(12)
	add.U8(3)
	add.U8(2)
	add.U16(50)
	add.Raw(make([]byte, 6))
	add.Fixed("", 21, true)
	if _, err := s.Apply(add.Bytes()); err != nil {
		t.Fatal(err)
	}
	remove := proto.NewWriter(proto.ExchangeAddDest)
	remove.U8(0)
	remove.Raw(make([]byte, 32))
	if _, err := s.Apply(remove.Bytes()); err != nil {
		t.Fatal(err)
	}
	if !s.Exchange.Desync || s.Exchange.Own[0].Item.Count != 1 || s.Exchange.Other[0].Count != 2 {
		t.Fatal("ambiguous removal guessed ownership")
	}
}
