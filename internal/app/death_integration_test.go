//go:build integration

package app

import (
	"os"
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/testworld"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

func TestDeathDialogAgainstRapture(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_DEATH") != "1" || os.Getenv("SHAIYA_TEST_PARTNER") == "" {
		t.Skip("requires disposable duel fixtures")
	}
	a, _, sendA, waitA := testworld.Enter(t)
	b, wb, sendB, waitB := testworld.EnterAs(t, os.Getenv("SHAIYA_TEST_PARTNER"), os.Getenv("SHAIYA_TEST_PARTNER_PASSWORD"))
	waitA(func() bool { return a.Entities[b.SelfID] != nil })
	waitB(func() bool { return b.Entities[a.SelfID] != nil })
	sendA(proto.DuelRequestPacket(b.SelfID))
	waitB(func() bool { return b.Duel.Phase == game.DuelIncoming })
	sendB(proto.DuelAnswerPacket(true))
	waitA(func() bool { return a.Duel.Phase == game.DuelWager })
	waitB(func() bool { return b.Duel.Phase == game.DuelWager })
	sendA(proto.DuelReadyPacket(0))
	waitA(func() bool { return a.Duel.OwnReady })
	sendB(proto.DuelReadyPacket(0))
	waitA(func() bool { return a.Duel.Phase == game.DuelFighting })
	waitB(func() bool { return b.Duel.Phase == game.DuelFighting })
	sendA(proto.UserAttackPacket(b.SelfID))
	waitB(func() bool { return b.Self.Dead })
	app := inputApp()
	app.state, app.world = b, wb
	app.items = map[sdata.ItemKey]*sdata.Item{}
	if os.Getenv("SHAIYA_TEST_DEATH_DUEL_RUNE") == "1" {
		arc, err := archive.Open(os.Getenv("SHAIYA_DATA"))
		if err != nil {
			t.Fatal(err)
		}
		defer arc.Close()
		raw, err := arc.ReadFile("Item/Item.SData")
		if err != nil {
			t.Fatal(err)
		}
		app.items, err = sdata.ParseItems(raw)
		if err != nil || !app.hasResurrectionRune() {
			t.Fatal("duel rune fixture missing", err)
		}
	}
	app.deathUI, _ = ui.NewDeathDialog(inputTextures{})
	app.clock = 1
	app.updateDeath(&platform.Input{}, 1024, 768)
	if !app.wv.death.initialized || app.wv.death.rune || app.wv.death.party {
		t.Fatalf("unexpected choices: %+v", app.wv.death)
	}
	x, y := 1024/2-128+96, 768/2-128+115
	app.updateDeath(&platform.Input{MouseX: x, MouseY: y, Pressed: true, MouseDown: true}, 1024, 768)
	app.updateDeath(&platform.Input{MouseX: x, MouseY: y, Clicked: true}, 1024, 768)
	if !b.Self.Dead {
		t.Fatal("predicted resurrection before confirmation")
	}
	deadline := time.Now().Add(10 * time.Second)
	for b.Self.Dead {
		app.pumpWorld()
		if app.state == nil || time.Now().After(deadline) {
			t.Fatal("revive timeout")
		}
		time.Sleep(time.Millisecond)
	}
	if b.Pools.HP == 0 {
		t.Fatal("revived without confirmed HP")
	}
	if os.Getenv("SHAIYA_TEST_DEATH_DUEL_RUNE") == "1" && !app.hasResurrectionRune() {
		t.Fatal("loss-free duel unexpectedly consumed a rune")
	}
	t.Log("bind-point choice sent via original dialog and confirmed by server")
}

func killDisposableWithMob(t *testing.T, st *game.State, send func([]byte), wait func(func() bool)) (float32, float32) {
	t.Helper()
	mob := testworld.LivingMob(t, st, send, wait)
	send(proto.MovePacket(0, 0, mob.X+1, mob.Y, mob.Z))
	send(proto.MobAttackPacket(mob.ID))
	wait(func() bool { return st.LastAttack.Source == st.SelfID || st.Self.Dead })
	send(proto.StopAttackPacket())
	wait(func() bool { return st.Self.Dead })
	return mob.X + 1, mob.Z
}

func clickDeathChoice(a *App, x int) {
	y := 768/2 - 128 + 115
	a.updateDeath(&platform.Input{MouseX: x, MouseY: y, Pressed: true, MouseDown: true}, 1024, 768)
	a.updateDeath(&platform.Input{MouseX: x, MouseY: y, Clicked: true}, 1024, 768)
}

