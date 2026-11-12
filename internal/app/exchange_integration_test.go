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

func TestExchangeAgainstRapture(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_PARTNER") == "" {
		t.Skip("requires two disposable accounts")
	}
	a, wa, _, waitA := testworld.Enter(t)
	b, wb, _, waitB := testworld.EnterAs(t, os.Getenv("SHAIYA_TEST_PARTNER"), os.Getenv("SHAIYA_TEST_PARTNER_PASSWORD"))
	a1, a2 := inputApp(), inputApp()
	a1.state, a1.world = a, wa
	a2.state, a2.world = b, wb
	a1.exchangeUI, _ = ui.NewExchangePanel(inputTextures{})
	a1.exchangeMoney, _ = ui.NewExchangeMoney(inputTextures{})
	a2.exchangeUI, _ = ui.NewExchangePanel(inputTextures{})
	a1.exchangeInvite, _ = ui.NewMessage(inputTextures{})
	a2.exchangeInvite, _ = ui.NewMessage(inputTextures{})
	var key [2]uint8
	var item proto.InventoryItem
	found := false
	for k, it := range a.Inventory {
		if k[0] == 1 && it.Type == 25 && it.TypeID == 4 && it.Count == 5 {
			key, item, found = k, it, true
			break
		}
	}
	if !found {
		t.Fatal("disposable character has no fixture stack")
	}
	count := func(s *game.State) int {
		n := 0
		for k, it := range s.Inventory {
			if k[0] > 0 && it.Type == item.Type && it.TypeID == item.TypeID {
				n += int(it.Count)
			}
		}
		return n
	}
	countA, countB, moneyA, moneyB := count(a), count(b), a.Details.Money, b.Details.Money
	waitA(func() bool { e := a.Entities[b.SelfID]; return e != nil && e.Kind == game.KindUser && e.Shape != nil })
	waitB(func() bool { e := b.Entities[a.SelfID]; return e != nil && e.Kind == game.KindUser })
	a.Target = b.SelfID
	a1.worldControls(&platform.Input{MouseX: 300, MouseY: 20, RightPressed: true, RightDown: true}, 1024, 768)
	if !a1.playerContext.open {
		t.Fatal("player context menu did not open")
	}
	cx, cy := a1.playerContext.x, a1.playerContext.y
	a1.worldControls(&platform.Input{MouseX: cx + 10, MouseY: cy + contextRowHeight + 5, Pressed: true, MouseDown: true}, 1024, 768)
	a1.worldControls(&platform.Input{MouseX: cx + 10, MouseY: cy + contextRowHeight + 5, Clicked: true}, 1024, 768)
	waitB(func() bool { return b.Exchange.InviteID == a.SelfID })
	if !a2.updateExchangeInvite(&platform.Input{}, 1024, 768) || !a2.exchangeInvite.Visible {
		t.Fatal("trade invitation not shown")
	}
	a2.updateExchangeInvite(&platform.Input{Keys: []platform.Key{platform.KeyEnter}}, 1024, 768)
	waitA(func() bool { return a.Exchange.Active })
	waitB(func() bool { return b.Exchange.Active })
	if a.Exchange.PartnerID != b.SelfID || b.Exchange.PartnerID != a.SelfID {
		t.Fatal("wrong partner")
	}
	a1.inventory.open = true
	a1.inventory.bag = key[0]
	ix, iy := a1.inventoryUI.Origin(1024, 768)
	sx, sy, _, ok := ui.InventoryCell(key[0], key[1])
	if !ok {
		t.Fatal("fixture outside inventory")
	}
	ex, ey := a1.exchangeUI.Origin(1024, 768)
	dx, dy, _ := ui.ExchangeCell(0, 0)
	pollA := func(cond func() bool) {
		t.Helper()
		deadline := time.Now().Add(8 * time.Second)
		for !cond() {
			a1.pumpWorld()
			if a1.state == nil {
				t.Fatal("disconnected")
			}
			if time.Now().After(deadline) {
				t.Fatal("trade UI response timed out")
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
	offer := func() {
		t.Helper()
		a1.worldControls(&platform.Input{MouseX: ix + sx + 1, MouseY: iy + sy + 1, Pressed: true, MouseDown: true}, 1024, 768)
		a1.worldControls(&platform.Input{MouseX: ex + dx + 1, MouseY: ey + dy + 1, Clicked: true}, 1024, 768)
		if a1.exchange.quantity == nil {
			t.Fatal("stack quantity dialog missing")
		}
		qx, qy := a1.quantityUI.X, a1.quantityUI.Y
		a1.worldControls(&platform.Input{MouseX: qx + 30, MouseY: qy + 40, Pressed: true, MouseDown: true}, 1024, 768)
		a1.worldControls(&platform.Input{Chars: []rune{'1'}, Keys: []platform.Key{platform.KeyEnter}}, 1024, 768)
		a1.worldControls(&platform.Input{MouseX: qx + 30, MouseY: qy + 70, Pressed: true, MouseDown: true}, 1024, 768)
		a1.worldControls(&platform.Input{MouseX: qx + 30, MouseY: qy + 70, Clicked: true}, 1024, 768)
		if a1.exchange.mutation != 1 || a.Exchange.Own[0].Item.Count != 0 {
			t.Fatal("offer changed before acknowledgement")
		}
		pollA(func() bool { return a1.exchange.mutation == 0 && a.Exchange.Own[0].Item.Count == 1 })
	}
	offer()
	waitB(func() bool { return b.Exchange.Other[0].Count == 1 })
	if count(a) != countA || count(b) != countB {
		t.Fatal("offer changed inventory before confirmation")
	}
	a1.worldControls(&platform.Input{MouseX: ex + 95, MouseY: ey + 180, Pressed: true, MouseDown: true}, 1024, 768)
	a1.worldControls(&platform.Input{MouseX: ex + 95, MouseY: ey + 180, Clicked: true}, 1024, 768)
	if !a1.exchange.moneyOpen {
		t.Fatal("money dialog missing")
	}
	a1.worldControls(&platform.Input{Chars: []rune{'5', '0'}, Keys: []platform.Key{platform.KeyEnter}}, 1024, 768)
	if a1.exchange.mutation != 3 || a.Exchange.OwnMoney != 0 {
		t.Fatal("money changed before acknowledgement")
	}
	pollA(func() bool { return a1.exchange.mutation == 0 && a.Exchange.OwnMoney == 50 })
	waitB(func() bool { return b.Exchange.OtherMoney == 50 })
	a1.worldControls(&platform.Input{MouseX: ex + dx + 1, MouseY: ey + dy + 1, RightPressed: true, RightDown: true}, 1024, 768)
	if a1.exchange.mutation != 2 || a.Exchange.Own[0].Item.Count != 1 {
		t.Fatal("removal changed before acknowledgement")
	}
	pollA(func() bool { return a1.exchange.mutation == 0 && a.Exchange.Own[0].Item.Count == 0 })
	waitB(func() bool { return b.Exchange.Other[0].Count == 0 })
	offer()
	waitB(func() bool { return b.Exchange.Other[0].Count == 1 })
	confirm := func(app *App) {
		x, y := app.exchangeUI.Origin(1024, 768)
		app.updateExchange(&platform.Input{MouseX: x + 70, MouseY: y + 215, Pressed: true, MouseDown: true}, 1024, 768)
		app.updateExchange(&platform.Input{MouseX: x + 70, MouseY: y + 215, Clicked: true}, 1024, 768)
	}
	confirm(a1)
	waitA(func() bool { return a.Exchange.OwnReady && a.Exchange.OwnConfirmed })
	waitB(func() bool { return b.Exchange.OtherReady && b.Exchange.OtherConfirmed })
	confirm(a2)
	waitA(func() bool { return !a.Exchange.Active && a.Exchange.EndSerial > 0 })
	waitB(func() bool { return !b.Exchange.Active && b.Exchange.EndSerial > 0 })
	if a.Exchange.EndResult != 0 || b.Exchange.EndResult != 0 {
		t.Fatal("trade rejected")
	}
	waitA(func() bool { return count(a) == countA-1 && a.Details.Money == moneyA-50 })
	waitB(func() bool { return count(b) == countB+1 && b.Details.Money == moneyB+50 })
}

func TestExchangeDeclineEditAndCancelAgainstRapture(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_PARTNER") == "" {
		t.Skip("requires two disposable accounts")
	}
	a, wa, sendA, waitA := testworld.Enter(t)
	b, wb, sendB, waitB := testworld.EnterAs(t, os.Getenv("SHAIYA_TEST_PARTNER"), os.Getenv("SHAIYA_TEST_PARTNER_PASSWORD"))
	a1, a2 := inputApp(), inputApp()
	a1.state, a1.world = a, wa
	a2.state, a2.world = b, wb
	a1.exchangeUI, _ = ui.NewExchangePanel(inputTextures{})
	a2.exchangeInvite, _ = ui.NewMessage(inputTextures{})
	waitA(func() bool { e := a.Entities[b.SelfID]; return e != nil && e.Kind == game.KindUser })
	waitB(func() bool { e := b.Entities[a.SelfID]; return e != nil && e.Kind == game.KindUser })
	var keyA, keyB [2]uint8
	var itemA, itemB proto.InventoryItem
	foundA, foundB := false, false
	for k, it := range a.Inventory {
		if k[0] > 0 && it.Type == 25 && it.TypeID == 4 && it.Count > 0 {
			keyA, itemA, foundA = k, it, true
			break
		}
	}
	for k, it := range b.Inventory {
		if k[0] > 0 && it.Type == 25 && it.TypeID == 4 && it.Count > 0 {
			keyB, itemB, foundB = k, it, true
			break
		}
	}
	if !foundA || !foundB {
		t.Fatal("traded fixture items not present")
	}
	moneyA, moneyB := a.Details.Money, b.Details.Money
	sendA(proto.ExchangeRequestPacket(b.SelfID))
	waitB(func() bool { return b.Exchange.InviteSerial == 1 })
	a2.updateExchangeInvite(&platform.Input{}, 1024, 768)
	a2.updateExchangeInvite(&platform.Input{Keys: []platform.Key{platform.KeyEscape}}, 1024, 768)
	waitA(func() bool { return a.Exchange.AnswerSerial > 0 && a.Exchange.Answer == 1 })
	if a.Exchange.Active || b.Exchange.Active {
		t.Fatal("declined trade opened")
	}
	sendA(proto.ExchangeRequestPacket(b.SelfID))
	waitB(func() bool { return b.Exchange.InviteSerial == 2 })
	sendB(proto.ExchangeAnswerPacket(true))
	waitA(func() bool { return a.Exchange.Active })
	waitB(func() bool { return b.Exchange.Active })
	sendA(proto.ExchangeAddPacket(keyA[0], keyA[1], 1, 0))
	waitA(func() bool { return a.Exchange.Own[0].Item.Count == 1 })
	waitB(func() bool { return b.Exchange.Other[0].Count == 1 })
	sendB(proto.ExchangeAddPacket(keyB[0], keyB[1], 1, 0))
	waitB(func() bool { return b.Exchange.Own[0].Item.Count == 1 })
	waitA(func() bool { return a.Exchange.Other[0].Count == 1 })
	x, y := a1.exchangeUI.Origin(1024, 768)
	cx, cy, _ := ui.ExchangeCell(0, 0)
	a1.worldControls(&platform.Input{MouseX: x + cx + 1, MouseY: y + cy + 1, RightPressed: true, RightDown: true}, 1024, 768)
	if a1.exchange.mutation != 0 || a.Exchange.Own[0].Item.Count != 1 || !a.Exchange.Active {
		t.Fatal("unsafe removal was not blocked")
	}
	sendA(proto.ExchangeReadyPacket(0))
	sendA(proto.ExchangeConfirmPacket(true))
	waitA(func() bool { return a.Exchange.OwnReady && a.Exchange.OwnConfirmed })
	waitB(func() bool { return b.Exchange.OtherReady && b.Exchange.OtherConfirmed })
	sendB(proto.ExchangeMoneyPacket(1))
	waitA(func() bool { return !a.Exchange.OwnReady && !a.Exchange.OwnConfirmed && a.Exchange.OtherMoney == 1 })
	waitB(func() bool { return !b.Exchange.OtherReady && !b.Exchange.OtherConfirmed && b.Exchange.OwnMoney == 1 })
	a1.updateExchange(&platform.Input{Keys: []platform.Key{platform.KeyEscape}}, 1024, 768)
	waitA(func() bool { return !a.Exchange.Active && a.Exchange.EndSerial > 0 })
	waitB(func() bool { return !b.Exchange.Active && b.Exchange.EndSerial > 0 })
	if a.Exchange.EndResult != 1 || b.Exchange.EndResult != 1 || a.Inventory[keyA] != itemA || b.Inventory[keyB] != itemB || a.Details.Money != moneyA || b.Details.Money != moneyB {
		t.Fatal("cancel changed inventory or balance")
	}
}
