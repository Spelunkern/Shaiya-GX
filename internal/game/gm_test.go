package game

import (
	"encoding/binary"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"testing"
)

func TestGMRoleIsAccountMetadataWithoutChangingNames(t *testing.T) {
	s := New()
	s.SelfID = 1
	enter := func(id uint32, marker uint8) {
		w := proto.NewWriter(proto.CharacterEnteredMap)
		w.U32(id)
		w.U8(marker)
		w.U16(0)
		w.F32(1)
		w.F32(2)
		w.F32(3)
		w.U32(0)
		w.U32(0)
		if _, err := s.Apply(w.Bytes()); err != nil {
			t.Fatal(err)
		}
	}
	enter(1, 2)
	enter(2, 2)
	enter(3, 0)
	if !s.Self.IsGM || !s.Entities[2].IsGM || s.Entities[3].IsGM {
		t.Fatal("GM enter metadata lost")
	}
	packet := func(id uint32, gm, turbo byte) []byte {
		p := []byte{1, 0x7f}
		p = binary.LittleEndian.AppendUint32(p, id)
		p = append(p, gm, turbo)
		return append(p, 0, 0)
	}
	if _, err := s.Apply(packet(1, 1, 1)); err != nil || !s.GMTurbo {
		t.Fatal("turbo not enabled", err)
	}
	if _, err := s.Apply(packet(3, 0, 0)); err != nil || !s.GMTurbo {
		t.Fatal("other character changed own turbo")
	}
	for _, p := range [][]byte{packet(1, 0, 1), packet(1, 2, 1), packet(1, 1, 1)[:9]} {
		if _, err := s.Apply(p); err == nil || !s.GMTurbo {
			t.Fatal("invalid packet accepted")
		}
	}
	enter(1, 0)
	if s.Self.IsGM || s.GMTurbo {
		t.Fatal("normal role retained GM state")
	}
	enter(2, 0)
	if s.Entities[2].IsGM {
		t.Fatal("remote role did not update")
	}
}