func TestDeathRuneHereAgainstRapture(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_DEATH_RUNE") != "1" {
		t.Skip("requires disposable rune fixture")
	}
	st, w, send, wait := testworld.EnterAs(t, os.Getenv("SHAIYA_TEST_PARTNER"), os.Getenv("SHAIYA_TEST_PARTNER_PASSWORD"))
	arc, err := archive.Open(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	raw, err := arc.ReadFile("Item/Item.SData")
	if err != nil {
		t.Fatal(err)
	}
	defs, err := sdata.ParseItems(raw)
	if err != nil {
		t.Fatal(err)
	}
	var key [2]uint8
	found := false
	for k, it := range st.Inventory {
		if it.Type == 100 && it.TypeID == 44 && it.Count > 0 {
			key = k
			found = true
			break
		}
	}
	if !found {
		t.Fatal("rune fixture absent")
	}
	x, z := killDisposableWithMob(t, st, send, wait)
	before := st.Inventory[key].Count
	app := inputApp()
	app.state, app.world, app.items = st, w, defs
	app.deathUI, _ = ui.NewDeathDialog(inputTextures{})
	app.clock = 1
	app.updateDeath(&platform.Input{}, 1024, 768)
	if !app.wv.death.rune {
		t.Fatal("real rune did not enable choice")
	}
	clickDeathChoice(app, 1024/2-128+96)
	if !app.wv.death.choosingLocation {
		t.Fatal("rune location choice did not open")
	}
	clickDeathChoice(app, 1024/2-128+144)
	if !st.Self.Dead {
		t.Fatal("resurrection predicted before server confirmation")
	}
	deadline := time.Now().Add(15 * time.Second)
	for st.Self.Dead || st.Inventory[key].Count >= before {
		app.pumpWorld()
		if app.state == nil || time.Now().After(deadline) {
			t.Fatal("rune resurrection or consumption timeout")
		}
		time.Sleep(time.Millisecond)
	}
	if st.Self.X != x || st.Self.Z != z {
		t.Fatalf("rune here moved from (%v,%v) to (%v,%v)", x, z, st.Self.X, st.Self.Z)
	}
	t.Log("rune consumed and resurrection confirmed at the death position")
}

func TestDeathPartyAgainstRapture(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_DEATH_PARTY") != "1" {
		t.Skip("requires disposable party fixtures")
	}
	leader, _, sendA, waitA := testworld.Enter(t)
	st, w, sendB, waitB := testworld.EnterAs(t, os.Getenv("SHAIYA_TEST_PARTNER"), os.Getenv("SHAIYA_TEST_PARTNER_PASSWORD"))
	waitA(func() bool { return leader.Entities[st.SelfID] != nil })
	waitB(func() bool { return st.Entities[leader.SelfID] != nil })
	sendA(proto.PartyInvitePacket(st.SelfID))
	waitB(func() bool { return st.Party.InviteSerial > 0 })
	sendB(proto.PartyAnswerPacket(true, st.SelfID))
	waitB(func() bool { return st.Party.Leader == leader.SelfID && len(st.Party.Members) == 2 })
	waitA(func() bool { return len(leader.Party.Members) == 2 })
	_, _ = killDisposableWithMob(t, st, sendB, waitB)
	sendA(proto.MovePacket(0, 0, 780, 39, 1650))
	waitB(func() bool { e := st.Entities[leader.SelfID]; return e != nil && e.X == 780 && e.Z == 1650 })
	app := inputApp()
	app.state, app.world = st, w
	app.items = map[sdata.ItemKey]*sdata.Item{}
	app.deathUI, _ = ui.NewDeathDialog(inputTextures{})
	app.clock = 1
	app.updateDeath(&platform.Input{}, 1024, 768)
	if !app.wv.death.party {
		t.Fatal("party choice missing")
	}
	clickDeathChoice(app, 1024/2-128+129)
	if !app.wv.death.waiting || !st.Self.Dead {
		t.Fatal("party waiting state incorrect")
	}
	deadline := time.Now().Add(38 * time.Second)
	for st.Self.Dead {
		app.pumpWorld()
		if app.state == nil || time.Now().After(deadline) {
			t.Fatal("party revival timed out")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if st.Self.X != 780 || st.Self.Z != 1650 {
		t.Fatalf("revived away from party leader: %v,%v", st.Self.X, st.Self.Z)
	}
	t.Log("party resurrection arrived at leader position after server delay")
}
