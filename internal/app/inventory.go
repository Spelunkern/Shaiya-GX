package app

import (
	"fmt"
	"slices"

	"github.com/Spelunkern/Shaiya-GX/internal/data/effect"
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

type inventoryView struct {
	open, selected, pending bool
	bag                     uint8
	hover                   [2]uint8
	hovered                 bool
	source                  [2]uint8
	deadline                float64
	cooldowns               map[uint8]float64
	tap                     map[[2]uint8]float64 // use highlight on a bag cell until this time
	restoration             *potionRestoration
	message                 string
	pressedItem, dragging   bool
	pressX, pressY          int
	preview                 *scene.Character
	previewAppearance       scene.Appearance
	previewLoaded           bool
}

func (v *inventoryView) pressItem(key [2]uint8, in *platform.Input) {
	v.source, v.pressedItem = key, true
	v.dragging, v.selected = false, false
	v.pressX, v.pressY = in.MouseX, in.MouseY
}

func (v *inventoryView) advanceItemDrag(in *platform.Input) {
	if v.pressedItem && !in.Pressed && (in.MouseDown || in.Clicked) &&
		(in.MouseX-v.pressX >= 4 || v.pressX-in.MouseX >= 4 ||
			in.MouseY-v.pressY >= 4 || v.pressY-in.MouseY >= 4) {
		v.pressedItem, v.dragging, v.selected = false, true, true
	}
}

func (v *inventoryView) cancelItemPointer() {
	v.pressedItem, v.dragging, v.selected = false, false, false
}

func (a *App) updateInventory(in *platform.Input, w, h int) bool {
	v := &a.inventory
	if v.bag < 1 || v.bag > 5 {
		v.bag = 1
	}
	for _, r := range in.Chars {
		if r == 'i' || r == 'I' {
			a.toggleInventory()
			return true
		}
	}
	if !v.open {
		return false
	}
	if in.HasKey(platform.KeyEscape) {
		v.open = false
		v.cancelItemPointer()
		return true
	}
	v.advanceItemDrag(in)
	x, y := a.inventoryUI.Origin(w, h)
	captured := panelPointer(in, x, y, ui.InventoryWidth, ui.InventoryHeight)
	var closed bool
	v.bag, closed = a.inventoryUI.Update(in, w, h, v.bag)
	if in.Pressed {
		v.message = ""
	}
	if closed {
		v.open = false
		v.cancelItemPointer()
		return true
	}
	if v.pending || a.shop.pending {
		return captured
	}
	if v.selected && !a.state.Self.Dead {
		for _, r := range in.Chars {
			if r == 'u' || r == 'U' {
				a.useInventoryItem(v.source)
				return true
			}
		}
	}

	merchant := a.state.Entities[a.shop.npcID]
	if v.selected && v.source[0] != 0 && merchant != nil && merchant.NpcType == 1 && !a.auctionBoard(a.shop.npcID) && !a.shop.overlay() {
		for _, r := range in.Chars {
			if r == 'v' || r == 'V' {
				a.sellItem(v.source, 1)
				return true
			}
		}
	}
	v.hover, v.hovered = ui.InventoryHit(in.MouseX-x, in.MouseY-y, v.bag)
	// Clicking uses an item; dragging keeps native bag positions authoritative.
	// Right click uses an item (modern-client habit, not the original's):
	// equipment is worn, everything else is used.
	if in.RightPressed && v.hovered && !a.state.Self.Dead {
		if it, ok := a.state.Inventory[v.hover]; ok {
			if v.hover[0] != 0 && len(inventoryEquipSlots(it.Type)) > 0 {
				a.equipInventoryItem(v.hover)
			} else {
				a.useInventoryItem(v.hover)
			}
		}
		return true
	}
	if in.Pressed && v.hovered && !a.state.Self.Dead {
		if _, ok := a.state.Inventory[v.hover]; ok {
			v.pressItem(v.hover, in)
		}
	}
	if in.Clicked {
		pressedItem, dragged := v.pressedItem, v.dragging
		v.cancelItemPointer()
		key, hit := v.hover, v.hovered
		if tab := ui.InventoryBagTab(in.MouseX-x, in.MouseY-y); dragged && tab >= 0 {
			bag := uint8(tab + 1)
			// Game.exe 0x4f5e31 scans the destination page for its first empty slot.
			if bag != v.source[0] {
				for slot := uint8(0); slot < ui.InventoryBagSlots; slot++ {
					key = [2]uint8{bag, slot}
					if _, occupied := a.state.Inventory[key]; !occupied {
						hit = true
						break
					}
				}
			}
		}
		if hit {
			switch {
			case dragged && key != v.source:
				if a.world == nil {
					return captured
				}
				if err := a.world.Send(proto.ItemMovePacket(v.source[0], v.source[1], key[0], key[1])); err != nil {
					a.backToLoginID(12128)
					return true
				}
				v.pending, v.deadline = true, a.clock+10
			case pressedItem && key == v.source:
				// A left click only selects, except the equipped mount: it is
				// used with the left button (right click keeps working too).
				if key == [2]uint8{0, 13} && !a.state.Self.Dead {
					if it, ok := a.state.Inventory[key]; ok && it.Type == 42 {
						a.useInventoryItem(key)
					}
				}
			}
		} else if dragged && !captured {
			a.noteItemDrop(v.source)
		}
	}
	return captured
}

// drawInventoryDrag draws the item held by the cursor above every window and
// the action bars, which a drop can target.
func (a *App) drawInventoryDrag() {
	v := &a.inventory
	if !v.open || !v.dragging || !v.selected {
		return
	}
	if it, ok := a.state.Inventory[v.source]; ok {
		if d := a.itemsItem(it.Type, it.TypeID); d != nil {
			a.drawItemIcon(d, a.lastMouseX-16, a.lastMouseY-16, 32, [4]float32{1, 1, 1, .85}, &it)
		}
	}
}

func (a *App) drawInventory(w, h int) {
	v := &a.inventory
	if !v.open {
		return
	}
	x, y := a.inventoryUI.Origin(w, h)
	a.clearTooltipUnderPanel(x, y, ui.InventoryWidth, ui.InventoryHeight)
	a.inventoryUI.Draw(a.sprites, a.font, w, h, v.bag, a.state.Details.Money, "")

	a.drawInventoryPreview(w, h)
	for key, it := range a.state.Inventory {
		if key[0] != 0 && key[0] != v.bag {
			continue
		}
		cx, cy, size, ok := ui.InventoryCell(key[0], key[1])
		if !ok {
			continue
		}
		cx, cy = cx+x, cy+y
		if v.selected && key == v.source {
			a.sprites.Draw(a.white, float32(cx), float32(cy), float32(size), float32(size), 0, 0, 1, 1, [4]float32{1, .8, .2, .3})
		}
		if d := a.itemsItem(it.Type, it.TypeID); d != nil {
			a.drawItemIcon(d, cx, cy, size, white, &it)
			a.drawItemCooldown(it, cx, cy, size)
		}
		if a.itemCount(it.Type, it.TypeID, it.Count) > 0 {
			text := fmt.Sprint(it.Count)
			a.font.DrawOutlined(a.sprites, text, cx+size-a.font.Width(text), cy+size-14, white)
		}
	}
	if v.dragging && v.selected {
		return // the dragged icon is drawn last, over every panel and bar
	}

	key, hovered := ui.InventoryHit(a.lastMouseX-x, a.lastMouseY-y, v.bag)
	if !hovered {
		return
	}
	a.queueInventoryItemTooltip(key)
}

func (a *App) queueInventoryItemTooltip(key [2]uint8) {
	it, ok := a.state.Inventory[key]
	if !ok {
		return
	}
	def := a.itemsItem(it.Type, it.TypeID)
	if def == nil {
		return
	}
	a.setTooltip(a.itemTooltip(def, &it, it.Count, itemSellingPrice(def.SellPrice)))
	if key[0] == 0 {
		return
	}
	for _, slot := range inventoryEquipSlots(it.Type) {
		equipped := a.state.Inventory[[2]uint8{0, slot}]
		if equipped.Type == 0 {
			continue
		}
		if equippedDef := a.itemsItem(equipped.Type, equipped.TypeID); equippedDef != nil {
			comparison := a.itemTooltip(equippedDef, &equipped, equipped.Count, "")
			comparison.tooltip = slices.Insert(comparison.tooltip, min(1, len(comparison.tooltip)), itemDetailLine{a.tr(12375), gray})
			a.tooltipEquipped = comparison
			break
		}
	}
}

// Server validation decides whether an item can be used or equipped.
func (a *App) useInventoryItem(key [2]uint8) {
	if a.world == nil || a.state == nil || a.state.Self.Dead || a.inventory.pending || a.shop.pending {
		return
	}
	it, ok := a.state.Inventory[key]
	if !ok {
		return
	}
	if key == [2]uint8{0, 13} {
		a.requestVehicle()
		return
	}
	if a.openAppearanceEditor(key) || a.itemCooldown(it) > 0 {
		return
	}
	if isQuestInventoryItem(it.Type) {
		item := sdata.ItemKey{Type: it.Type, TypeID: it.TypeID}
		if !a.offerItemQuest(key, item, 4) {
			a.offerItemQuest(key, item, 3)
		}
		return
	}
	if err := a.world.Send(a.inventoryUsePacket(key[0], key[1], it)); err != nil {
		a.backToLoginID(12129)
		return
	}
	a.inventory.pending, a.inventory.deadline = true, a.clock+10
	if a.inventory.tap == nil {
		a.inventory.tap = map[[2]uint8]float64{}
	}
	a.inventory.tap[key] = a.clock + .12 // the icon is "tapped" like a pressed shortcut
}
func itemSellingPrice(price uint32) string {
	if price == 0 {
		return ""
	}
	return fmt.Sprintf("Sell: %d gold", price)
}
func (a *App) equipInventoryItem(key [2]uint8) {
	it, ok := a.state.Inventory[key]
	if !ok || key[0] == 0 || a.world == nil {
		return
	}
	slots := inventoryEquipSlots(it.Type)
	if len(slots) == 0 {
		return
	}
	slot := slots[0]
	for _, candidate := range slots {
		if a.state.Inventory[[2]uint8{0, candidate}].Type == 0 {
			slot = candidate
			break
		}
	}
	if err := a.world.Send(proto.ItemMovePacket(key[0], key[1], 0, slot)); err != nil {
		a.backToLoginID(12130)
		return
	}
	a.inventory.selected, a.inventory.dragging = false, false
	a.inventory.pending, a.inventory.deadline = true, a.clock+10
}
func inventoryEquipSlots(typ uint8) []uint8 {
	if effect.WeaponFamily(typ) != 0 {
		return []uint8{5}
	}
	groups := [][]uint8{{16, 31, 66, 72, 81, 87}, {17, 32, 67, 73, 82, 88}, {18, 33, 68, 74, 83, 89}, {20, 35, 70, 76, 85, 91}, {21, 36, 71, 77, 86, 92}, {19, 34, 69, 75, 84, 90}, {24, 39}, {23, 38}, {22, 37}, {40, 41}, {42}}
	slots := [][]uint8{{0}, {1}, {2}, {3}, {4}, {6}, {7}, {8}, {9, 10}, {11, 12}, {13}}
	for i, group := range groups {
		for _, t := range group {
			if typ == t {
				return slots[i]
			}
		}
	}
	return nil
}

func (a *App) inventoryUsePacket(bag, slot uint8, it proto.InventoryItem) []byte {
	if it.Type == 100 && (a.items != nil || a.arc != nil) {
		if def := a.itemsItem(it.Type, it.TypeID); def != nil && def.Special == 41 {
			return proto.TargetResurrectionItemPacket(bag, slot, a.state.Target)
		}
	}
	return proto.UseItemPacket(bag, slot)
}
