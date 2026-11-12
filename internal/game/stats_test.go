package game

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

func TestStatAcknowledgementUpdatesOnlyOwnConfirmedBase(t *testing.T) {
	s := New()
	s.SelfID = 1
	s.Details.StatPoint = 10
	s.StatsKnown = true
	s.Slots[0] = proto.CharacterSlot{ID: 1, Str: 65535, Dex: 2, Rec: 3, Int: 4, Wis: 5, Luc: 6}
	s.Slots[1] = proto.CharacterSlot{ID: 2, Str: 99}
	d := proto.StatPoints{1, 2, 3, 1, 1, 1}
	if _, err := s.Apply(proto.StatPointsPacket(d)); err != nil {
		t.Fatal(err)
	}
	c := s.Slots[0]
	if s.Details.StatPoint != 1 || c.Str != 65535 || c.Dex != 4 || c.Rec != 6 || c.Int != 5 || c.Wis != 6 || c.Luc != 7 || s.Slots[1].Str != 99 || s.StatsKnown {
		t.Fatal("acknowledgement applied incorrectly", c, s.Details)
	}
	before := s.Slots[0]
	if _, err := s.Apply(proto.StatPointsPacket(d)[:5]); err == nil || s.Slots[0] != before {
		t.Fatal("corrupt packet changed stats")
	}
	for kind, want := range []uint32{100, 200, 300} {
		w := proto.NewWriter(proto.CharacterMaxEnergy)
		w.U32(1)
		w.U8(uint8(kind))
		w.U32(want)
		if _, err := s.Apply(w.Bytes()); err != nil {
			t.Fatal(err)
		}
	}
	if s.Details.MaxHP != 100 || s.Details.MaxSP != 200 || s.Details.MaxMP != 300 {
		t.Fatal("HP/SP/MP order changed", s.Details)
	}
}

func TestLevelUpAddsJobAttributeWithoutSpendingAllocation(t *testing.T) {
	for job := uint8(0); job < 6; job++ {
		s := New()
		s.SelfID = 1
		s.StatsKnown = true
		s.Slots[0] = proto.CharacterSlot{ID: 1, Level: 4, Class: job}
		w := proto.NewWriter(proto.LevelUpSelf)
		w.U32(1)
		w.U16(7)
		w.U16(15)
		w.U16(9)
		w.U32(100)
		w.U32(200)
		if _, err := s.Apply(w.Bytes()); err != nil {
			t.Fatal(err)
		}
		c := s.Slots[0]
		values := [6]uint16{c.Str, c.Rec, c.Dex, c.Luc, c.Int, c.Wis}
		for i, n := range values {
			if i == int(job) && n != 3 || i != int(job) && n != 0 {
				t.Fatal("job attribute mismatch", job, c)
			}
		}
		if s.Details.StatPoint != 15 || s.Details.SkillPoint != 9 || s.StatsKnown {
			t.Fatal("level-up allocation/totals mismatch")
		}
	}
}
