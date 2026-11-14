//go:build integration

package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/testworld"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
	"os"
	"strings"
	"testing"
	"time"
)

func TestForgeContinuousAgainstServer(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_FORGE_CONTINUOUS") != "1" {
		t.Skip("requires isolated forge fixtures")
	}
	st, world, send, wait := testworld.Enter(t)
	a := inputApp()
	a.state, a.world = st, world
	a.blacksmithUI, _ = ui.NewBlacksmith(inputTextures{})
	a.gemConfirm, _ = ui.NewMessage(inputTextures{})
	arc, err := archive.Open(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	a.arc = arc
	raw, err := os.ReadFile(os.Getenv("SHAIYA_TEST_ITEM_FILE"))
	if err != nil {
		t.Fatal(err)
	}
	a.items, err = sdata.ParseItems(raw)
	if err != nil {
		t.Fatal(err)
	}
	a.wv.pos[0], a.wv.pos[1], a.wv.pos[2] = st.Self.X, st.Self.Y, st.Self.Z
	send(proto.ChatPacket(proto.ChannelNormal, false, "/nmake 3 2"))
	var smith uint32
	wait(func() bool {
		for id, e := range st.Entities {
			dx, dz := e.X-st.Self.X, e.Z-st.Self.Z
			if e.Kind == game.KindNpc && e.NpcType == 3 && dx*dx+dz*dz < 100 {
				smith = id
				return true
			}
		}
		return false
	})
	send(proto.NpcTalkPacket(smith))
	a.shop.npcID = smith
	a.worldControls(&platform.Input{}, 1024, 768)
	ix, iy := a.inventoryUI.Origin(1024, 768)
	bx, by := a.blacksmithUI.Origin(1024, 768)
	click := func(x, y int) {
		a.worldControls(&platform.Input{MouseX: x, MouseY: y, Pressed: true, MouseDown: true}, 1024, 768)
		a.worldControls(&platform.Input{MouseX: x, MouseY: y, Clicked: true}, 1024, 768)
	}
	pump := func(done func() bool) {
		t.Helper()
		deadline := time.Now().Add(10 * time.Second)
		for !done() {
			a.pumpWorld()
			if a.state == nil || time.Now().After(deadline) {
				t.Fatal("forge timeout")
			}
			time.Sleep(time.Millisecond)
		}
	}
	drag := func(key [2]uint8, slot int) {
		t.Helper()
		sx, sy, _, _ := ui.InventoryCell(key[0], key[1])
		dx, dy := a.blacksmithUI.Slot(slot)
		a.worldControls(&platform.Input{MouseX: ix + sx + 2, MouseY: iy + sy + 2, Pressed: true, MouseDown: true}, 1024, 768)
		a.worldControls(&platform.Input{MouseX: bx + dx + 2, MouseY: by + dy + 2, Clicked: true}, 1024, 768)
		if !a.blacksmith.selected[slot] || a.blacksmith.keys[slot] != key {
			t.Fatalf("drag rejected slot%d key%v chat%v", slot, key, st.Chat)
		}
	}
	confirm := func() {
		t.Helper()
		pump(func() bool { return a.gemConfirm.Visible })
		a.worldControls(&platform.Input{Keys: []platform.Key{platform.KeyEnter}}, 1024, 768)
		if !a.shop.pending {
			t.Fatal("no authoritative request after confirmation")
		}
		pump(func() bool { return !a.shop.pending })
	}
	target := [2]uint8{1, 0}
	// Move out and back by hovering the native bag tab while holding the item.
	for _, bag := range []uint8{2, 1} {
		sx, sy, _, _ := ui.InventoryCell(1, 0)
		a.worldControls(&platform.Input{MouseX: ix + sx + 2, MouseY: iy + sy + 2, Pressed: true, MouseDown: true}, 1024, 768)
		a.worldControls(&platform.Input{MouseX: ix + 12 + int(bag-1)*47, MouseY: iy + 260, MouseDown: true}, 1024, 768)
		if a.inventory.bag != bag || !a.inventory.dragging {
			t.Fatal("held-item bag switch failed")
		}
		a.worldControls(&platform.Input{MouseX: ix + sx + 2, MouseY: iy + sy + 2, Clicked: true}, 1024, 768)
		pump(func() bool { return !a.inventory.pending })
		if _, ok := st.Inventory[[2]uint8{bag, 0}]; !ok {
			t.Fatal("cross-bag move not confirmed")
		}
	}
	initialMoney := st.Details.Money
	drag(target, 0)
	drag([2]uint8{1, 4}, 1)
	drag([2]uint8{1, 5}, 3)
	click(bx+300, by+380)
	confirm()
	if st.Inventory[target].Gems[0] != 1 || !a.blacksmith.selected[0] || !a.blacksmith.selected[1] || st.Inventory[[2]uint8{1, 4}].Count != 1 {
		t.Fatal("link cleared selections or stack")
	}
	// Extraction can succeed or fail, but only its chosen socket is removed.
	drag([2]uint8{1, 6}, 3)
	click(bx+435, by+380)
	confirm()
	if st.Inventory[target].Gems[0] != 0 || !a.blacksmith.selected[0] || st.Inventory[[2]uint8{1, 6}].Count != 1 {
		t.Fatal("extraction lost equipment or hammer stack")
	}
	drag([2]uint8{1, 5}, 3)
	click(bx+300, by+380)
	confirm()
	if !a.blacksmith.selected[0] || a.blacksmith.selected[1] {
		t.Fatal("depletion did not clear only the lapis slot")
	}
	click(bx+82, by+305)
	if a.blacksmithUI.Mode != ui.ForgeEnchant {
		t.Fatal("enchant tab failed")
	}
	drag(target, 0)
	drag([2]uint8{1, 1}, 1)
	drag([2]uint8{1, 2}, 2)
	for range 2 {
		click(bx+350, by+380)
		if a.blacksmith.quantity || a.gemConfirm.Visible || !a.shop.pending {
			t.Fatal("enchant must execute immediately with one material")
		}
		pump(func() bool { return !a.shop.pending })
		if !a.blacksmith.selected[0] || !a.blacksmith.selected[1] || !a.blacksmith.selected[2] {
			t.Fatal("enchant cleared a remaining stack")
		}
	}
	if st.Inventory[[2]uint8{1, 1}].Count != 1 || st.Inventory[[2]uint8{1, 2}].Count != 1 {
		t.Fatal("wrong enchant consumption")
	}
	click(bx+82, by+335)
	if a.blacksmithUI.Mode != ui.ForgeRemake {
		t.Fatal("recreate tab failed")
	}
	drag(target, 0)
	drag([2]uint8{1, 3}, 2)
	for range 2 {
		click(bx+350, by+380)
		if a.gemConfirm.Visible || !a.shop.pending {
			t.Fatal("recreation was not immediate")
		}
		pump(func() bool { return !a.shop.pending })
		if !a.blacksmith.selected[0] || !a.blacksmith.selected[2] {
			t.Fatal("recreation cleared stack/equipment")
		}
	}
	if st.Inventory[[2]uint8{1, 3}].Count != 1 || strings.Trim(st.Inventory[target].Craft, "0") == "" {
		t.Fatal("recreation not applied")
	}
	// Custom perfect and random STR runes preserve all sibling pairs.
	for _, fixture := range []struct {
		slot    uint8
		perfect bool
	}{{7, true}, {8, false}} {
		before, _ := parseCraftValues(st.Inventory[target].Craft)
		drag([2]uint8{1, fixture.slot}, 2)
		click(bx+350, by+380)
		pump(func() bool { return !a.shop.pending })
		after, _ := parseCraftValues(st.Inventory[target].Craft)
		maxOJ := a.itemsItem(1, 2).RequiredStats[4]
		if after.stats[0] < 1 || after.stats[0] > maxOJ || fixture.perfect && after.stats[0] != maxOJ {
			t.Fatal("custom rune wrong roll", after)
		}
		for i := 1; i < 9; i++ {
			if after.stats[i] != before.stats[i] {
				t.Fatal("custom rune changed sibling", i)
			}
		}
		if after.enchant != before.enchant || st.Inventory[[2]uint8{1, fixture.slot}].Count != 2 {
			t.Fatal("custom rune consume/enchant failed")
		}
	}
	// Buy independent gem variants and a new rune through the actual mall packet.
	rawCash, err := arc.ReadFile("Cash.SData")
	if err != nil {
		t.Fatal(err)
	}
	products, err := sdata.ParseCash(rawCash)
	if err != nil {
		t.Fatal(err)
	}
	for _, typ := range []uint32{30, 95, 101} {
		found := false
		for _, product := range products {
			id := product.Items[0].ItemID
			if product.Bag == 0 || id/1000 != typ || typ == 30 && id%1000 == 1 || typ == 95 && id%1000 == 1 || typ == 101 && id%1000 != 18 {
				continue
			}
			single := true
			for _, entry := range product.Items[1:] {
				if entry.ItemID != 0 {
					single = false
				}
			}
			if !single {
				continue
			}
			if product.Items[0].Count != 255 {
				t.Fatal("mall does not display 255", id)
			}
			send(proto.MallBuyPacket(product.Code))
			pump(func() bool {
				total := 0
				for _, it := range st.Inventory {
					if uint32(it.Type)*1000+uint32(it.TypeID) == id {
						total += int(it.Count)
					}
				}
				return total == 255
			})
			found = true
			break
		}
		if !found {
			t.Fatal("no 255 mall fixture for", typ)
		}
	}
	expected := initialMoney - 3*4095 - 2*2500000
	if st.Details.Money != expected {
		t.Fatalf("gold %d, expected%d", st.Details.Money, expected)
	}
	t.Logf("native UI: repeated link/extract/enchant/recreate, retained stacks/equipment, gold=%d craft=%s", st.Details.Money, st.Inventory[target].Craft)
}
