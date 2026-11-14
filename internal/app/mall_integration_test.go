//go:build integration

package app

import (
	"os"
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/testworld"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

func TestMallAgainstRapture(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_MALL") != "1" || os.Getenv("SHAIYA_TEST_PARTNER") == "" {
		t.Skip("requires two disposable mall accounts")
	}
	arc, err := archive.Open(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	s1, w1, send1, _ := testworld.Enter(t)
	s2, w2, send2, _ := testworld.EnterAs(t, os.Getenv("SHAIYA_TEST_PARTNER"), os.Getenv("SHAIYA_TEST_PARTNER_PASSWORD"))
	a1, a2 := inputApp(), inputApp()
	a1.state, a1.world, a1.arc = s1, w1, arc
	a2.state, a2.world, a2.arc = s2, w2, arc
	a1.mallUI, _ = ui.NewMall(inputTextures{}, nil)
	a2.mallUI, _ = ui.NewMall(inputTextures{}, nil)
	a1.mallConfirm, _ = ui.NewMessage(inputTextures{})
	a2.mallConfirm, _ = ui.NewMessage(inputTextures{})
	if err := a1.prepareMall(); err != nil {
		t.Fatal(err)
	}
	if err := a2.prepareMall(); err != nil {
		t.Fatal(err)
	}
	// Older catalogs may keep the probe SKU hidden.
	price := uint32(0)
	skuVisible := false
	for _, product := range a1.mallCatalog {
		if product.Code == "AP_1" {
			price = product.Cost
			if product.Bag != 0 {
				skuVisible = true
			}
			break
		}
	}
	if !skuVisible {
		a1.mall.products[0] = append(a1.mall.products[0], ui.MallProduct{Code: "AP_1", Name: "Test AP", Cost: price})
	}
	click := func(a *App, dx, dy int) {
		t.Helper()
		x, y := a.mallUI.Origin(1024, 768)
		a.updateMall(&platform.Input{Pressed: true, MouseDown: true, MouseX: x + dx, MouseY: y + dy}, 1024, 768)
		a.updateMall(&platform.Input{Clicked: true, MouseX: x + dx, MouseY: y + dy}, 1024, 768)
	}
	selectProduct := func(code string) {
		t.Helper()
		category, index := -1, -1
		for c, products := range a1.mall.products {
			for i, p := range products {
				if p.Code == code {
					category, index = c, i
					break
				}
			}
		}
		if category < 0 {
			t.Fatal("product missing", code)
		}
		if a1.mallUI.Category != category {
			click(a1, 20+category*128, 135)
		}
		a1.mallUI.Scroll = max(0, index/2-2)
		x, y := ui.MallProductRect(index - a1.mallUI.Scroll*2)
		click(a1, x+5, y+5)
		if a1.mallUI.Selected != index {
			t.Fatalf("product %s not selected", code)
		}
	}
	wait := func(a *App, condition func() bool) {
		t.Helper()
		end := time.Now().Add(15 * time.Second)
		for !condition() {
			a.pumpWorld()
			if a.state == nil {
				t.Fatal("mall disconnected")
			}
			if time.Now().After(end) {
				t.Fatal("mall response timed out")
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
	wait(a1, func() bool { return s1.Mall.StoredReady })
	wait(a2, func() bool { return s2.Mall.StoredReady })
	a1.mallCommand("/mall points")
	wait(a1, func() bool { return s1.Mall.PointsReady })
	initialPoints := s1.Mall.Points
	if uint64(initialPoints) < uint64(price)*2 {
		t.Fatal("fixture needs enough points for buy and gift")
	}
	before := len(s1.Inventory)
	selectProduct("AP_1")
	click(a1, 295, 410)
	if !a1.mallConfirm.Visible || a1.mall.modal != ui.MallBuy {
		t.Fatal("buy dialog missing")
	}
	a1.updateMallConfirm(&platform.Input{Keys: []platform.Key{platform.KeyEnter}}, 1024, 768)
	if len(s1.Inventory) != before || s1.Mall.Points != initialPoints {
		t.Fatal("purchase mutated before confirmation")
	}
	wait(a1, func() bool { return s1.Mall.PurchaseSerial == 1 })
	if s1.Mall.LastPurchase.Result != 0 || s1.Mall.Points != initialPoints-price || len(s1.Mall.LastPurchase.Items) != 1 {
		t.Fatal("purchase result", s1.Mall.LastPurchase)
	}
	bought := s1.Mall.LastPurchase.Items[0]
	if it := s1.Inventory[[2]uint8{bought.Bag, bought.Slot}]; it.Type != 100 || it.TypeID != 251 || it.Count != 1 {
		t.Fatal("confirmed purchase inventory", it)
	}
	a1.mallCommand("/mall buy INVALID_CODE")
	wait(a1, func() bool { return s1.Mall.PurchaseSerial == 2 })
	if s1.Mall.LastPurchase.Result != 1 || s1.Mall.Points != initialPoints-price {
		t.Fatal("invalid product", s1.Mall.LastPurchase)
	}
	click(a1, 395, 410)
	if !a1.mallConfirm.Visible || a1.mall.modal != ui.MallGift {
		t.Fatal("gift dialog missing")
	}
	a1.updateMallConfirm(&platform.Input{Chars: []rune(s2.Slots[0].Name)}, 1024, 768)
	a1.updateMallConfirm(&platform.Input{Keys: []platform.Key{platform.KeyEnter}}, 1024, 768)
	wait(a1, func() bool { return s1.Mall.GiftSerial == 1 })
	if s1.Mall.LastGift.Result != 0 || s1.Mall.Points != initialPoints-2*price {
		t.Fatal("gift result", s1.Mall.LastGift)
	}
	wait(a2, func() bool { return len(s2.Mall.Stored) > 0 })
	var slot uint8
	found := false
	for k, g := range s2.Mall.Stored {
		if g.Type == 100 && g.TypeID == 251 {
			slot, found = k, true
			break
		}
	}
	if !found {
		t.Fatal("gift not delivered", s2.Mall.Stored)
	}
	before = len(s2.Inventory)
	click(a2, 35, 410)
	if !a2.mallUI.GiftOpen {
		t.Fatal("gift box did not open")
	}
	a2.mallUI.GiftPage = int(slot) / 40
	gx, gy := a2.mallUI.GiftOrigin(1024, 768)
	dx, dy := ui.MallGiftCell(int(slot))
	a2.updateMall(&platform.Input{RightPressed: true, RightDown: true, MouseX: gx + dx + 5, MouseY: gy + dy + 5}, 1024, 768)
	if len(s2.Inventory) != before {
		t.Fatal("gift claimed before confirmation")
	}
	wait(a2, func() bool { return s2.Mall.ClaimSerial == 1 })
	if s2.Mall.PendingClaim != nil || len(s2.Mall.Stored) != 0 {
		t.Fatal("confirmed gift not resolved", s2.Mall)
	}
	claimed := false
	var claimedItem proto.InventoryItem
	for _, it := range s2.Inventory {
		if it.Type == 100 && it.TypeID == 251 && it.Count == 1 {
			claimed = true
			claimedItem = it
		}
	}
	if !claimed {
		t.Fatal("claimed gift missing from inventory")
	}
	send2(proto.LogoutPacket())
	wait(a2, func() bool { return s2.SelfID == 0 })
	if err := w2.Shutdown(10 * time.Second); err != nil {
		t.Fatal(err)
	}
	send1(proto.LogoutPacket())
	wait(a1, func() bool { return s1.SelfID == 0 })
	if err := w1.Shutdown(10 * time.Second); err != nil {
		t.Fatal(err)
	}
	st, _, _, _ := testworld.EnterAs(t, os.Getenv("SHAIYA_TEST_PARTNER"), os.Getenv("SHAIYA_TEST_PARTNER_PASSWORD"))
	claimed = false
	for _, it := range st.Inventory {
		if it.Type == 100 && it.TypeID == 251 && it.Count == 1 {
			if it != claimedItem {
				t.Fatalf("claimed inventory differs after reload: %+v -> %+v", claimedItem, it)
			}
			claimed = true
		}
	}
	if !claimed {
		t.Fatal("claimed gift not persisted")
	}
	t.Log("disposable accounts: points, purchase, rejection, gift, claim and persistence confirmed")
}
