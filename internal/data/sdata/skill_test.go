package sdata

import "testing"

func FuzzSkills(f *testing.F) {
	f.Add([]byte{0, 0, 0, 0})
	f.Fuzz(func(t *testing.T, b []byte) { ParseSkills(b) })
}

func TestSkillIDsAreOneBased(t *testing.T) {
	// One group, nine empty-string records with their declared levels.
	b := make([]byte, 4+9*124)
	b[0] = 1
	for i := 0; i < 9; i++ {
		b[4+i*124+8] = byte(i + 1)
	}
	b[12+47] = 24
	b[12+49] = 4
	b[12+3], b[12+5], b[12+6] = 100, 2, 37
	b[12+59], b[12+60], b[12+111] = 2, 0, 1
	defs, err := ParseSkills(b)
	if err != nil {
		t.Fatal(err)
	}
	if len(defs) != 9 || defs[SkillKey{1, 1}].Level != 1 {
		t.Fatal("group did not map to wire skill 1")
	}
	if _, exists := defs[SkillKey{0, 1}]; exists {
		t.Fatal("zero-based wire skill ID")
	}
	if d := defs[SkillKey{1, 1}]; d.AttackRange != 24 || d.Attribute != 4 || d.Animation != 100 || d.Toggle != 2 || d.Sound != 37 {
		t.Fatalf("EP5 visual fields: %+v", d)
	}
	if d := defs[SkillKey{1, 1}]; d.KeepTime != 2 || d.FixRange != 1 || d.KeepSeconds() != 7200 {
		t.Fatal("EP5 duration fields", d)
	}
}
