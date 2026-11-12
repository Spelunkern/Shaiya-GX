package game

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

func TestAdditionalSkillResultsDoNotPredictResources(t *testing.T) {
	s := New()
	s.SelfID = 1
	s.Pools = proto.Pools{HP: 100, SP: 80, MP: 60}
	for _, op := range []proto.Opcode{proto.SkillUserDrain, proto.SkillMobDrain, proto.SkillPartyMember, proto.SkillMobOther} {
		w := proto.NewWriter(op)
		w.U8(1)
		w.U32(1)
		w.U32(3)
		w.U16(55)
		w.U8(2)
		w.U16(10)
		w.U16(11)
		w.U16(12)
		if op == proto.SkillUserDrain || op == proto.SkillMobDrain {
			w.U16(4)
			w.U16(5)
			w.U16(6)
		}
		p := w.Bytes()
		before := s.LastSkill
		if _, err := s.Apply(p[:len(p)-1]); err == nil {
			t.Fatalf("%x accepted truncated result", op)
		}
		if s.LastSkill != before {
			t.Fatal("truncated result changed state")
		}
		if _, err := s.Apply(p); err != nil {
			t.Fatalf("%x: %v", op, err)
		}
		if s.LastSkill.HP != 10 || s.LastSkill.SP != 11 || s.LastSkill.MP != 12 {
			t.Fatal("incorrect result", s.LastSkill)
		}
		if (op == proto.SkillUserDrain || op == proto.SkillMobDrain) && (s.LastSkill.HealedHP != 4 || s.LastSkill.HealedSP != 5 || s.LastSkill.HealedMP != 6) {
			t.Fatal("drain healing lost", s.LastSkill)
		}
	}
	for _, op := range []proto.Opcode{proto.SkillUserExtraDamage, proto.SkillExtraDamage} {
		w := proto.NewWriter(op)
		w.U32(1)
		w.U32(3)
		w.U16(55)
		w.U8(2)
		w.U16(13)
		w.U16(14)
		w.U16(15)
		if _, err := s.Apply(w.Bytes()); err != nil {
			t.Fatal(err)
		}
		if s.LastSkillExtra.HP != 13 || s.LastSkillExtra.Target != 3 {
			t.Fatal("extra damage lost")
		}
	}
	for _, op := range []proto.Opcode{proto.SkillUserReflectedDamage, proto.SkillMobReflectedDamage} {
		w := proto.NewWriter(op)
		w.U32(3)
		w.U32(1)
		w.U16(8)
		w.U16(7)
		w.U16(6)
		if _, err := s.Apply(w.Bytes()); err != nil {
			t.Fatal(err)
		}
		if s.LastReflectedDamage.HP != 8 || s.LastReflectedDamage.Target != 1 {
			t.Fatal("reflected hit lost")
		}
	}
	if s.Pools != (proto.Pools{HP: 100, SP: 80, MP: 60}) {
		t.Fatal("skill presentation altered authoritative pools", s.Pools)
	}
}
