package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/data/text"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
	"testing"
)

func continuousForgeApp() *App {
	a := inputApp()
	a.blacksmithUI, _ = ui.NewBlacksmith(inputTextures{})
	a.gemConfirm, _ = ui.NewMessage(inputTextures{})
	a.shop.npcID = 42
	a.blacksmith.npcID = 42
	a.state.Entities[42] = &game.Entity{ID: 42, Kind: game.KindNpc, NpcType: 3}
	a.msg = text.SysMsg{62: "Cannot link", 63: "Duplicate lapis", 64: "Not enough slots"}
	a.items = map[sdata.ItemKey]*sdata.Item{
		{Type: 1, TypeID: 1}:     {Type: 1, TypeID: 1, Slot: 1},
		{Type: 30, TypeID: 1}:    {Type: 30, TypeID: 1, Slot: 1, MinLevel: 1},
		{Type: 30, TypeID: 2}:    {Type: 30, TypeID: 2, Slot: 1, Country: 1},
		{Type: 95, TypeID: 1}:    {Type: 95, TypeID: 1, Slot: 1, MinLevel: 1},
		{Type: 25, TypeID: 43}:   {Type: 25, TypeID: 43},
		{Type: 100, TypeID: 171}: {Type: 100, TypeID: 171, Special: 62},
	}
	return a
}
func TestLapisAndLapisiaTooltipsHaveNoOwnSockets(t *testing.T) {
	a := continuousForgeApp()
	for _, typ := range []uint8{30, 95} {
		it := proto.InventoryItem{Type: typ, TypeID: 1, Count: 1}
		a.drawItemDetails(a.itemsItem(typ, 1), &it, 1, 0, 0, 1024, 768, "")
		if len(a.tooltipSockets) != 0 {
			t.Fatal("consumable has own sockets", typ)
		}
	}
}
func TestRejectedLapisDragUsesNativeReasonAndDoesNotPlace(t *testing.T) {
	a := continuousForgeApp()
	item := proto.InventoryItem{Bag: 1, Slot: 0, Type: 1, TypeID: 1, Count: 1}
	gem := proto.InventoryItem{Bag: 1, Slot: 1, Type: 30, TypeID: 2, Count: 1}
	a.state.Inventory[[2]uint8{1, 0}] = item
	a.state.Inventory[[2]uint8{1, 1}] = gem
	if !a.placeBlacksmithItem(0, [2]uint8{1, 0}, item) {
		t.Fatal("equipment rejected")
	}
	if a.placeBlacksmithItem(1, [2]uint8{1, 1}, gem) || a.blacksmith.selected[1] || len(a.state.Chat) != 1 || a.state.Chat[0].MessageID != 62 {
		t.Fatal("incompatible lapis was placed or missing native refusal")
	}
	gem.TypeID = 1
	item.Gems[0] = 1
	a.blacksmith.items[0] = item
	if a.placeBlacksmithItem(1, [2]uint8{1, 1}, gem) || a.state.Chat[1].MessageID != 63 {
		t.Fatal("duplicate not rejected")
	}
	item.Gems[0] = 2
	a.blacksmith.items[0] = item
	if a.placeBlacksmithItem(1, [2]uint8{1, 1}, gem) || a.state.Chat[2].MessageID != 64 {
		t.Fatal("capacity not enforced")
	}
}
func TestResultRetainsEquipmentAndRemainingConsumableStacks(t *testing.T) {
	for _, success := range []bool{false, true} {
		a := continuousForgeApp()
		a.blacksmith.keys = [4][2]uint8{{1, 0}, {1, 1}, {1, 2}, {1, 3}}
		for i, k := range a.blacksmith.keys {
			it := proto.InventoryItem{Bag: 1, Slot: uint8(i), Type: 25, TypeID: 43, Count: 3}
			if i == 0 {
				it.Type, it.TypeID, it.Count = 1, 1, 1
			}
			a.state.Inventory[k] = it
			a.blacksmith.items[i] = it
			a.blacksmith.selected[i] = true
		}
		it := a.state.Inventory[[2]uint8{1, 1}]
		it.Count = 2
		a.state.Inventory[[2]uint8{1, 1}] = it
		a.finishGemLink(success)
		if a.blacksmith.selected != [4]bool{true, true, true, true} || a.blacksmith.items[1].Count != 2 {
			t.Fatal("result cleared selection or remaining stack")
		}
		delete(a.state.Inventory, [2]uint8{1, 1})
		a.syncBlacksmithItems()
		if a.blacksmith.selected[1] || !a.blacksmith.selected[0] || !a.blacksmith.selected[2] {
			t.Fatal("depletion cleared unrelated slots")
		}
	}
}
func TestRuneCanBeRemovedWithoutConsumingAndRecreationNeedsNoConfirmation(t *testing.T) {
	a := continuousForgeApp()
	a.blacksmithUI.Mode = ui.ForgeRemake
	item := proto.InventoryItem{Bag: 1, Slot: 0, Type: 1, TypeID: 1, Count: 1}
	rune := proto.InventoryItem{Bag: 1, Slot: 1, Type: 100, TypeID: 171, Count: 3}
	a.state.Inventory[[2]uint8{1, 0}] = item
	a.state.Inventory[[2]uint8{1, 1}] = rune
	a.placeBlacksmithItem(0, [2]uint8{1, 0}, item)
	if !a.placeBlacksmithItem(2, [2]uint8{1, 1}, rune) {
		t.Fatal("rune stack rejected")
	}
	x, y := a.blacksmithUI.Origin(1024, 768)
	sx, sy := a.blacksmithUI.Slot(2)
	a.updateBlacksmith(&platform.Input{MouseX: x + sx + 1, MouseY: y + sy + 1, Clicked: true}, 1024, 768)
	if a.blacksmith.selected[2] || a.state.Inventory[[2]uint8{1, 1}].Count != 3 || !a.blacksmith.selected[0] {
		t.Fatal("removing rune consumed it or cleared equipment")
	}
	a.placeBlacksmithItem(2, [2]uint8{1, 1}, rune)
	a.updateBlacksmith(&platform.Input{MouseX: x + 350, MouseY: y + 380, Pressed: true, MouseDown: true}, 1024, 768)
	a.updateBlacksmith(&platform.Input{MouseX: x + 350, MouseY: y + 380, Clicked: true}, 1024, 768)
	if a.gemConfirm.Visible || a.blacksmith.confirm == 6 {
		t.Fatal("recreation requested confirmation")
	}
}

