package app

import (
	"slices"
	"strings"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/data/text"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

func TestInventoryGearTooltipUsesOccupiedEquipmentSlots(t *testing.T) {
	a := inputApp()
	a.msg = text.SysMsg{5219: "Sale price"}
	a.items = map[sdata.ItemKey]*sdata.Item{
		{Type: 21, TypeID: 1}: {Type: 21, TypeID: 1, Name: "Bag Boots", SellPrice: 15},
		{Type: 21, TypeID: 2}: {Type: 21, TypeID: 2, Name: "Worn Boots", SellPrice: 30, Slot: 1},
		{Type: 30, TypeID: 3}: {Type: 30, TypeID: 3, Name: "Craft Lapis", Stats: [6]uint16{7}},
		{Type: 25, TypeID: 1}: {Type: 25, TypeID: 1, Name: "Potion"},
		{Type: 19, TypeID: 1}: {Type: 19, TypeID: 1, Name: "Shield"},
		{Type: 22, TypeID: 1}: {Type: 22, TypeID: 1, Name: "Bag Ring"},
		{Type: 22, TypeID: 2}: {Type: 22, TypeID: 2, Name: "Left Ring"},
		{Type: 22, TypeID: 3}: {Type: 22, TypeID: 3, Name: "Right Ring"},
	}
	put := func(bag, slot, typ, id uint8) {
		a.state.Inventory[[2]uint8{bag, slot}] = proto.InventoryItem{Bag: bag, Slot: slot, Type: typ, TypeID: id, Count: 1}
	}
	put(1, 0, 21, 1)
	put(0, 4, 21, 2)
	worn := a.state.Inventory[[2]uint8{0, 4}]
	worn.Gems[0] = 3
	a.state.Inventory[[2]uint8{0, 4}] = worn
	a.queueInventoryItemTooltip([2]uint8{1, 0})
	p, c := a.tooltip, a.tooltipEquipped
	if len(p) == 0 || p[0].text != "Bag Boots" || len(c.tooltip) < 2 || c.tooltip[0].text != "Worn Boots" || c.tooltip[1].text != "Equipped" {
		t.Fatalf("wrong tooltip pair: %v, %v", p, c.tooltip)
	}
	if len(a.tooltipSockets) != 0 || len(c.tooltipSockets) != 1 || c.tooltipSockets[0].definition != a.itemsItem(30, 3) {
		t.Fatal("the equipped item's linked lapis did not stay on its own card")
	}
	for _, row := range c.tooltip {
		if strings.HasPrefix(row.text, "Sale price") {
			t.Fatal("the comparison labeled worn gear for sale")
		}
	}
	if !slices.ContainsFunc(p, func(row itemDetailLine) bool { return row.text == "Sale price : 15" }) {
		t.Fatal("the primary tooltip lost the bag item's selling price")
	}

	put(1, 1, 25, 1)
	put(1, 2, 19, 1)
	for _, key := range [][2]uint8{{1, 1}, {1, 2}, {0, 4}} {
		a.queueInventoryItemTooltip([2]uint8{1, 0})
		a.queueInventoryItemTooltip(key)
		if len(a.tooltip) == 0 || len(a.tooltipEquipped.tooltip) != 0 {
			t.Fatalf("unexpected comparison at %v", key)
		}
	}

	put(1, 3, 22, 1)
	put(0, 10, 22, 3)
	for _, want := range []string{"Right Ring", "Left Ring"} {
		a.queueInventoryItemTooltip([2]uint8{1, 3})
		rows := a.tooltipEquipped.tooltip
		if len(rows) == 0 || rows[0].text != want {
			t.Fatalf("accessory comparison got %v, want %q", rows, want)
		}
		put(0, 9, 22, 2)
	}
	a.lastMouseX, a.lastMouseY = 50, 50
	a.clearTooltipUnderPanel(49, 49, 5, 5)
	if len(a.tooltip) != 0 || len(a.tooltipEquipped.tooltip) != 0 {
		t.Fatal("a covering panel left an inventory tooltip visible")
	}
}

func TestTooltipPairFitsViewportAndStaysBesidePointer(t *testing.T) {
	for _, tc := range []struct {
		name           string
		x, y, w, h     int
		height         int
		equippedOnLeft bool
	}{
		{"right-edge", 1000, 760, 1024, 768, 320, true},
		{"left-edge", 5, 10, 1024, 768, 320, false},
		{"small-viewport", 317, 177, 320, 180, 450, true},
		{"narrow-viewport", 10, 180, 300, 220, 150, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			primary, equipped, top, scale := tooltipPairOrigin(tc.x, tc.y, tc.w, tc.h, tc.height)
			left, right := min(primary, equipped), max(primary, equipped)
			cardWidth := tooltipCardWidth * scale
			if (equipped < primary) != tc.equippedOnLeft || scale <= 0 || scale > 1 ||
				left < 0 || right-left < cardWidth || right+cardWidth > float32(tc.w)+.01 ||
				top < 0 || top+float32(tc.height)*scale > float32(tc.h)+.01 {
				t.Fatalf("tooltip cards do not fit together: primary=%g equipped=%g top=%g scale=%g", primary, equipped, top, scale)
			}
		})
	}
}
