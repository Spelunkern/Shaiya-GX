package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"testing"
)

func TestRemoteSittingKeepsSeatedPoseWithoutMovement(t *testing.T) {
	e := &game.Entity{Sitting: true}
	if clip := remoteLocomotion(nil, e); clip != "012_sit" {
		t.Fatal("seated actor chose", clip)
	}
	e.Sitting = false
	if clip := remoteLocomotion(nil, e); clip != "000_normal" {
		t.Fatal("standing actor chose", clip)
	}
}

func TestAttackQueuesWhilePostureIsPending(t *testing.T) {
	a := inputApp()
	a.state.SelfID = 1
	a.state.Self.Sitting = true
	a.wv.sitPending = true
	a.state.Target = 2
	a.state.Entities[2] = &game.Entity{ID: 2, Kind: game.KindMob}
	if !a.attackTarget() || a.wv.sitAttack != 2 {
		t.Fatal("attack was not deferred until standing")
	}
	a.wv.sitAttack = 0
	a.state.Entities[2].Dead = true
	if !a.attackTarget() || a.wv.sitAttack != 0 {
		t.Fatal("dead target queued an attack")
	}
}
