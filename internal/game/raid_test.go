package game

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

func raidRecord(w *proto.Writer, slot uint16, id uint32) {
	w.U16(slot)
	w.U32(id)
	w.Fixed("Member", 21, true)
	w.U16(10)
	w.U8(1)
	for i := 0; i < 6; i++ {
		w.U32(uint32(i+1) * 100)
	}
	w.U16(0)
	w.F32(1)
	w.F32(2)
	w.F32(3)
	w.U8(0)
}

func raidChunk(more bool, begin, end int) []byte {
	w := proto.NewWriter(proto.RaidRoster)
	if more {
		w.U8(1)
	} else {
		w.U8(0)
	}
	w.U8(0)
	w.U8(1)
	w.U16(3)
	w.U8(1)
	w.U8(1)
	w.U8(uint8(end - begin))
	for i := begin; i < end; i++ {
		raidRecord(w, uint16(i), uint32(i+1))
	}
	return w.Bytes()
}

func TestRaidChunkedRosterAndUpdates(t *testing.T) {
	s := New()
	s.SelfID = 7
	s.Party.Members = []proto.PartyMember{{ID: 90}}
	if _, err := s.Apply(raidChunk(true, 0, 6)); err != nil {
		t.Fatal(err)
	}
	if len(s.Party.Members) != 1 || s.Party.Members[0].ID != 90 {
		t.Fatal("partial snapshot exposed")
	}
	if _, err := s.Apply(raidChunk(false, 6, 7)); err != nil {
		t.Fatal(err)
	}
	if !s.Party.Raid || len(s.Party.Members) != 7 || s.Party.Leader != 1 || s.Party.RaidSubLeader != 2 || !s.Party.RaidAutoJoin || s.Party.RaidDivision != 3 {
		t.Fatalf("roster %+v", s.Party)
	}
	swap := proto.NewWriter(proto.RaidSwap)
	swap.U32(0)
	swap.U32(9)
	swap.U32(9)
	swap.U32(1)
	if _, err := s.Apply(swap.Bytes()); err != nil {
		t.Fatal(err)
	}
	if s.Party.RaidSlots[9] != 1 || s.Party.RaidSlots[0] != 0 || s.Party.Leader != 1 {
		t.Fatal("swap lost slot or leader", s.Party)
	}
	pools := proto.NewWriter(proto.RaidPools)
	pools.U32(2)
	pools.U32(111)
	pools.U32(222)
	pools.U32(333)
	if _, err := s.Apply(pools.Bytes()); err != nil {
		t.Fatal(err)
	}
	if m := s.Party.Member(2); m.HP != 111 || m.SP != 222 || m.MP != 333 {
		t.Fatal("raid pools", m)
	}
	if _, err := s.Apply(proto.RaidExpelPacket(2)); err != nil {
		t.Fatal(err)
	}
	if s.Party.Member(2) != nil || s.Party.RaidSlots[1] != 0 {
		t.Fatal("expelled member retained")
	}
	leave := proto.NewWriter(proto.PartyLeaveAlt)
	leave.U32(7)
	if _, err := s.Apply(leave.Bytes()); err != nil {
		t.Fatal(err)
	}
	if s.Party.Raid || len(s.Party.Members) != 0 {
		t.Fatal("self departure retained raid")
	}
}

func TestRaidRejectsMalformedRosterWithoutPartialCommit(t *testing.T) {
	s := New()
	s.Party.Members = []proto.PartyMember{{ID: 99}}
	if _, err := s.Apply(raidChunk(true, 0, 6)); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Apply(raidChunk(false, 0, 1)); err == nil {
		t.Fatal("duplicate member accepted")
	}
	if len(s.Party.Members) != 1 || s.Party.Members[0].ID != 99 || s.Party.raidPending != nil {
		t.Fatal("bad snapshot modified party")
	}
	bad := raidChunk(false, 0, 1)
	bad = bad[:len(bad)-1]
	if _, err := s.Apply(bad); err == nil {
		t.Fatal("truncated member accepted")
	}
}
