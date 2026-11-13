package proto

import "testing"

func TestSocialListsBoundedAndLatin1(t *testing.T) {
	for _, op := range []Opcode{FriendList, BlockList} {
		w := NewWriter(op)
		w.U8(1)
		w.U32(123)
		if op == FriendList {
			w.U8(4)
			w.U8(1)
		}
		w.Fixed("André", 21, true)
		w.Text("Tank")
		p := w.Bytes()
		rows, err := DecodeSocialList(p)
		if err != nil || len(rows) != 1 || rows[0].Name != "André" || rows[0].Memo != "Tank" {
			t.Fatal(rows, err)
		}
		for i := 2; i < len(p); i++ {
			if _, err := DecodeSocialList(p[:i]); err == nil {
				t.Fatalf("accepted truncation %d", i)
			}
		}
		if _, err := DecodeSocialList(append(p, 0)); err == nil {
			t.Fatal("accepted trailing data")
		}
	}
}
func TestMonsterSkillPacketKeepsAttackSlot(t *testing.T) {
	for _, op := range []Opcode{MobSkillUse, MobSkillDrain, MobSkillOther} {
		w := NewWriter(op)
		w.U8(0)
		w.U32(123)
		w.U32(456)
		if op != MobSkillOther {
			w.U8(2)
		}
		w.U16(53)
		w.U8(100)
		w.U16(3)
		w.U16(4)
		w.U16(5)
		if op == MobSkillDrain {
			w.U16(6)
			w.U16(7)
			w.U16(8)
		}
		r, err := DecodeSkillResult(w.Bytes())
		if err != nil || r.ID != 53 || r.Level != 100 || r.HP != 3 || r.Target != 456 || (op != MobSkillOther && r.AttackSlot != 2) {
			t.Fatal(r, err)
		}
	}
}
