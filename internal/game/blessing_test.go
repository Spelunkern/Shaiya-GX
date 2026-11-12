package game

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

func blessingPacket(op proto.Opcode, country uint8, count, remaining uint32) []byte {
	w := proto.NewWriter(op)
	w.U8(country)
	w.U32(count)
	if op == proto.BlessInit {
		w.U32(remaining)
	}
	return w.Bytes()
}

func TestBlessingSnapshotAndBroadcast(t *testing.T) {
	s := New()
	s.Phase = PhaseLoading
	apply := func(p []byte) {
		t.Helper()
		if _, err := s.Apply(p); err != nil {
			t.Fatal(err)
		}
	}
	apply(blessingPacket(proto.BlessInit, 0, 1200, 500))
	apply(blessingPacket(proto.BlessInit, 0, 0, 0)) // Rapture placeholder during loading
	if s.Blessing.Count != 1200 || s.Blessing.RemainingMS != 500 {
		t.Fatal("placeholder replaced real snapshot", s.Blessing)
	}
	apply(blessingPacket(proto.CountryKillCount, 0, 0, 0))
	if s.Blessing.Count != 0 || s.Blessing.RemainingMS != 0 {
		t.Fatal("confirmed reset ignored")
	}
	apply(blessingPacket(proto.BlessInit, 0, 0, 0))
	s.Phase = PhaseInWorld
	apply(blessingPacket(proto.BlessInit, 0, 250, 0))
	apply(blessingPacket(proto.BlessInit, 0, 0, 0))
	if s.Blessing.Count != 0 {
		t.Fatal("later authoritative snapshot ignored")
	}
	before := s.Blessing
	if _, err := s.Apply([]byte{0x11, 0x02, 0}); err == nil {
		t.Fatal("truncated packet accepted")
	}
	if s.Blessing != before {
		t.Fatal("truncated packet changed blessing")
	}
}
