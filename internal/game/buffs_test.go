package game

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

func TestBuffInstancesRefreshAndRemovalStayAuthoritative(t *testing.T) {
	s := New()
	add := func(id uint32, level uint8) {
		t.Helper()
		w := proto.NewWriter(proto.BuffAdd)
		w.U32(id)
		w.U16(31)
		w.U8(level)
		if _, err := s.Apply(w.Bytes()); err != nil {
			t.Fatal(err)
		}
	}
	add(9, 1)
	add(2, 1)
	add(9, 2)
	if len(s.Buffs) != 2 || s.Buffs[0].Instance != 2 || s.Buffs[1].Level != 2 {
		t.Fatal("instance refresh duplicated/reordered effects", s.Buffs)
	}
	w := proto.NewWriter(proto.CharacterActiveBuffs)
	w.U8(1)
	w.U32(2)
	w.U16(31)
	w.U8(1)
	w.U32(0)
	if _, err := s.Apply(w.Bytes()); err != nil {
		t.Fatal(err)
	}
	if len(s.Buffs) != 1 || !s.Buffs[0].HasRemaining || s.Buffs[0].Remaining != 0 {
		t.Fatal("zero remaining snapshot must retain the server's effect", s.Buffs)
	}
	w = proto.NewWriter(proto.BuffRemove)
	w.U32(2)
	if _, err := s.Apply(w.Bytes()); err != nil || len(s.Buffs) != 0 {
		t.Fatal("server removal did not clear effect", err, s.Buffs)
	}
}
