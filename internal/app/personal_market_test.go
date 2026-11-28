package app

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

func TestPersonalMarketSetupControlsKeepTheirPress(t *testing.T) {
	a := inputApp()
	a.personalMarketUI, _ = ui.NewPersonalMarket(inputTextures{})
	a.state.MyShop.Phase = proto.MyShopSetup
	a.personalMarket.title.Max = proto.MyShopTitleBytes
	a.msg = map[int]string{10001: "Market message required"}
	const w, h = 1024, 768
	x, y := a.personalMarketUI.Origin(w, h)
	click := func(dx, dy int) {
		t.Helper()
		a.worldControls(&platform.Input{MouseX: x + dx, MouseY: y + dy, Pressed: true, MouseDown: true}, w, h)
		if a.windowDrag.position != nil {
			t.Fatalf("market control at (%d, %d) started a drag", dx, dy)
		}
		controls := a.worldControls(&platform.Input{MouseX: x + dx, MouseY: y + dy, Clicked: true}, w, h)
		if controls.Clicked {
			t.Fatal("market release leaked into the world")
		}
	}

	click(35, 368)
	if len(a.state.Chat) != 1 || a.state.Chat[0].MessageID != 10001 {
		t.Fatal("Open Market did not reach the title check")
	}
	click(30, 73)
	if !a.personalMarket.editing {
		t.Fatal("market message did not take focus")
	}
	a.worldControls(&platform.Input{Chars: []rune("Apulune Potions")}, w, h)
	a.worldControls(&platform.Input{Keys: []platform.Key{platform.KeyEnter}}, w, h)
	if a.personalMarket.editing || a.personalMarket.title.Text != "Apulune Potions" {
		t.Fatal("market message was not accepted")
	}

	click(148, 369)
	click(174, 15)
	a.worldControls(&platform.Input{MouseX: x + 100, MouseY: y + 33, Pressed: true, MouseDown: true}, w, h)
	if a.windowDrag.position != &a.personalMarketUI.Position {
		t.Fatal("blank market frame did not start a drag")
	}
	a.worldControls(&platform.Input{MouseX: x + 130, MouseY: y + 48, Clicked: true}, w, h)
	x, y = a.personalMarketUI.Origin(w, h)
	click(174, 15)
}

func TestPersonalMarketBuyerUsesItsOwnWindowForDrag(t *testing.T) {
	a := inputApp()
	a.personalMarketUI, _ = ui.NewPersonalMarket(inputTextures{})
	a.state.ShopView.Open, a.state.ShopView.Seller = true, 42
	a.state.ShopView.Present[0] = true
	a.state.ShopView.Items[0] = proto.MyShopItem{Price: 125, Item: proto.InventoryItem{Type: 1, Count: 2}}
	const w, h = 1024, 768
	sx, sy := a.personalMarketUI.Origin(w, h)
	a.marketUI.Position.Move(sx, sy, w, h, ui.MarketWidth, ui.MarketHeight)
	x, y := a.marketUI.Origin(w, h)
	cx, cy, _ := ui.MarketCell(0)
	a.worldControls(&platform.Input{MouseX: x + cx + 4, MouseY: y + cy + 4, Pressed: true, MouseDown: true}, w, h)
	if a.windowDrag.position != nil {
		t.Fatal("buyer item started a seller-window drag")
	}
	a.worldControls(&platform.Input{MouseX: x + cx + 4, MouseY: y + cy + 4, Clicked: true}, w, h)
	if a.personalMarket.dialog.op != proto.MyShopBuy {
		t.Fatal("buyer item did not open the quantity picker")
	}

	a.personalMarket.dialog = marketDialog{}
	a.marketUI.Position.Move(420, 175, w, h, ui.MarketWidth, ui.MarketHeight)
	x, y = a.marketUI.Origin(w, h)
	if !a.updateWindowDrag(&platform.Input{MouseX: x + 70, MouseY: y + 90, Pressed: true, MouseDown: true}, w, h) ||
		a.windowDrag.position != &a.marketUI.Position {
		t.Fatal("buyer frame did not drag its visible market window")
	}
}
