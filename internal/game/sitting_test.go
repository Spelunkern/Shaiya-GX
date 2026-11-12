package game

import (
	"bytes"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"testing"
)

func TestSittingFollowsConfirmedMotion(t *testing.T) {
	s := New()
	s.SelfID = 1
	if !bytes.Equal(proto.MotionPacket(proto.MotionSitDown), []byte{6, 5, 1}) {
		t.Fatal("sit request wire format")
	}
	frame := func(id uint32, action uint8) []byte {
		w := proto.NewWriter(proto.CharacterMotion)
		w.U32(id)
		w.U8(action)
		return w.Bytes()
	}
	p := frame(2, proto.MotionSitDown)
	if _, err := s.Apply(p[:len(p)-1]); err == nil || len(s.Entities) != 0 {
		t.Fatal("truncated motion mutated state")
	}
	if _, err := s.Apply(p); err != nil {
		t.Fatal(err)
	}
	if !s.Entities[2].Sitting || s.Self.Sitting {
		t.Fatal("incorrect remote posture")
	}
	if _, err := s.Apply(frame(1, proto.MotionSitDown)); err != nil || !s.Self.Sitting {
		t.Fatal("self sit missing", err)
	}
	if _, err := s.Apply(frame(1, 5)); err != nil || !s.Self.Sitting {
		t.Fatal("gesture cleared sitting", err)
	}
	if _, err := s.Apply(frame(1, proto.MotionStandUp)); err != nil || s.Self.Sitting {
		t.Fatal("stand missing", err)
	}
}
