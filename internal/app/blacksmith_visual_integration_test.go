//go:build integration

package app

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/go-gl/gl/v4.5-core/gl"
)

func TestBlacksmithVisual(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires OpenGL")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Title: "blacksmith probe", Width: 1024, Height: 734, Hidden: true, Debug: true})
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
	a.state.Inventory[[2]uint8{1, 1}] = proto.InventoryItem{Bag: 1, Slot: 1, Type: 1, TypeID: 1, Count: 1, Quality: 728, Craft: "01020300000001000003", Gems: [6]uint8{1}}
	a.state.Inventory[[2]uint8{1, 2}] = proto.InventoryItem{Bag: 1, Slot: 2, Type: 30, TypeID: 1, Count: 1}
	a.blacksmith.keys = [4][2]uint8{{1, 1}, {1, 2}}
	a.blacksmith.selected = [4]bool{true, true}
	for i, key := range a.blacksmith.keys {
		a.blacksmith.items[i] = a.state.Inventory[key]
	}
	gl.Viewport(0, 0, 1024, 734)
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	a.sprites.Begin(1024, 734)
	a.drawBlacksmith(1024, 734, "Blacksmith")
	item := a.state.Inventory[[2]uint8{1, 1}]
	if def := a.itemsItem(item.Type, item.TypeID); def != nil {
		a.drawItemDetails(def, &item, item.Count, 650, 80, 1024, 734, "Sell: 10 gold")
	}
	a.lastMouseX, a.lastMouseY = 940, 70
	a.drawSharedTooltip(1024, 734)
	a.sprites.End()
	if err := gl.GetError(); err != gl.NO_ERROR {
		t.Fatalf("blacksmith GL error %#x", err)
	}
	dir := os.Getenv("SHAIYA_TEST_ARTIFACTS")
	if dir == "" {
		dir = t.TempDir()
	}
	if err := render.SavePNG(filepath.Join(dir, "blacksmith-preview.png"), 1024, 734); err != nil {
		t.Fatal(err)
	}
	a.loadNPCs()
	def := a.npcDefs[sdata.NPCKey{Type: 2, ID: 32}]
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	a.sprites.Begin(1024, 734)
	a.drawGatekeeper(1024, 734, &game.Entity{NpcType: 2, TypeID: 32}, def)
	a.lastMouseX, a.lastMouseY = 940, 70
	a.drawSharedTooltip(1024, 734)
	a.sprites.End()
	if err := render.SavePNG(filepath.Join(dir, "gatekeeper-preview.png"), 1024, 734); err != nil {
		t.Fatal(err)
	}
}