func TestInventoryHeldItemCanSwitchAllBags(t *testing.T) {
	a := inputApp()
	a.inventory.open = true
	source := [2]uint8{1, 0}
	a.state.Inventory[source] = proto.InventoryItem{Bag: 1, Slot: 0, Type: 30, TypeID: 1, Count: 255}
	x, y := a.inventoryUI.Origin(1024, 768)
	a.updateInventory(&platform.Input{MouseX: x + 17, MouseY: y + 285, Pressed: true, MouseDown: true}, 1024, 768)
	for bag := uint8(2); bag <= 5; bag++ {
		a.updateInventory(&platform.Input{MouseX: x + 12 + int(bag-1)*47, MouseY: y + 260, MouseDown: true}, 1024, 768)
		if a.inventory.bag != bag || !a.inventory.dragging || a.inventory.source != source {
			t.Fatal("bag switch lost held source", a.inventory)
		}
	}
}

func TestRecreatableTooltipUsesNativeMaximumAndMaterial(t *testing.T) {
	a := continuousForgeApp()
	d := &sdata.Item{Type: 16, Server: 3, RequiredStats: [6]uint16{0, 0, 0, 0, 13, 25043}, Description: "Description"}
	a.drawItemDetails(d, nil, 1, 0, 0, 1024, 768, "")
	if a.tooltipExtra.maxOJ != 13 || a.tooltipExtra.material != a.itemsItem(25, 43) {
		t.Fatal("native OJ/material not exposed", a.tooltipExtra)
	}
	d.Server = 0
	a.drawItemDetails(d, nil, 1, 0, 0, 1024, 768, "")
	if a.tooltipExtra.maxOJ != 0 || a.tooltipExtra.material != nil {
		t.Fatal("non-recreatable item advertises OJ")
	}
}
