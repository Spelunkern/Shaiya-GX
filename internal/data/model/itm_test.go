package model

import "testing"

func FuzzITM(f *testing.F) {
	f.Add(append([]byte("ITM"), make([]byte, 12)...))
	f.Fuzz(func(t *testing.T, b []byte) { ParseITM(b) })
}

func TestIT2BindingsAndTruncation(t *testing.T) {
	b := make([]byte, 3+12+24+16*2*32)
	copy(b, "IT2")
	b[11] = 1                 // one record after empty mesh and texture lists
	b[39] = 7                 // first primary bone
	b[45], b[46] = 0x80, 0x3f // position X = 1
	b[69], b[70] = 0x80, 0x3f // rotation W = 1
	table, err := ParseITM(b)
	if err != nil {
		t.Fatal(err)
	}
	binding := table.Records[0].Bindings[0][0]
	if binding.Bone != 7 || binding.Position[0] != 1 || binding.Rotation[3] != 1 {
		t.Fatalf("bad IT2 binding: %+v", binding)
	}
	if _, err := ParseITM(b[:len(b)-1]); err == nil {
		t.Fatal("accepted truncated IT2 transform")
	}
}
