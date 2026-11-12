package app

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

func TestMallPointerDoesNotReachWorld(t *testing.T) {
	a := inputApp()
	a.mallUI, _ = ui.NewMall(inputTextures{}, nil)
	a.mall.open = true
	a.mall.products[0] = []ui.MallProduct{{Code: "test", Name: "Product"}}
	x, y := a.mallUI.Origin(1024, 768)
	pressed := a.worldControls(&platform.Input{Pressed: true, MouseDown: true, MouseX: x + 25, MouseY: y + 175}, 1024, 768)
	released := a.worldControls(&platform.Input{Clicked: true, MouseX: x + 25, MouseY: y + 175}, 1024, 768)
	if pressed.Pressed || released.Clicked || a.mallUI.Selected != 0 {
		t.Fatal("product click leaked or was not selected")
	}
	a.worldControls(&platform.Input{Pressed: true, MouseDown: true, MouseX: x + 25, MouseY: y + 175}, 1024, 768)
	released = a.worldControls(&platform.Input{Clicked: true, MouseX: 10, MouseY: 700}, 1024, 768)
	if released.Clicked {
		t.Fatal("release outside mall leaked to world")
	}
	a.mallUI.GiftOpen = true
	gx, gy := a.mallUI.GiftOrigin(1024, 768)
	right := a.worldControls(&platform.Input{RightPressed: true, RightDown: true, MouseX: gx + 15, MouseY: gy + 80}, 1024, 768)
	if right.RightPressed || a.playerContext.open {
		t.Fatal("gift box right click leaked to world")
	}
}
