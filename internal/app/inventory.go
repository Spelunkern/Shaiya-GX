package app

import (
	"fmt"

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
	message                 string
	dragging                bool
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
				if v.source == [2]uint8{0, 13} {
					a.requestVehicle()
					v.selected = false
					return true
				}
				if a.openAppearanceEditor(v.source) {
					v.selected = false
					return true
				}
				if err := a.world.Send(proto.UseItemPacket(v.source[0], v.source[1])); err != nil {
					a.backToLogin("Unable to use item.")
					return true
				}
				v.pending, v.deadline, v.selected = true, a.clock+10, false
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
	// A click keeps the item selected; releasing over another cell also
	// supports dragging. Wire positions stay unchanged until the response.
	if in.Pressed && v.hovered && (!v.selected || v.source == v.hover) && !a.state.Self.Dead {
		if _, ok := a.state.Inventory[v.hover]; ok {
			v.source, v.selected, v.dragging = v.hover, true, true
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
			} else if !wasDrag {
				v.selected = false
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
	message := "Select item, then destination."
	if a.warehouseOpen() {
		message = "Drag items between inventory and warehouse."
	} else if a.blacksmithOpen() {
		message = "Drag equipment and lapis to the blacksmith."
	} else if e := a.state.Entities[a.shop.npcID]; e != nil && e.NpcType == 1 {
		message = "V: sell 1 | U: use selected item"
	}
	if v.selected && v.source == [2]uint8{0, 13} {
		message = "U: mount or dismount | /mount"
	}
	if v.message != "" {
		message = v.message
	}
	if v.pending || a.shop.pending {
		message = "Waiting for server..."
	}
	a.inventoryUI.Draw(a.sprites, a.font, w, h, v.bag, a.state.Details.Money, message)
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
		}
		if it.Count > 1 {
			text := fmt.Sprint(it.Count)
			a.font.Draw(a.sprites, text, cx+size-a.font.Width(text), cy+size-14, white, size)
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
	a.drawItemDetails(def, &it, it.Count, left, top, w, h, fmt.Sprintf("Sell: %d gold", def.SellPrice))
}
