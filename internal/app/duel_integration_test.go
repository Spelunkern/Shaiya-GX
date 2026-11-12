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
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

func TestDuelPhasesAgainstRapture(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_PARTNER") == "" {
		t.Skip("requires two disposable accounts")
	}
	a, wa, _, waitA := testworld.Enter(t)
	b, wb, _, waitB := testworld.EnterAs(t, os.Getenv("SHAIYA_TEST_PARTNER"), os.Getenv("SHAIYA_TEST_PARTNER_PASSWORD"))
	a1, a2 := inputApp(), inputApp()
	a1.state, a1.world = a, wa
	a2.state, a2.world = b, wb
	a1.duelUI, _ = ui.NewDuelPanel(inputTextures{})
	a2.duelUI, _ = ui.NewDuelPanel(inputTextures{})
	a1.duelMessage, _ = ui.NewMessage(inputTextures{})
	a2.duelMessage, _ = ui.NewMessage(inputTextures{})
	a2.exchangeMoney, _ = ui.NewExchangeMoney(inputTextures{})
	waitA(func() bool { e := a.Entities[b.SelfID]; return e != nil && e.Kind == game.KindUser })
	waitB(func() bool { e := b.Entities[a.SelfID]; return e != nil && e.Kind == game.KindUser })
	moneyA, moneyB := a.Details.Money, b.Details.Money
	a.Target = b.SelfID
	a1.worldControls(&platform.Input{MouseX: 300, MouseY: 20, RightPressed: true, RightDown: true}, 1024, 768)
	if !a1.playerContext.open {
		t.Fatal("context menu missing")
	}
	cx, cy := a1.playerContext.x, a1.playerContext.y
	a1.worldControls(&platform.Input{MouseX: cx + 10, MouseY: cy + 2*contextRowHeight + 5, Pressed: true, MouseDown: true}, 1024, 768)
	a1.worldControls(&platform.Input{MouseX: cx + 10, MouseY: cy + 2*contextRowHeight + 5, Clicked: true}, 1024, 768)
	waitA(func() bool { return a.Duel.Phase == game.DuelOutgoing })
	waitB(func() bool { return b.Duel.Phase == game.DuelIncoming })
	if !a2.updateDuelInvite(&platform.Input{}, 1024, 768) || !a2.duelMessage.Visible {
		t.Fatal("duel invitation missing")
	}
	a2.updateDuelInvite(&platform.Input{Keys: []platform.Key{platform.KeyEscape}}, 1024, 768)
	waitA(func() bool { return a.Duel.Phase == game.DuelNone && a.Duel.Result.Result == 0 })
	if b.Duel.Phase != game.DuelNone {
		t.Fatal("declined invitation still active")
	}
	if !a1.duelCommand("/duel") {
		t.Fatal("duel request not handled")
	}
	waitA(func() bool { return a.Duel.Phase == game.DuelOutgoing })
	waitB(func() bool { return b.Duel.RequestSerial >= 2 && b.Duel.Phase == game.DuelIncoming })
	a2.updateDuelInvite(&platform.Input{}, 1024, 768)
	a2.updateDuelInvite(&platform.Input{Keys: []platform.Key{platform.KeyEnter}}, 1024, 768)
	waitA(func() bool { return a.Duel.Phase == game.DuelWager })
	waitB(func() bool { return b.Duel.Phase == game.DuelWager })
	var key [2]uint8
	var wager proto.InventoryItem
	found := false
	for k, it := range a.Inventory {
		if k[0] == 1 && it.Type == 25 && it.TypeID == 4 && it.Count > 1 {
			key, wager, found = k, it, true
			break
		}
	}
	if !found || b.Details.Money < 1 {
		t.Fatal("wager fixture unavailable")
	}
	a1.inventory.open = true
	a1.inventory.bag = key[0]
	ix, iy := a1.inventoryUI.Origin(1024, 768)
	sx, sy, _, _ := ui.InventoryCell(key[0], key[1])
	ax, ay := a1.duelUI.Origin(1024, 768)
	dx, dy, _ := ui.ExchangeCell(0, 0)
	poll := func(app *App, cond func() bool) {
		t.Helper()
		deadline := time.Now().Add(8 * time.Second)
		for !cond() {
			app.pumpWorld()
			if app.state == nil {
				t.Fatal("disconnected")
			}
			if time.Now().After(deadline) {
				t.Fatal("duel UI response timed out")
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
	a1.worldControls(&platform.Input{MouseX: ix + sx + 1, MouseY: iy + sy + 1, Pressed: true, MouseDown: true}, 1024, 768)
	a1.worldControls(&platform.Input{MouseX: ax + dx + 1, MouseY: ay + dy + 1, Clicked: true}, 1024, 768)
	if a1.duel.quantity == nil {
		t.Fatal("wager quantity dialog missing")
	}
	qx, qy := a1.quantityUI.X, a1.quantityUI.Y
	a1.worldControls(&platform.Input{MouseX: qx + 30, MouseY: qy + 40, Pressed: true, MouseDown: true}, 1024, 768)
	a1.worldControls(&platform.Input{Chars: []rune{'1'}, Keys: []platform.Key{platform.KeyEnter}}, 1024, 768)
	a1.worldControls(&platform.Input{MouseX: qx + 30, MouseY: qy + 70, Pressed: true, MouseDown: true}, 1024, 768)
	a1.worldControls(&platform.Input{MouseX: qx + 30, MouseY: qy + 70, Clicked: true}, 1024, 768)
	if a1.duel.mutation != 1 || a.Duel.Own[0].Item.Count != 0 {
		t.Fatal("wager applied before acknowledgement")
	}
	poll(a1, func() bool { return a1.duel.mutation == 0 && a.Duel.Own[0].Item.Count == 1 })
	waitB(func() bool { return b.Duel.Other[0].Count == 1 })
	if a.Inventory[key] != wager {
		t.Fatal("wager changed inventory before duel")
	}
	bx, by := a2.duelUI.Origin(1024, 768)
	click := func(app *App, x, y int) {
		app.worldControls(&platform.Input{MouseX: x, MouseY: y, Pressed: true, MouseDown: true}, 1024, 768)
		app.worldControls(&platform.Input{MouseX: x, MouseY: y, Clicked: true}, 1024, 768)
	}
	click(a2, bx+95, by+180)
	if !a2.duel.moneyOpen {
		t.Fatal("gold dialog missing")
	}
	a2.worldControls(&platform.Input{Chars: []rune{'1'}, Keys: []platform.Key{platform.KeyEnter}}, 1024, 768)
	poll(a2, func() bool { return a2.duel.mutation == 0 && b.Duel.OwnMoney == 1 })
	poll(a1, func() bool { return a.Duel.OtherMoney == 1 })
	a1.worldControls(&platform.Input{MouseX: ax + dx + 1, MouseY: ay + dy + 1, RightPressed: true, RightDown: true}, 1024, 768)
	poll(a1, func() bool { return a1.duel.mutation == 0 && a.Duel.Own[0].Item.Count == 0 })
	poll(a2, func() bool { return b.Duel.Other[0].Count == 0 })
	click(a2, bx+95, by+180)
	a2.worldControls(&platform.Input{Chars: []rune{'0'}, Keys: []platform.Key{platform.KeyEnter}}, 1024, 768)
	poll(a2, func() bool { return a2.duel.mutation == 0 && b.Duel.OwnMoney == 0 })
	poll(a1, func() bool { return a.Duel.OtherMoney == 0 })
	if a.Inventory[key] != wager || a.Details.Money != moneyA || b.Details.Money != moneyB {
		t.Fatal("removed wagers changed inventory or balances")
	}
	click(a1, ax+70, ay+215)
	poll(a1, func() bool { return a1.duel.mutation == 0 && a.Duel.OwnReady })
	poll(a2, func() bool { return b.Duel.OtherReady })
	click(a1, ax+70, ay+215)
	poll(a1, func() bool { return a1.duel.mutation == 0 && !a.Duel.OwnReady })
	poll(a2, func() bool { return !b.Duel.OtherReady })
	click(a1, ax+70, ay+215)
	poll(a1, func() bool { return a1.duel.mutation == 0 && a.Duel.OwnReady })
	poll(a2, func() bool { return b.Duel.OtherReady })
	click(a2, bx+70, by+215)
	poll(a1, func() bool { return a.Duel.Phase == game.DuelCounting })
	poll(a2, func() bool { return b.Duel.Phase == game.DuelCounting })
	poll(a1, func() bool { return a.Duel.Phase == game.DuelFighting })
	poll(a2, func() bool { return b.Duel.Phase == game.DuelFighting })
	a2.duelCommand("/forfeit")
	poll(a1, func() bool { return a.Duel.Phase == game.DuelNone && a.Duel.Result.Result == 2 })
	poll(a2, func() bool { return b.Duel.Phase == game.DuelNone && b.Duel.Result.Result == 2 })
	if a.Duel.Result.Player != b.SelfID || b.Duel.Result.Player != b.SelfID || a.Details.Money != moneyA || b.Details.Money != moneyB {
		t.Fatal("duel surrender changed balance or wrong player")
	}
}

// The fixtures are disposable and placed together by n2_smoke.py. This
// exercises a real auto-attack and the server's wager removal and drops.
func TestDuelVictoryWagersAgainstRapture(t *testing.T) {
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
	var key [2]uint8
	var item proto.InventoryItem
	for k, it := range a.Inventory {
		if k[0] == 1 && it.Type == 25 && it.TypeID == 4 && it.Count > 1 {
			key, item = k, it
			break
		}
	}
	if item.Count < 2 || b.Details.Money < 1 {
		t.Fatal("wager fixture unavailable")
	}
	initialMoney, winnerMoney := b.Details.Money, a.Details.Money
	sendA(proto.DuelRequestPacket(b.SelfID))
	waitB(func() bool { return b.Duel.Phase == game.DuelIncoming })
	sendB(proto.DuelAnswerPacket(true))
	waitA(func() bool { return a.Duel.Phase == game.DuelWager })
	waitB(func() bool { return b.Duel.Phase == game.DuelWager })
	sendA(proto.DuelAddPacket(key[0], key[1], 1, 0))
	waitA(func() bool { return a.Duel.Own[0].Item.Count == 1 })
	waitB(func() bool { return b.Duel.Other[0].Count == 1 })
	sendB(proto.DuelMoneyPacket(1))
	waitA(func() bool { return a.Duel.OtherMoney == 1 })
	waitB(func() bool { return b.Duel.OwnMoney == 1 })
	sendA(proto.DuelReadyPacket(0))
	waitA(func() bool { return a.Duel.OwnReady })
	sendB(proto.DuelReadyPacket(0))
	waitA(func() bool { return a.Duel.Phase == game.DuelCounting })
	waitB(func() bool { return b.Duel.Phase == game.DuelCounting })
	waitA(func() bool { return a.Inventory[key].Count == item.Count-1 })
	waitB(func() bool { return b.Details.Money == initialMoney-1 })
	if a.Inventory[key].Count != item.Count-1 || b.Details.Money != initialMoney-1 {
		t.Fatalf("wagers not withdrawn: item %d/%d, gold %d/%d", a.Inventory[key].Count, item.Count, b.Details.Money, initialMoney)
	}
	waitA(func() bool { return a.Duel.Phase == game.DuelFighting })
	waitB(func() bool { return b.Duel.Phase == game.DuelFighting })
	before := make(map[uint32]bool, len(a.GroundItems))
	for id := range a.GroundItems {
		before[id] = true
	}
	a.Target = b.SelfID
	appA.updateCombat(&platform.Input{Chars: []rune{'f'}}, 1024, 768)
	deadline := time.Now().Add(30 * time.Second)
	for a.Duel.Phase != game.DuelNone || b.Duel.Phase != game.DuelNone || !b.Self.Dead {
		appA.pumpWorld()
		appB.pumpWorld()
		if appA.state == nil || appB.state == nil {
			t.Fatal("disconnected during duel")
		}
		if time.Now().After(deadline) {
			t.Fatalf("duel timed out: phases %d/%d, loser dead=%v", a.Duel.Phase, b.Duel.Phase, b.Self.Dead)
		}
		time.Sleep(10 * time.Millisecond)
	}
	if a.Duel.Outcome != 1 || b.Duel.Outcome != 2 {
		t.Fatalf("wrong duel outcome %d/%d", a.Duel.Outcome, b.Duel.Outcome)
	}
	var itemDrop, moneyDrop uint32
	waitA(func() bool {
		for id, drop := range a.GroundItems {
			if before[id] {
				continue
			}
			if drop.Type == 25 && drop.TypeID == 4 {
				itemDrop = id
			}
			if drop.Type == 26 && drop.TypeID == 0 {
				moneyDrop = id
			}
		}
		return itemDrop != 0 && moneyDrop != 0
	})
	for _, id := range []uint32{itemDrop, moneyDrop} {
		drop := a.GroundItems[id]
		sendA(proto.MovePacket(0, 0, drop.X, drop.Y, drop.Z))
		sendA(proto.PickupPacket(id))
		waitA(func() bool { _, exists := a.GroundItems[id]; return !exists })
	}
	waitA(func() bool { return a.Inventory[key].Count == item.Count && a.Details.Money == winnerMoney+1 })
	sendB(proto.RevivePacket())
	waitB(func() bool { return !b.Self.Dead })
	t.Log("real PvP victory and both wager drops recovered by the winner")
}
