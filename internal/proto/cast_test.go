package proto

import "testing"

func TestSkillCastWire(t *testing.T) {
	p := []byte{0x10, 5, 42, 0, 0, 0, 99, 0, 0, 0, 37, 0, 1}
	m, err := DecodeSkillCast(p)
	if err != nil || m != (SkillCast{Source: 42, Target: 99, ID: 37, Level: 1}) {
		t.Fatalf("cast=%+v %v", m, err)
	}
	for n := 0; n < len(p); n++ {
		if _, err := DecodeSkillCast(p[:n]); err == nil {
			t.Fatalf("accepted %d bytes", n)
		}
	}
}
