package game

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

func TestPvPCountsFollowServerAndRejectTruncation(t *testing.T) {
	s := New()
	s.SelfID = 100
	for i, want := range []uint32{12, 34, 56, 78} {
		w := proto.NewWriter(proto.UserKillCountUpdate)
		w.U8(uint8(i))
		w.U32(want)
		p := w.Bytes()
		if _, err := s.Apply(p[:len(p)-1]); err == nil {
			t.Fatal("truncated count accepted")
		}
		if s.PvPCountsKnown[i] {
			t.Fatal("invalid count was marked known")
		}
		if _, err := s.Apply(p); err != nil {
			t.Fatal(err)
		}
	}
	if s.Details.Kills != 12 || s.Details.Deaths != 34 || s.Details.Wins != 56 || s.Details.Losses != 78 {
		t.Fatalf("counts: %+v", s.Details)
	}
	s.Entities[200] = &Entity{Kind: KindUser, ID: 200, Shape: &proto.UserShape{ID: 200, Kills: 3}}
	w := proto.NewWriter(proto.CharacterPvPKills)
	w.U32(200)
	w.U32(44)
	p := w.Bytes()
	if _, err := s.Apply(p[:len(p)-1]); err == nil {
		t.Fatal("truncated visible count accepted")
	}
	if s.Entities[200].Shape.Kills != 3 {
		t.Fatal("invalid packet mutated shape")
	}
	if _, err := s.Apply(p); err != nil {
		t.Fatal(err)
	}
	if s.Entities[200].Shape.Kills != 44 || s.Entities[200].PvPKills != 44 {
		t.Fatal("visible kill count not updated")
	}
	if _, err := s.Apply(append(p, 0)); err == nil {
		t.Fatal("count with trailing data accepted")
	}
	w = proto.NewWriter(proto.CharacterPvPKills)
	w.U32(100)
	w.U32(13)
	if _, err := s.Apply(w.Bytes()); err != nil {
		t.Fatal(err)
	}
	if s.Details.Kills != 13 || s.Self.PvPKills != 13 {
		t.Fatal("self kill count not updated")
	}
}
