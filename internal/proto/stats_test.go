package proto

import (
	"bytes"
	"testing"
)

func TestStatPointWireOrder(t *testing.T) {
	d := StatPoints{1, 0x1234, 3, 4, 5, 6}
	want := []byte{8, 2, 1, 0, 0x34, 0x12, 3, 0, 4, 0, 5, 0, 6, 0}
	if got := StatPointsPacket(d); !bytes.Equal(got, want) {
		t.Fatalf("allocation bytes %x", got)
	}
	if got, err := DecodeStatPoints(want); err != nil || got != d {
		t.Fatal(got, err)
	}
	for i := range want {
		if _, err := DecodeStatPoints(want[:i]); err == nil {
			t.Fatal("truncated acknowledgement", i)
		}
	}
	if (StatPoints{65535, 65535, 65535, 65535, 65535, 65535}).Total() != 393210 {
		t.Fatal("allocation total overflowed")
	}
}

func TestAdditionalStatsAndResetOrders(t *testing.T) {
	w := NewWriter(CharacterAdditionalStats)
	for i := uint32(1); i <= 12; i++ {
		w.U32(i)
	}
	s, err := DecodeAdditionalStats(w.Bytes())
	if err != nil || s.Str != 1 || s.Rec != 2 || s.Dex != 5 || s.MaxAttack != 8 || s.Resistance != 12 {
		t.Fatal(s, err)
	}
	for i := range w.Bytes() {
		if _, err := DecodeAdditionalStats(w.Bytes()[:i]); err == nil {
			t.Fatal("truncated totals", i)
		}
	}
	w = NewWriter(StatsReset)
	w.U8(1)
	for i := uint16(1); i <= 7; i++ {
		w.U16(i)
	}
	r, err := DecodeStatReset(w.Bytes())
	if err != nil || r.Points != 1 || r.Str != 2 || r.Rec != 3 || r.Dex != 6 || r.Luc != 7 {
		t.Fatal(r, err)
	}
	for i := range w.Bytes() {
		if _, err := DecodeStatReset(w.Bytes()[:i]); err == nil {
			t.Fatal("truncated successful reset", i)
		}
	}
	if r, err := DecodeStatReset([]byte{0x14, 2, 0}); err != nil || r.Result != 0 {
		t.Fatal("native reset refusal is a single byte", r, err)
	}
}

func TestMaxEnergyWireOrder(t *testing.T) {
	p := []byte{0x0b, 5, 4, 3, 2, 1, 2, 0x78, 0x56, 0x34, 0x12}
	if m, err := DecodeMaxEnergy(p); err != nil || m.ID != 0x01020304 || m.Kind != 2 || m.Value != 0x12345678 {
		t.Fatal(m, err)
	}
	for i := range p {
		if _, err := DecodeMaxEnergy(p[:i]); err == nil {
			t.Fatal("truncated maximum", i)
		}
	}
}
