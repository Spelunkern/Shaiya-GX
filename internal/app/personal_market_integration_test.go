//go:build integration

package app

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/render/gl"
	"github.com/Spelunkern/Shaiya-GX/internal/testworld"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

func TestPersonalMarketBetweenClients(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_PERSONAL_MARKET") != "1" || os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires disposable seller, buyer and original assets")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	const w, h = 1280, 768
	win, err := platform.Open(platform.Config{Title: "personal market", Width: w, Height: h, Hidden: true})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
	if err := render.Init(false); err != nil {
		t.Fatal(err)
	}
	arc, err := archive.OpenDirectory(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	sst, sw, _, waitSeller := testworld.Enter(t)
	bst, bw, _, waitBuyer := testworld.EnterAs(t, os.Getenv("SHAIYA_TEST_PARTNER"), os.Getenv("SHAIYA_TEST_PARTNER_PASSWORD"))
	waitSeller(func() bool { return sst.Entities[bst.SelfID] != nil && sst.InventoryReady })
	waitBuyer(func() bool { return bst.Entities[sst.SelfID] != nil && bst.InventoryReady })
	seller, err := New(Config{}, arc)
	if err != nil {
		t.Fatal(err)
	}
	defer seller.Close()
	buyer, err := New(Config{}, arc)
	if err != nil {
		t.Fatal(err)
	}
	defer buyer.Close()
	seller.state, seller.world, seller.screen = sst, sw, screenWorld
	buyer.state, buyer.world, buyer.screen = bst, bw, screenWorld
	seller.enterWorld()
	buyer.enterWorld()
	pump := func(description string, done func() bool) {
		t.Helper()
		deadline := time.Now().Add(8 * time.Second)
		for !done() {
			seller.pumpWorld()
			buyer.pumpWorld()
			if seller.state == nil || buyer.state == nil || time.Now().After(deadline) {
				t.Fatalf("%s: seller phase %d, buyer seller %d, open %v, listing %v; pending %04x/%04x", description, sst.MyShop.Phase, bst.ShopView.Seller, bst.ShopView.Open, bst.ShopView.Present[0], seller.personalMarket.pending, buyer.personalMarket.pending)
			}
			time.Sleep(time.Millisecond)
		}
	}
	click := func(a *App, x, y int) {
		t.Helper()
		a.worldControls(&platform.Input{MouseX: x, MouseY: y, MouseDown: true, Pressed: true}, w, h)
		a.worldControls(&platform.Input{MouseX: x, MouseY: y, Clicked: true}, w, h)
	}
	draw := func(a *App, name string) {
		t.Helper()
		out := os.Getenv("SHAIYA_TEST_ARTIFACTS")
		if out == "" {
			return
		}
		gl.Viewport(0, 0, w, h)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		a.sprites.Begin(w, h)
		a.drawPersonalMarket(w, h)
		a.drawPersonalMarketModal(w)
		a.sprites.End()
		if err := render.SavePNG(filepath.Join(out, name+".png"), w, h); err != nil {
			t.Fatal(err)
		}
	}
	if sst.MapID != 35 || bst.MapID != 35 || sst.Inventory[[2]uint8{1, 0}].Count != 4 || sst.Inventory[[2]uint8{1, 1}].Count != 1 {
		t.Fatal("market fixture requires two characters in Apulune and two seller listings")
	}
	seller.worldControls(&platform.Input{Chars: []rune{'b'}}, w, h)
	sx, sy := seller.subActionsUI.Origin(w, h)
	cx, cy := ui.SubActionCell(15)
	click(seller, sx+cx+3, sy+cy+3)
	pump("open Setup Market", func() bool { return sst.MyShop.Phase == proto.MyShopSetup })
	seller.subActionsOpen = false
	mx, my := seller.personalMarketUI.Origin(w, h)
	place := func(bagSlot, index int, price string) {
		t.Helper()
		ix, iy := seller.inventoryUI.Origin(w, h)
		bx, by, _, _ := ui.InventoryCell(1, uint8(bagSlot))
		px, py := ui.PersonalMarketCell(index)
		seller.worldControls(&platform.Input{MouseX: ix + bx + 3, MouseY: iy + by + 3, MouseDown: true, Pressed: true}, w, h)
		seller.worldControls(&platform.Input{MouseX: mx + px + 4, MouseY: my + py + 4, MouseDown: true}, w, h)
		seller.worldControls(&platform.Input{MouseX: mx + px + 4, MouseY: my + py + 4, Clicked: true}, w, h)
		if seller.personalMarket.dialog.op != proto.MyShopAdd {
			t.Fatalf("drag from bag slot %d did not open price", bagSlot)
		}
		seller.worldControls(&platform.Input{Chars: []rune(price), Keys: []platform.Key{platform.KeyEnter}}, w, h)
		pump("acknowledge offered item", func() bool { return sst.MyShop.Slots[index].Used })
	}
	place(0, 0, "125")
	place(1, 1, "999")
	secondX, secondY := ui.PersonalMarketCell(1)
	seller.worldControls(&platform.Input{
		MouseX: mx + secondX + 5, MouseY: my + secondY + 5, RightPressed: true, RightDown: true,
	}, w, h)
	pump("remove second offer", func() bool { return !sst.MyShop.Slots[1].Used })
	firstX, firstY := ui.PersonalMarketCell(0)
	click(seller, mx+firstX+5, my+firstY+5)
	seller.worldControls(&platform.Input{Chars: []rune("160"), Keys: []platform.Key{platform.KeyEnter}}, w, h)
	pump("change unit price", func() bool { return sst.MyShop.Slots[0].Price == 160 })
	const message = "Apulune Potions"
	click(seller, mx+30, my+73)
	if !seller.personalMarket.editing {
		t.Fatal("click did not focus the market message")
	}
	seller.worldControls(&platform.Input{Chars: []rune(message)}, w, h)
	seller.worldControls(&platform.Input{Keys: []platform.Key{platform.KeyEnter}}, w, h)
	if seller.personalMarket.editing || seller.personalMarket.title.Text != message {
		t.Fatal("typed market message was not committed")
	}
	draw(seller, "market-setup")
	click(seller, mx+35, my+368)
	pump("seller seated and advert visible", func() bool {
		return sst.MyShop.Phase == proto.MyShopSelling && sst.Self.Sitting && bst.ShopSigns[sst.SelfID] == message
	})
	t.Log("seller typed the market message, opened the stall, and the buyer received the advertised title")
	openBuyer := func() {
		t.Helper()
		buyer.state.Target = sst.SelfID
		tx, ty := buyer.hud.TargetOrigin(w, h)
		buyer.worldControls(&platform.Input{MouseX: tx + 30, MouseY: ty + 18, RightPressed: true, RightDown: true}, w, h)
		c := buyer.playerContext
		actions := buyer.contextActions(sst.SelfID)
		if !c.open || len(actions) == 0 || actions[0].label != "View market" {
			t.Fatal("seller menu failed to offer View market")
		}
		click(buyer, c.x+17, c.y+ui.PlayerMenuTextInset+2)
		pump("browse market", func() bool { return bst.ShopView.Open && bst.ShopView.Present[0] })
	}
	openBuyer()
	buyer.worldControls(&platform.Input{Keys: []platform.Key{platform.KeyEscape}}, w, h)
	pump("buyer leaves independently", func() bool { return !bst.ShopView.Open && sst.MyShop.Phase == proto.MyShopSelling })
	openBuyer()
	beforeS, beforeB := sst.Details.Money, bst.Details.Money
	purchase := func() {
		t.Helper()
		x, y := buyer.marketUI.Origin(w, h)
		cx, cy, _ := ui.MarketCell(0)
		click(buyer, x+cx+5, y+cy+5)
		q := buyer.quantityUI
		if buyer.personalMarket.dialog.op != proto.MyShopBuy || q.Value() != 1 {
			t.Fatal("buy did not start a one-unit quantity dialog")
		}
		buyer.worldControls(&platform.Input{MouseX: q.X + 36, MouseY: q.Y + 41, Pressed: true, Chars: []rune{'2'}, Keys: []platform.Key{platform.KeyEnter}}, w, h)
		if q.Value() != 2 {
			t.Fatal("buyer could not choose two units")
		}
		draw(buyer, "market-purchase")
		click(buyer, q.X+37, q.Y+77)
	}
	purchase()
	pump("partial purchase updates both bags and gold", func() bool {
		return bst.ShopView.BuyResult == proto.MyShopBuyOK && bst.ShopView.Items[0].Item.Count == 2 &&
			sst.Inventory[[2]uint8{1, 0}].Count == 2 && bst.Inventory[[2]uint8{1, 0}].Count == 2 &&
			sst.Details.Money == beforeS+320 && bst.Details.Money == beforeB-320
	})
	t.Logf("buyer received two items for 320 gold and seller received 320 gold (seller %d → %d, buyer %d → %d)",
		beforeS, sst.Details.Money, beforeB, bst.Details.Money)
	click(seller, mx+35, my+368)
	pump("stop closes buyer and removes advert", func() bool {
		return sst.MyShop.Phase == proto.MyShopSetup && !bst.ShopView.Open && bst.ShopSigns[sst.SelfID] == ""
	})
	click(seller, mx+35, my+368)
	pump("reopen retains unsold units", func() bool { return sst.MyShop.Phase == proto.MyShopSelling && bst.ShopSigns[sst.SelfID] != "" })
	openBuyer()
	purchase()
	pump("last sale closes the market", func() bool {
		return sst.MyShop.Phase == proto.MyShopInactive && !bst.ShopView.Open && bst.ShopSigns[sst.SelfID] == "" &&
			sst.Inventory[[2]uint8{1, 0}].Count == 0 && bst.Inventory[[2]uint8{1, 0}].Count == 4 &&
			sst.Details.Money == beforeS+640 && bst.Details.Money == beforeB-640
	})
	t.Log("the buyer received the last two items and the empty stall closed for both clients")
}
