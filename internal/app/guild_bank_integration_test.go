//go:build integration

package app

import (
	"os"
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/testworld"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

func TestGuildBankAgainstRapture(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_GUILD") == "" || os.Getenv("SHAIYA_TEST_PARTNER") == "" {
		t.Skip("requires disposable ranked guild")
	}
	a, wa, _, waitA := testworld.Enter(t)
	b, wb, sendB, waitB := testworld.EnterAs(t, os.Getenv("SHAIYA_TEST_PARTNER"), os.Getenv("SHAIYA_TEST_PARTNER_PASSWORD"))
	appA, appB := inputApp(), inputApp()
	appA.state, appA.world = a, wa
	appB.state, appB.world = b, wb
	appA.bankUI, _ = ui.NewWarehouse(inputTextures{})
	appA.items = map[sdata.ItemKey]*sdata.Item{{Type: 25, TypeID: 4}: {Type: 25, TypeID: 4}}
	waitA(func() bool { return a.Guild.ID != 0 && a.Guild.Rank == 1 && a.WarehouseReady })
	waitB(func() bool { return b.Guild.ID == a.Guild.ID && b.Guild.Rank == 9 })
	var npc uint32
	waitA(func() bool {
		for id, e := range a.Entities {
			if e.Kind == game.KindNpc && e.NpcType == 6 {
				npc = id
				return true
			}
		}
		return false
	})
	appA.wv.pos[0], appA.wv.pos[1], appA.wv.pos[2] = a.Self.X, a.Self.Y, a.Self.Z
	appA.worldControls(&platform.Input{Chars: []rune{'n'}}, 1024, 768)
	if appA.shop.npcID != npc {
		t.Fatal("banker not opened")
	}
	appA.worldControls(&platform.Input{}, 1024, 768)
	bx, by := appA.bankUI.Origin(1024, 768)
	click := func(x, y int) {
		appA.worldControls(&platform.Input{MouseX: x, MouseY: y, Pressed: true, MouseDown: true}, 1024, 768)
		appA.worldControls(&platform.Input{MouseX: x, MouseY: y, Clicked: true}, 1024, 768)
	}
	click(bx+240, by+85)
	if !appA.bank.guild {
		t.Fatal("guild bank tab not selected")
	}
	source := [2]uint8{1, 23}
	item := a.Inventory[source]
	if item.Type != 25 || item.TypeID != 4 || item.Count != 5 {
		t.Fatal("bank item fixture missing")
	}
	appA.inventory.bag = 1
	ix, iy := appA.inventoryUI.Origin(1024, 768)
	sx, sy, _, _ := ui.InventoryCell(1, 23)
	dx, dy, _ := ui.WarehouseCell(0, 0)
	drag := func(x, y, xx, yy int) {
		appA.worldControls(&platform.Input{MouseX: x, MouseY: y, Pressed: true, MouseDown: true}, 1024, 768)
		appA.worldControls(&platform.Input{MouseX: xx, MouseY: yy, Clicked: true}, 1024, 768)
	}
	poll := func(app *App, cond func() bool) {
		t.Helper()
		deadline := time.Now().Add(8 * time.Second)
		for !cond() {
			app.pumpWorld()
			if app.state == nil || time.Now().After(deadline) {
				t.Fatal("guild bank response timed out")
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
	drag(ix+sx+1, iy+sy+1, bx+dx+1, by+dy+1)
	if !appA.bank.guildPending || a.Inventory[source] != item {
		t.Fatal("guild deposit changed state before confirmation")
	}
	poll(appA, func() bool { return !appA.inventory.pending })
	poll(appB, func() bool { return b.Guild.Bank[0].Count == item.Count })
	if _, ok := a.Inventory[source]; ok || a.Guild.Bank[0].Count != item.Count {
		t.Fatal("deposit missing")
	}
	dx1, dy1, _ := ui.WarehouseCell(0, 1)
	drag(bx+dx+1, by+dy+1, bx+dx1+1, by+dy1+1)
	poll(appA, func() bool { return !appA.inventory.pending && a.Guild.Bank[1].Count == item.Count })
	poll(appB, func() bool { return b.Guild.Bank[1].Count == item.Count })
	drag(bx+dx1+1, by+dy1+1, ix+sx+1, iy+sy+1)
	poll(appA, func() bool { return !appA.inventory.pending && a.Inventory[source].Count == item.Count })
	poll(appB, func() bool { _, ok := b.Guild.Bank[1]; return !ok })
	if _, ok := a.Guild.Bank[1]; ok {
		t.Fatal("withdrawal left guild bank item")
	}
	sendB(proto.ItemMovePacket(1, 23, proto.GuildBankBag, 2))
	poll(appB, func() bool { return b.Guild.BankResultSerial > 0 })
	if b.Guild.BankResult.Result != 4 || b.Inventory[source].Count != 5 {
		t.Fatal("rank-nine deposit not refused")
	}
	drag(ix+sx+1, iy+sy+1, bx+dx+1, by+dy+1)
	poll(appA, func() bool { return !appA.inventory.pending && a.Guild.Bank[0].Count == 5 })
	if err := wa.Shutdown(10 * time.Second); err != nil {
		t.Fatal(err)
	}
	a2, _, _, waitA2 := testworld.Enter(t)
	waitA2(func() bool { return a2.Guild.ID == b.Guild.ID && a2.Guild.Bank[0].Count == 5 })
	t.Log("guild bank deposit, broadcast, rearrangement, withdrawal, refusal and relog confirmed")
}
