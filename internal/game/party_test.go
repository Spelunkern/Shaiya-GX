package game

import (
	"reflect"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

func TestPartyAcknowledgedRosterLeaderLeaveAndMalformedAtomicity(t *testing.T) {
	s := New()
	s.SelfID = 1
	s.Slots[0] = proto.CharacterSlot{ID: 1, Name: "Self", Level: 4, Class: 0}
	s.Pools = proto.Pools{HP: 100, MP: 50, SP: 70}
	s.Details = proto.Details{MaxHP: 200, MaxMP: 60, MaxSP: 80}
	apply := func(w *proto.Writer) {
		t.Helper()
		if _, err := s.Apply(w.Bytes()); err != nil {
			t.Fatal(err)
		}
	}
	roster := proto.NewWriter(proto.PartyRoster)
	roster.U8(0)
	roster.U8(0)
	apply(roster)
	if s.Party.Leader != 1 || len(s.Party.Members) != 1 || s.Party.Members[0].Name != "Self" {
		t.Fatal(s.Party)
	}
	joined := proto.NewWriter(proto.PartyJoined)
	joined.U32(2)
	joined.Fixed("Other", 21, true)
	joined.U16(5)
	joined.U8(3)
	for _, n := range []uint32{150, 300, 40, 90, 50, 100} {
		joined.U32(n)
	}
	joined.U16(0)
	joined.F32(1)
	joined.F32(2)
	joined.F32(3)
	joined.U8(0)
	before := append([]proto.PartyMember(nil), s.Party.Members...)
	for n := 0; n < len(joined.Bytes()); n++ {
		if _, err := s.Apply(joined.Bytes()[:n]); err == nil && n >= 2 {
			t.Fatal("malformed join accepted", n)
		}
		if !reflect.DeepEqual(before, s.Party.Members) {
			t.Fatal("malformed packet changed party")
		}
	}
	apply(joined)
	apply(joined)
	if len(s.Party.Members) != 2 {
		t.Fatal("duplicated join", s.Party)
	}
	pool := proto.NewWriter(proto.PartyPools)
	pool.U32(2)
	pool.U32(123)
	pool.U32(456)
	pool.U32(789)
	apply(pool)
	if m := s.Party.Member(2); m.HP != 123 || m.SP != 456 || m.MP != 789 {
		t.Fatal("pool wire order", m)
	}
	leader := proto.NewWriter(proto.PartyLeader)
	leader.U32(2)
	apply(leader)
	if s.Party.Leader != 2 {
		t.Fatal("leader did not change")
	}
	left := proto.NewWriter(proto.PartyLeft)
	left.U32(2)
	apply(left)
	if len(s.Party.Members) != 1 {
		t.Fatal("local party dissolved ahead of server")
	}
	left = proto.NewWriter(proto.PartyLeft)
	left.U32(1)
	apply(left)
	if len(s.Party.Members) != 0 || s.Party.Leader != 0 {
		t.Fatal("self departure left stale party")
	}
}
