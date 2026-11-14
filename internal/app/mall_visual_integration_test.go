//go:build integration

package app

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
	"github.com/go-gl/gl/v4.5-core/gl"
)

func TestMallOriginalAssetsGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL and original assets")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Title: "mall", Width: 1024, Height: 768, Hidden: true, Debug: true})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
	if err := render.Init(true); err != nil {
		t.Fatal(err)
	}
	arc, err := archive.Open(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	a, err := New(Config{}, arc)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	a.state = game.New()
	a.state.SelfID = 1
	a.state.Slots[0] = proto.CharacterSlot{ID: 1, Name: "Interloper", Level: 60}
	a.state.Mall.Points, a.state.Mall.PointsReady = 1000, true

	if a.itemsItem(25, 1).Count != 255 || a.itemsItem(1, 1).Count != 1 {
		t.Fatal("client stack definitions differ from server")
	}
	if err := a.prepareMall(); err != nil {
		t.Fatal(err)
	}
	if len(a.mall.products[0]) == 0 || len(a.mall.products[3]) == 0 {
		t.Fatal("original product categories empty")
	}
	testingIcons := 0
	for _, products := range a.mall.products {
		for _, p := range products {
			if strings.HasPrefix(p.Code, "T_") {
				testingIcons++
				if p.Icon == nil {
					t.Fatalf("testing product has no native item icon: %s", p.Code)
				}
			}
		}
	}
	t.Logf("%d testing product icons resolved", testingIcons)
	gl.Viewport(0, 0, 1024, 768)
	gl.ClearColor(.06, .06, .06, 1)
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	a.sprites.Begin(1024, 768)
	a.drawMall(1024, 768)
	a.sprites.End()
	if dir := os.Getenv("SHAIYA_TEST_ARTIFACTS"); dir != "" {
		if err := render.SavePNG(filepath.Join(dir, "mall-preview.png"), 1024, 768); err != nil {
			t.Fatal(err)
		}
	}
	if dir := os.Getenv("SHAIYA_TEST_ARTIFACTS"); dir != "" {
		for _, category := range []int{1, 2} {
			a.mallUI.Category = category
			gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
			a.sprites.Begin(1024, 768)
			a.drawMall(1024, 768)
			a.sprites.End()
			if err := render.SavePNG(filepath.Join(dir, fmt.Sprintf("mall-section-%d.png", category+1)), 1024, 768); err != nil {
				t.Fatal(err)
			}
		}
		a.mallUI.Category = 0
	}
	x, y := a.mallUI.Origin(1024, 768)
	click := func(dx, dy int) {
		a.updateMall(&platform.Input{Pressed: true, MouseDown: true, MouseX: x + dx, MouseY: y + dy}, 1024, 768)
		a.updateMall(&platform.Input{Clicked: true, MouseX: x + dx, MouseY: y + dy}, 1024, 768)
	}
	click(30, 175)
	if a.mallUI.Selected != 0 {
		t.Fatal("product not selected")
	}
	click(295, 410)
	if !a.mallConfirm.Visible || a.mall.modal != ui.MallBuy {
		t.Fatal("buy confirmation missing")
	}
	a.updateMallConfirm(&platform.Input{Keys: []platform.Key{platform.KeyEscape}}, 1024, 768)
	if a.mallConfirm.Visible || a.mall.pending != 0 {
		t.Fatal("cancelled purchase sent")
	}
	click(35, 410)
	if !a.mallUI.GiftOpen {
		t.Fatal("gift box did not open")
	}
	a.state.Mall.Stored = map[uint8]proto.MallStoredItem{0: {Slot: 0, Type: 100, TypeID: 251, Count: 1}, 44: {Slot: 44, Type: 25, TypeID: 4, Count: 3}}
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	a.sprites.Begin(1024, 768)
	a.drawMall(1024, 768)
	a.sprites.End()
	if dir := os.Getenv("SHAIYA_TEST_ARTIFACTS"); dir != "" {
		if err := render.SavePNG(filepath.Join(dir, "mall-gifts-preview.png"), 1024, 768); err != nil {
			t.Fatal(err)
		}
	}
	if code := gl.GetError(); code != gl.NO_ERROR {
		t.Fatalf("mall GL error %#x", code)
	}
}
