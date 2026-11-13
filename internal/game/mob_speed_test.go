package game

import (
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"testing"
)

func TestMobSpeedSnapshotAndUpdates(t *testing.T) {
	s := New()
	s.Entities[7] = &Entity{ID: 7, Kind: KindMob}
	w := proto.NewWriter(proto.TargetMobGetState)
	w.U32(7)
	w.U32(123)
	w.U8(3)
	w.U8(1)
	if _, err := s.Apply(w.Bytes()); err != nil {
		t.Fatal(err)
	}
	e := s.Entities[7]
	if e.HP != 123 || !e.MobSpeedKnown || e.MobMoveSpeed != 1 {
		t.Fatal(e)
	}
	w = proto.NewWriter(proto.MobSpeed)
	w.U32(7)
	w.U8(255)
	w.U8(255)
	if _, err := s.Apply(w.Bytes()); err != nil {
		t.Fatal(err)
	}
	if e.MobAttackSpeed != 255 || e.MobMoveSpeed != 255 {
		t.Fatal(e)
	}
	before := *e
	if _, err := s.Apply(w.Bytes()[:6]); err == nil || *e != before {
		t.Fatal("truncated speed mutated entity")
	}
}
