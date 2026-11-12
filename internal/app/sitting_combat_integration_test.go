//go:build integration

package app

import (
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/testworld"
)

func TestSittingCombatAgainstRapture(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_SITTING") != "1" || os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires disposable accounts and GL")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	window, err := platform.Open(platform.Config{Title: "sitting combat", Width: 512, Height: 384, Hidden: true})
	if err != nil {
		t.Fatal(err)
	}
	defer window.Close()
	if err := render.Init(true); err != nil {
		t.Fatal(err)
	}
	arc, err := archive.Open(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	st, world, send, wait := testworld.Enter(t)
	other, _, sendOther, waitOther := testworld.EnterAs(t, os.Getenv("SHAIYA_TEST_PARTNER"), os.Getenv("SHAIYA_TEST_PARTNER_PASSWORD"))
	wait(func() bool { return st.Entities[other.SelfID] != nil })
	waitOther(func() bool { return other.Entities[st.SelfID] != nil })
	send(proto.DuelRequestPacket(other.SelfID))
	waitOther(func() bool { return other.Duel.Phase == game.DuelIncoming })
	sendOther(proto.DuelAnswerPacket(true))
	wait(func() bool { return st.Duel.Phase == game.DuelWager })
	waitOther(func() bool { return other.Duel.Phase == game.DuelWager })
	send(proto.DuelReadyPacket(0))
	sendOther(proto.DuelReadyPacket(0))
	wait(func() bool { return st.Duel.Phase == game.DuelFighting })
	waitOther(func() bool { return other.Duel.Phase == game.DuelFighting })
	a, err := New(Config{}, arc)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	a.state, a.world, a.screen = st, world, screenWorld
	a.enterWorld()
	pump := func(cond func() bool) {
		t.Helper()
		end := time.Now().Add(12 * time.Second)
		for !cond() {
			a.pumpWorld()
			if a.state == nil || time.Now().After(end) {
				t.Fatal("confirmation timed out")
			}
			time.Sleep(time.Millisecond)
		}
	}
	a.updateWorld(&platform.Input{Chars: []rune{'c'}}, .01)
	pump(func() bool { return st.Self.Sitting && !a.wv.sitPending })
	st.Target = other.SelfID
	if !a.attackTarget() || a.wv.sitAttack != other.SelfID || !st.Self.Sitting {
		t.Fatal("attack was not deferred while seated")
	}
	pump(func() bool { return !st.Self.Sitting && !a.wv.sitPending })
	for i := 0; i < 600 && a.wv.sitAttack != 0; i++ {
		a.updateWorld(&platform.Input{}, 1.0/60)
	}
	if a.wv.sitAttack != 0 {
		t.Fatal("deferred attack did not resume")
	}
	pump(func() bool { return st.LastAttack.Source == st.SelfID && st.LastAttack.Target == other.SelfID })
	a.updateWorld(&platform.Input{Chars: []rune{'c'}}, .01)
	pump(func() bool { return st.Self.Sitting && !a.wv.sitPending })
	// The echo is ordered after the stop request. Subsequent swings would prove
	// that the server continued attacking while the character was sitting.
	deadline := time.NewTimer(3 * time.Second)
	defer deadline.Stop()
	for {
		select {
		case p, ok := <-world.Packets:
			if !ok {
				t.Fatal("world disconnected")
			}
			if proto.Op(p) == proto.CharacterUserAutoAttack {
				r, e := proto.DecodeAttackResult(p)
				if e != nil {
					t.Fatal(e)
				}
				if r.Source == st.SelfID {
					t.Fatal("server continued attacking while seated")
				}
			}
			if _, e := st.Apply(p); e != nil {
				t.Fatal(e)
			}
		case <-deadline.C:
			t.Log("deferred attack resumed after standing; auto-attack stopped before sitting")
			return
		}
	}
}
