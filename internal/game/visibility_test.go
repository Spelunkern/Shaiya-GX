package game

import (
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"testing"
)

func TestVisibilityStateAndLeave(t *testing.T) {
	s := New()
	s.SelfID = 1
	s.Entities[2] = &Entity{ID: 2, Kind: KindUser}
	for _, id := range []uint32{1, 2} {
		w := proto.NewWriter(proto.GXVisibility)
		w.U32(id)
		w.U8(1)
		if _, err := s.Apply(w.Bytes()); err != nil {
			t.Fatal(err)
		}
	}
	if !s.Self.Concealed || !s.Entities[2].Concealed {
		t.Fatal("missing ghost state")
	}
	w := proto.NewWriter(proto.UserLeave)
	w.U32(2)
	if _, err := s.Apply(w.Bytes()); err != nil {
		t.Fatal(err)
	}
	if s.Entities[2] != nil {
		t.Fatal("hidden enemy remains targetable in client")
	}
	w = proto.NewWriter(proto.GXVisibility)
	w.U32(2)
	w.U8(1)
	s.Apply(w.Bytes())
	if s.Entities[2] != nil {
		t.Fatal("concealment resurrected unknown entity")
	}
	if _, err := s.Apply(w.Bytes()[:6]); err == nil {
		t.Fatal("accepted truncated visibility")
	}
}
