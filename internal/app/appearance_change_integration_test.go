//go:build integration

package app

import (
	"os"
	"runtime"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/session"
	"github.com/go-gl/gl/v4.5-core/gl"
)

func TestConfirmedAppearanceRefreshesActorsAndPortraitGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL and original assets")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Title: "appearance change", Width: 96, Height: 96, Hidden: true})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
	if err := render.Init(false); err != nil {
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
	defer func() { a.world = nil; a.Close() }()
	if err := a.Preview("world-status"); err != nil {
		t.Fatal(err)
	}
	a.state.Slots[0] = proto.CharacterSlot{ID: 1, Hair: 1, Face: 1, Height: 2}
	a.refreshSelf(a.wv)
	first := a.wv.self
	if first == nil || a.hudPortrait(96, 96) == nil {
		t.Fatal("initial appearance missing")
	}
	initialPortrait := a.wv.portraitAppearance
	a.items = map[sdata.ItemKey]*sdata.Item{{Type: 25, TypeID: 7}: {Special: 44}}
	a.inventory.open = true
	a.state.Inventory[[2]uint8{1, 2}] = proto.InventoryItem{Bag: 1, Slot: 2, Type: 25, TypeID: 7, Count: 1}
	if !a.openAppearanceEditor([2]uint8{1, 2}) {
		t.Fatal("editor did not open")
	}
	a.appearanceUI.Update(&platform.Input{MouseX: 203, MouseY: 26, Pressed: true, MouseDown: true}, 96, 96)
	a.appearanceUI.Update(&platform.Input{MouseX: 203, MouseY: 26, Clicked: true}, 96, 96)
	if a.appearanceUI.Values[0] != 2 {
		t.Fatal("hair control did not advance")
	}
	a.sprites.Begin(96, 96)
	a.drawInventoryPreview(96, 96)
	a.appearanceUI.Draw(a.sprites, a.font, 96, 96)
	a.sprites.End()
	if a.inventory.preview == nil || a.inventory.previewAppearance.Hair != 2 || a.state.Slots[0].Hair != 1 {
		t.Fatal("draft preview changed server state or failed to render")
	}
	a.updateAppearanceEditor(&platform.Input{Keys: []platform.Key{platform.KeyEscape}}, 96, 96)
	a.state.Entities[42] = &game.Entity{ID: 42, Kind: game.KindUser, Shape: &proto.UserShape{ID: 42, Hair: 1, Face: 1, Height: 2}}
	a.prepareRemotePlayers()
	remote := a.wv.actors[42]
	if remote == nil || remote.character == nil {
		t.Fatal("initial remote actor missing")
	}
	packets := make(chan []byte, 2)
	a.world = &session.World{Packets: packets}
	change := func(id uint32) []byte {
		w := proto.NewWriter(proto.ChangeAppearance)
		w.U32(id)
		w.U8(2)
		w.U8(1)
		w.U8(2)
		w.U8(0)
		return w.Bytes()
	}
	packets <- change(1)
	packets <- change(42)
	a.pumpWorld()
	if a.wv.self == nil || a.wv.self == first {
		t.Fatal("self model did not refresh")
	}
	if a.hudPortrait(96, 96) == nil || a.wv.portraitAppearance == initialPortrait {
		t.Fatal("portrait did not refresh")
	}
	a.prepareRemotePlayers()
	if a.wv.actors[42].character == nil || a.wv.actors[42] == remote || a.wv.actors[42].shape.Hair != 2 {
		t.Fatal("remote model did not refresh")
	}
}

func TestAppearanceEditorLayoutGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL and original assets")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Title: "appearance editor", Width: 1024, Height: 768, Hidden: true})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
	if err := render.Init(false); err != nil {
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
	if err := a.Preview("world-status"); err != nil {
		t.Fatal(err)
	}
	a.state.Slots[0] = proto.CharacterSlot{ID: 1, Hair: 1, Face: 1, Height: 2}
	a.items = map[sdata.ItemKey]*sdata.Item{{Type: 25, TypeID: 7}: {Special: 45}}
	a.state.Inventory[[2]uint8{1, 2}] = proto.InventoryItem{Bag: 1, Slot: 2, Type: 25, TypeID: 7, Count: 1}
	if !a.openAppearanceEditor([2]uint8{1, 2}) {
		t.Fatal("editor did not open")
	}
	gl.Viewport(0, 0, 1024, 768)
	gl.Disable(gl.SCISSOR_TEST)
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	a.sprites.Begin(1024, 768)
	a.drawInventory(1024, 768)
	a.appearanceUI.Draw(a.sprites, a.font, 1024, 768)
	a.sprites.End()
	if file := os.Getenv("SHAIYA_APPEARANCE_SCREENSHOT"); file != "" {
		if err := render.SavePNG(file, 1024, 768); err != nil {
			t.Fatal(err)
		}
	}
}
