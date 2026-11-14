//go:build integration

package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
	"github.com/go-gl/gl/v4.5-core/gl"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestElementBadgesAcrossInventoryAndBarsGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL and original assets")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Title: "Element badges", Width: 1024, Height: 768, Hidden: true, Debug: true})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
	if err = render.Init(true); err != nil {
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
	a.lastMouseX, a.lastMouseY = -100, -100
	a.itemsItem(1, 2)
	var gems [8]uint8
	for key, d := range a.items {
		if key.Type == 30 && d.Attribute >= 1 && d.Attribute <= 8 {
			gems[d.Attribute-1] = key.TypeID
		}
	}
	for i, g := range gems {
		if g == 0 {
			t.Fatal("missing native elemental gem", i+1)
		}
	}
	a.inventory.open, a.inventory.bag = true, 1
	ix, iy := a.inventoryUI.Origin(1024, 768)
	var regions []image.Rectangle
	addRegion := func(x, y int) { n := ui.ItemBadgeSize(32); regions = append(regions, image.Rect(x, y, x+n, y+n)) }
	for i, g := range gems {
		a.state.Inventory[[2]uint8{1, uint8(i)}] = proto.InventoryItem{Bag: 1, Slot: uint8(i), Type: 1, TypeID: 2, Count: 1, Gems: [6]uint8{g}}
		x, y, _, _ := ui.InventoryCell(1, uint8(i))
		addRegion(ix+x, iy+y)
	}
	a.state.Inventory[[2]uint8{1, 8}] = proto.InventoryItem{Bag: 1, Slot: 8, Type: 1, TypeID: 2, Count: 1, Gems: [6]uint8{gems[0], gems[1]}}
	x8, y8, _, _ := ui.InventoryCell(1, 8)
	addRegion(ix+x8, iy+y8)
	a.state.Inventory[[2]uint8{1, 9}] = proto.InventoryItem{Bag: 1, Slot: 9, Type: 1, TypeID: 2, Count: 1}
	for _, it := range []proto.InventoryItem{{Bag: 0, Slot: 5, Type: 1, TypeID: 2, Count: 1, Gems: [6]uint8{gems[0]}}, {Bag: 0, Slot: 1, Type: 17, TypeID: 1, Count: 1, Gems: [6]uint8{gems[1]}}} {
		a.state.Inventory[[2]uint8{it.Bag, it.Slot}] = it
		x, y, _, _ := ui.InventoryCell(it.Bag, it.Slot)
		addRegion(ix+x, iy+y)
	}
	a.state.QuickSlots = []proto.QuickSlot{{Bar: 0, Slot: 0, Bag: 1, Number: 0}, {Bar: 0, Slot: 1, Bag: 1, Number: 1}, {Bar: 1, Slot: 0, Bag: 0, Number: 5}}
	a.actionBar.Position[1].Vertical = true
	for _, index := range []int{0, 1, 10} {
		x, y := a.actionBar.SlotOrigin(index, 1024, 768)
		addRegion(x, y)
	}
	out := os.Getenv("SHAIYA_TEST_ARTIFACTS")
	if out == "" {
		out = t.TempDir()
	}
	if err = os.MkdirAll(out, 0755); err != nil {
		t.Fatal(err)
	}
	capture := func(name string, draw func()) image.Image {
		t.Helper()
		gl.Viewport(0, 0, 1024, 768)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		a.sprites.Begin(1024, 768)
		draw()
		a.sprites.End()
		if e := gl.GetError(); e != gl.NO_ERROR {
			t.Fatalf("GL error %x", e)
		}
		path := filepath.Join(out, name+".png")
		if err := render.SavePNG(path, 1024, 768); err != nil {
			t.Fatal(err)
		}
		f, err := os.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		im, err := png.Decode(f)
		if err != nil {
			t.Fatal(err)
		}
		return im
	}
	draw := func() { a.drawInventory(1024, 768); a.drawActionBar(1024, 768) }
	badged := capture("inventory-equipment-and-bars", draw)
	saved := map[*sdata.Item]uint8{}
	for _, d := range a.items {
		saved[d] = d.Attribute
		d.Attribute = 0
	}
	control := capture("inventory-neutral-control", draw)
	for d, v := range saved {
		d.Attribute = v
	}
	counts := make([]int, len(regions))
	for y := 0; y < 768; y++ {
		for x := 0; x < 1024; x++ {
			if badged.At(x, y) == control.At(x, y) {
				continue
			}
			inside := false
			for i, r := range regions {
				if image.Pt(x, y).In(r) {
					counts[i]++
					inside = true
				}
			}
			if !inside {
				t.Fatalf("badge changed pixels outside its icon corner at %d,%d", x, y)
			}
		}
	}
	for i, n := range counts {
		if n < 12 {
			t.Fatalf("missing badge at icon%d: changed pixels%d", i, n)
		}
	}
	// Desaturation must be real grayscale, scoped to one sprite; normal
	// sprites after it must retain exactly the same color as those before it.
	gem := a.itemsItem(30, gems[0])
	gray := capture("active-gray-active-socket-icons", func() {
		a.drawSocketLapis(gem, 40, 40, 32, false)
		a.drawSocketLapis(gem, 100, 40, 32, true)
		a.drawSocketLapis(gem, 160, 40, 32, false)
	})
	colored := 0
	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			r, g, b, _ := gray.At(100+x, 40+y).RGBA()
			if r != g || g != b {
				t.Fatal("inactive elemental gem is not grayscale")
			}
			if gray.At(40+x, 40+y) != gray.At(160+x, 40+y) {
				t.Fatal("grayscale leaked into next sprite")
			}
			r, g, b, _ = gray.At(40+x, 40+y).RGBA()
			if r != g || g != b {
				colored++
			}
		}
	}
	if colored < 12 {
		t.Fatal("active gem lost its color")
	}
	// The exact same instance must retain its badge in storage and offers.
	it := a.state.Inventory[[2]uint8{1, 0}]
	a.state.Warehouse = map[uint8]proto.InventoryItem{0: it}
	a.state.Exchange.Active = true
	a.state.Exchange.Other[0] = proto.ExchangeItem{Type: it.Type, TypeID: it.TypeID, Count: 1, Gems: it.Gems}
	a.exchangeUI.Position.Move(700, 82, 1024, 768, ui.ExchangeWidth, ui.ExchangeHeight)
	capture("warehouse-and-trade", func() { a.drawWarehouse(1024, 768, sdata.NPC{}); a.drawExchange(1024, 768) })
	a.state.Duel.Phase = game.DuelWager
	a.state.Duel.Other[0] = a.state.Exchange.Other[0]
	capture("duel-offer", func() { a.drawDuel(1024, 768) })
}
