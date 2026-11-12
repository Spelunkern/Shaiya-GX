package app

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

func TestRemotePositionInterpolatesAndRetargets(t *testing.T) {
	e := &game.Entity{X: 10, Y: 3, Z: 20}
	var p remotePosition
	if got := p.update(e, 0); got != (render.V3{10, 3, 20}) {
		t.Fatal(got)
	}
	e.X, e.MoveSerial = 14, 1
	if got := p.update(e, 1); got[0] != 10 {
		t.Fatal(got)
	}
	if got := p.update(e, 1.25); got[0] != 12 {
		t.Fatal(got)
	}
	e.X, e.MoveSerial = 16, 2
	if got := p.update(e, 1.25); got[0] != 12 {
		t.Fatal(got)
	}
	if got := p.update(e, 1.375); got[0] != 14 {
		t.Fatal(got)
	}
	if got := p.update(e, 1.5); got[0] != 16 {
		t.Fatal(got)
	}
	if e.X != 16 {
		t.Fatal("presentation changed server state")
	}
}

func TestMonsterMovementUsesConfirmedPacketsForVisualInterpolation(t *testing.T) {
	s := game.New()
	spawn := proto.NewWriter(proto.MobEnter)
	spawn.U32(42)
	spawn.Bool(false)
	spawn.U16(1)
	spawn.F32(10)
	spawn.F32(20)
	if _, err := s.Apply(spawn.Bytes()); err != nil {
		t.Fatal(err)
	}
	e := s.Entities[42]
	var p remotePosition
	p.update(e, 0)
	move := proto.NewWriter(proto.MobMove)
	move.U32(42)
	move.U8(1)
	move.F32(14)
	move.F32(20)
	if _, err := s.Apply(move.Bytes()); err != nil {
		t.Fatal(err)
	}
	if e.MoveSerial != 1 || e.X != 14 {
		t.Fatal("server movement not applied")
	}
	if got := p.update(e, 1); got[0] != 10 {
		t.Fatal(got)
	}
	if got := p.update(e, 1.25); got[0] != 12 {
		t.Fatal(got)
	}
	if e.X != 14 {
		t.Fatal("visual movement changed authoritative position")
	}
	if _, err := s.Apply(move.Bytes()[:5]); err == nil || e.MoveSerial != 1 {
		t.Fatal("truncated movement changed state")
	}
}

func TestRemotePositionSnapsOnResetOrLargeJump(t *testing.T) {
	e := &game.Entity{}
	var p remotePosition
	p.update(e, 0)
	e.X, e.MoveSerial = 21, 1
	if got := p.update(e, 1); got[0] != 21 {
		t.Fatal(got)
	}
	e.Y, e.MoveSerial = 9, 2
	if got := p.update(e, 2); got[1] != 9 {
		t.Fatal(got)
	}
	e.X, e.PositionResetSerial = 22, 1
	if got := p.update(e, 3); got[0] != 22 {
		t.Fatal(got)
	}
}
