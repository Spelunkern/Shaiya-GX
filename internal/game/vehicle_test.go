package game

import (
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"testing"
)

func TestVehicleResultsAndRemoteShapes(t *testing.T) {
	s := New()
	s.SelfID = 7
	ack := func(result byte) []byte {
		w := proto.NewWriter(proto.UseVehicle)
		w.Raw([]byte{result, 1})
		return w.Bytes()
	}
	shape := func(id uint32, value byte) []byte {
		w := proto.NewWriter(proto.CharacterMountShape)
		w.U32(id)
		w.U8(value)
		return w.Bytes()
	}
	if _, err := s.Apply(ack(0)); err != nil || s.Self.Mounted {
		t.Fatal("refused mount changed state")
	}
	if _, err := s.Apply(ack(1)); err != nil || !s.Self.Mounted {
		t.Fatal("accepted mount missing")
	}
	if _, err := s.Apply(shape(7, 2)); err != nil || !s.Self.Mounted || s.Self.MountShape != 2 {
		t.Fatal("mount shape missing")
	}
	if _, err := s.Apply(shape(9, 3)); err != nil || !s.Entities[9].Mounted {
		t.Fatal("remote mount missing")
	}
	if _, err := s.Apply(shape(7, 0)); err != nil || s.Self.Mounted {
		t.Fatal("forced dismount missing")
	}
	if _, err := s.Apply(ack(1)); err != nil || !s.Self.Mounted {
		t.Fatal("second mount missing")
	}
	// Rapture can emit shape zero for a valid Range=0 mount. Its echo must
	// not undo the accepted toggle, although a later forced zero dismounts.
	if _, err := s.Apply(shape(7, 0)); err != nil || !s.Self.Mounted {
		t.Fatal("zero-shape echo undid accepted mount")
	}
	if _, err := s.Apply(ack(1)); err != nil || s.Self.Mounted {
		t.Fatal("dismount toggle missing")
	}
	if _, err := s.Apply(shape(7, 0)); err != nil || s.Self.Mounted {
		t.Fatal("dismount echo")
	}
}
