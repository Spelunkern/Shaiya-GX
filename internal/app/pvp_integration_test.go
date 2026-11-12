//go:build integration

package app

import (
	"os"
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/testworld"
)

func TestPvPCountsAtWorldEntryAgainstRapture(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_PVP_COUNTS") != "1" {
		t.Skip("requires disposable account")
	}
	st, _, _, wait := testworld.Enter(t)
	wait(func() bool { return st.PvPCountsKnown == [4]bool{true, true, true, true} && st.Blessing.Known })
	if st.Unknown[proto.UserKillCountUpdate] != 0 || !st.Self.PvPKillsKnown || st.Self.PvPKills != st.Details.Kills {
		t.Fatal("initial PvP counts were not applied")
	}
	t.Logf("server PvP counters: %d kills, %d deaths, %d victories, %d defeats", st.Details.Kills, st.Details.Deaths, st.Details.Wins, st.Details.Losses)
	t.Logf("server faction blessing: faction=%d count=%d reset_ms=%d", st.Blessing.Country, st.Blessing.Count, st.Blessing.RemainingMS)
}

// n2_smoke.py puts two disposable accounts of opposing factions together on
// an attackable map. Rapture remains authoritative for hostility and damage.
func TestFactionPvPAgainstRapture(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_PARTNER") == "" {
		t.Skip("requires two disposable accounts")
	}
	a, wa, sendA, waitA := testworld.Enter(t)
	b, wb, sendB, waitB := testworld.EnterAs(t, os.Getenv("SHAIYA_TEST_PARTNER"), os.Getenv("SHAIYA_TEST_PARTNER_PASSWORD"))
	appA, appB := inputApp(), inputApp()
	appA.state, appA.world = a, wa
	appB.state, appB.world = b, wb
	waitA(func() bool { return a.Entities[b.SelfID] != nil })
	waitB(func() bool { return b.Entities[a.SelfID] != nil })
	if a.Duel.Phase != game.DuelNone || b.Duel.Phase != game.DuelNone {
		t.Fatal("fixture unexpectedly in a duel")
	}
	initialHP := b.Pools.HP
	a.Target = b.SelfID
	appA.updateCombat(&platform.Input{Chars: []rune{'f'}}, 1024, 768)
	deadline := time.Now().Add(20 * time.Second)
	for a.LastAttack.Source != a.SelfID || a.LastAttack.Target != b.SelfID || (b.Pools.HP >= initialHP && !b.Self.Dead) {
		appA.pumpWorld()
		appB.pumpWorld()
		if appA.state == nil || appB.state == nil {
			t.Fatal("disconnected during PvP")
		}
		if time.Now().After(deadline) {
			t.Fatalf("PvP timed out: last attack %+v, target HP %d/%d dead=%v", a.LastAttack, b.Pools.HP, initialHP, b.Self.Dead)
		}
		time.Sleep(10 * time.Millisecond)
	}
	sendA(proto.StopAttackPacket())
	if b.Self.Dead {
		sendB(proto.RevivePacket())
		waitB(func() bool { return !b.Self.Dead })
	}
	t.Logf("opposing-faction PvP confirmed: attacker %d, target %d, target HP %d/%d", a.SelfID, b.SelfID, b.Pools.HP, initialHP)
}
