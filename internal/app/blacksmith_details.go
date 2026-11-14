package app

import (
	"fmt"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

func (a *App) blacksmithGhosts() map[int]proto.InventoryItem {
	result := map[int]proto.InventoryItem{}
	v := &a.blacksmith
	if a.blacksmithUI.Mode != ui.ForgeEnchant || !v.selected[0] || !v.quote.known {
		return result
	}
	q := v.quote.value
	if !v.selected[2] && q.Material != 0 {
		result[2] = proto.InventoryItem{Type: uint8(q.Material / 1000), TypeID: uint8(q.Material % 1000), Count: 1}
	}
	return result
}
func (a *App) blacksmithHoveredItem(mx, my int) (proto.InventoryItem, bool, bool) {
	v := &a.blacksmith
	for slot := 0; slot < 4; slot++ {
		sx, sy := a.blacksmithUI.Slot(slot)
		if sx < 0 || mx < sx || mx >= sx+32 || my < sy || my >= sy+32 {
			continue
		}
		if v.selected[slot] {
			if it, ok := a.state.Inventory[v.keys[slot]]; ok {
				return it, true, false
			}
		}
		if it, ok := a.blacksmithGhosts()[slot]; ok {
			return it, true, true
		}
	}
	if a.blacksmithUI.Mode == ui.ForgeLapis && v.selected[0] {
		if it, ok := a.state.Inventory[v.keys[0]]; ok {
			_, inactive := a.itemElementState(a.forgeDefinition(it), &it)
			for i, id := range it.Gems {
				sx, sy := 270+i*39, 206
				if id != 0 && mx >= sx && mx < sx+32 && my >= sy && my < sy+32 {
					return proto.InventoryItem{Type: 30, TypeID: id, Count: 1}, true, inactive[i]
				}
			}
		}
	}
	return proto.InventoryItem{}, false, false
}
func (a *App) drawBlacksmithDetails(w, h int) {
	x, y := a.blacksmithUI.Origin(w, h)
	if a.blacksmithUI.Mode == ui.ForgeEnchant && a.blacksmith.selected[0] && a.blacksmith.quote.known {
		q := a.blacksmith.quote.value
		line := fmt.Sprintf("Success: %s%%", forgePercent(q.Rate))
		a.font.Draw(a.sprites, line, x+380-a.font.Width(line)/2, y+349, white, 230)
	}
	for slot, it := range a.blacksmithGhosts() {
		if d := a.itemsItem(it.Type, it.TypeID); d != nil {
			sx, sy := a.blacksmithUI.Slot(slot)
			a.drawItemIcon(d, x+sx, y+sy, 32, [4]float32{.4, .4, .4, .65})
		}
	}
	// Recreation already has a persistent equipment tooltip; avoid duplicating it on hover.
	if a.blacksmithUI.Mode == ui.ForgeRemake {
		sx, sy := a.blacksmithUI.Slot(0)
		if a.lastMouseX >= x+sx && a.lastMouseX < x+sx+32 && a.lastMouseY >= y+sy && a.lastMouseY < y+sy+32 {
			return
		}
	}
	if it, ok, ghost := a.blacksmithHoveredItem(a.lastMouseX-x, a.lastMouseY-y); ok {
		if d := a.itemsItem(it.Type, it.TypeID); d != nil {
			a.drawItemDetails(d, &it, it.Count, 0, 0, w, h, itemSellingPrice(d.SellPrice))
			if ghost {
				a.tooltipMuted = true
				for i := range a.tooltip {
					a.tooltip[i].color = [4]float32{.6, .6, .6, 1}
				}
			}
		}
	}
}

// The recreation comparison is rebuilt from the authoritative instance every
// frame, independently of hover and of the pre-request selection snapshot.
func (a *App) drawRecreationDetails(w, h int) {
	if a.state == nil || !a.blacksmithOpen() || a.blacksmithUI.Mode != ui.ForgeRemake || !a.blacksmith.selected[0] {
		return
	}
	it, ok := a.state.Inventory[a.blacksmith.keys[0]]
	if !ok {
		return
	}
	d := a.itemsItem(it.Type, it.TypeID)
	if d == nil {
		return
	}
	extra := a.tooltipExtra
	rows, item, description, sockets, muted := a.tooltip, a.tooltipItem, a.tooltipDescription, a.tooltipSockets, a.tooltipMuted
	defer func() {
		a.tooltipExtra = extra
		a.tooltip, a.tooltipItem, a.tooltipDescription, a.tooltipSockets, a.tooltipMuted = rows, item, description, sockets, muted
	}()
	a.drawItemDetails(d, &it, it.Count, 0, 0, w, h, itemSellingPrice(d.SellPrice))
	x, y := a.blacksmithUI.Origin(w, h)
	position := [2]int{x + ui.BlacksmithWidth + 8, y + 90}
	a.drawSharedTooltipAt(w, h, &position)
}
