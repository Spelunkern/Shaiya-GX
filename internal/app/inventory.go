package app

import (
	"fmt"
	"github.com/Spelunkern/Shaiya-GX/internal/data/effect"

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
	restoration             *potionRestoration
	message                 string
	dragging                bool
	pressX, pressY          int
	preview                 *scene.Character
	previewAppearance       scene.Appearance
	previewLoaded           bool
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
		return true
	}
	x, y := a.inventoryUI.Origin(w, h)
	captured := panelPointer(in, x, y, ui.InventoryWidth, ui.InventoryHeight)
	var closed bool
	v.bag, closed = a.inventoryUI.Update(in, w, h, v.bag)
	if in.Pressed {
		v.message = ""
	}
	if closed {
		v.open = false
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
	if v.selected && v.source[0] != 0 && merchant != nil && merchant.NpcType == 1 && !a.shop.questMode {
		for _, r := range in.Chars {
			if r == 'v' || r == 'V' {
				a.sellItem(v.source, 1)
				return true
			}
		}
	}
	v.hover, v.hovered = ui.InventoryHit(in.MouseX-x, in.MouseY-y, v.bag)
	// Clicking uses an item; dragging keeps native bag positions authoritative.
	if in.RightPressed && v.hovered && !a.state.Self.Dead {
		a.equipInventoryItem(v.hover)
		return true
	}
	if in.Pressed && v.hovered && !a.state.Self.Dead {
		if _, ok := a.state.Inventory[v.hover]; ok {
			v.source, v.selected, v.dragging = v.hover, true, true
			v.pressX, v.pressY = in.MouseX, in.MouseY
		}
	}
	if in.Clicked {
		wasDrag := v.dragging
		v.dragging = false
		if !v.hovered {
			if wasDrag {
				v.selected = false
			}
		} else if v.selected {
			key := v.hover
			if key != v.source {
				v.selected = false
				if err := a.world.Send(proto.ItemMovePacket(v.source[0], v.source[1], key[0], key[1])); err != nil {
					a.backToLogin("Unable to move inventory item.")
					return true
				}
				v.pending, v.deadline = true, a.clock+10
			} else {
				v.selected = false
				if wasDrag && in.MouseX-v.pressX > -4 && in.MouseX-v.pressX < 4 && in.MouseY-v.pressY > -4 && in.MouseY-v.pressY < 4 {
					a.useInventoryItem(key)
				}
			}
		}
	}
	return captured
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
			a.drawItemIcon(d, cx, cy, size, white)
			a.drawItemCooldown(it, cx, cy, size)
		}
		if it.Count > 1 {
			text := fmt.Sprint(it.Count)
			a.font.DrawOutlined(a.sprites, text, cx+size-a.font.Width(text), cy+size-14, white)
		}
	}
	if v.dragging && v.selected {
		if it, ok := a.state.Inventory[v.source]; ok {
			if d := a.itemsItem(it.Type, it.TypeID); d != nil {
				a.drawItemIcon(d, a.lastMouseX-16, a.lastMouseY-16, 32, [4]float32{1, 1, 1, .85})
			}
		}
		return
	}

	key, hovered := ui.InventoryHit(a.lastMouseX-x, a.lastMouseY-y, v.bag)
	if !hovered {
		return
	}
	it, ok := a.state.Inventory[key]
	if !ok {
		return
	}
	def := a.itemsItem(it.Type, it.TypeID)
	if def == nil {
		return
	}
	left, top := max(0, x-304), min(y+260, max(0, h-110))
	a.drawItemDetails(def, &it, it.Count, left, top, w, h, itemSellingPrice(def.SellPrice))
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
	if err := a.world.Send(proto.UseItemPacket(key[0], key[1])); err != nil {
		a.backToLogin("Unable to use item.")
		return
	}
	a.inventory.pending, a.inventory.deadline = true, a.clock+10
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
		a.backToLogin("Unable to equip item.")
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
