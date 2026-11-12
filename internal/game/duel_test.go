package game

import (
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"testing"
)

func TestDuelAuthoritativeLifecycle(t *testing.T) {
	s := New()
	s.SelfID = 1
	s.Inventory[[2]uint8{1, 2}] = proto.InventoryItem{Type: 25, TypeID: 4, Count: 5}
	request := proto.NewWriter(proto.DuelRequest)
	request.U32(2)
	request.U32(1)
	p := request.Bytes()
	for i := 2; i < len(p); i++ {
		if _, err := s.Apply(p[:i]); err == nil {
			t.Fatalf("accepted truncated challenge %d", i)
		}
		if s.Duel.Phase != DuelNone {
			t.Fatal("truncated challenge mutated state")
		}
	}
	if _, err := s.Apply(p); err != nil || s.Duel.Phase != DuelIncoming || s.Duel.Opponent != 2 {
		t.Fatal("challenge", s.Duel, err)
	}
	window := proto.NewWriter(proto.DuelWindow)
	window.U32(2)
	window.U8(0)
	if _, err := s.Apply(window.Bytes()); err != nil || s.Duel.Phase != DuelWager {
		t.Fatal("window", err)
	}
	add := proto.NewWriter(proto.DuelAdd)
	add.Raw([]byte{1, 2, 3, 0})
	if _, err := s.Apply(add.Bytes()); err != nil || s.Duel.Own[0].Item.Count != 3 || s.Inventory[[2]uint8{1, 2}].Count != 5 {
		t.Fatal("offer", err)
	}
	other := proto.NewWriter(proto.DuelAddOther)
	other.U8(0)
	other.U8(12)
	other.U8(3)
	other.U8(2)
	other.U16(50)
	other.Raw(make([]byte, 6))
	other.Fixed("", 21, true)
	if _, err := s.Apply(other.Bytes()); err != nil || s.Duel.Other[0].Count != 2 {
		t.Fatal("other", err)
	}
	remove := proto.NewWriter(proto.DuelRemove)
	remove.Raw([]byte{1, 0})
	if _, err := s.Apply(remove.Bytes()); err != nil || s.Duel.Own[0].Item.Count != 0 || s.Duel.Other[0].Count != 2 {
		t.Fatal("side-aware removal", err)
	}
	ready := proto.NewWriter(proto.DuelReady)
	ready.Raw([]byte{3, 0})
	if _, err := s.Apply(ready.Bytes()); err != nil || !s.Duel.OwnReady || !s.Duel.OtherReady {
		t.Fatal("ready", err)
	}
	countdown := proto.NewWriter(proto.DuelCountdown)
	countdown.F32(100)
	countdown.F32(200)
	if _, err := s.Apply(countdown.Bytes()); err != nil || s.Duel.Phase != DuelCounting || s.Duel.CenterZ != 200 {
		t.Fatal("countdown", err)
	}
	if _, err := s.Apply(proto.NewWriter(proto.DuelStart).Bytes()); err != nil || s.Duel.Phase != DuelFighting {
		t.Fatal("start", err)
	}
	end := proto.NewWriter(proto.DuelEnd)
	end.U8(2)
	end.U32(2)
	if _, err := s.Apply(end.Bytes()); err != nil || s.Duel.Phase != DuelNone || s.Duel.Result.Player != 2 {
		t.Fatal("end", err)
	}
}
