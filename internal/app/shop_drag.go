package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

// A cross-panel drop belongs to its source, before the destination can handle
// the release. Game.exe 0x5086c0 buys into bags; 0x5082d0 sells into the market.
// Purchases contain no destination slot: Rapture chooses the resulting stack.
func (a *App) updateCommerceDrag(in *platform.Input, w, h int) bool {
	v := &a.shop
	e := a.state.Entities[v.npcID]
	merchant := e != nil && e.NpcType == 1 && !v.questMode && !a.state.Self.Dead
	busy := v.pending || a.inventory.pending
	mx, my := a.marketUI.Origin(w, h)
	if v.dragging {
		if !merchant || busy || in.HasKey(platform.KeyEscape) || (!in.MouseDown && !in.Clicked) {
			v.dragging = false
			return true
		}
		if in.Clicked {
			v.dragging = false
			cell, hit := ui.MarketHit(in.MouseX-mx, in.MouseY-my)
			sameProduct := hit && v.page*ui.MarketPageSize+cell == int(v.dragIndex)
			ix, iy := a.inventoryUI.Origin(w, h)
			key, bagCell := ui.InventoryHit(in.MouseX-ix, in.MouseY-iy, a.inventory.bag)
			overInventory := a.inventory.open && panelPointer(in, ix, iy, ui.InventoryWidth, ui.InventoryHeight)
			if (sameProduct && !overInventory) || (overInventory && bagCell && key[0] != 0) {
				if sameProduct && !overInventory {
					a.buyItem(v.dragIndex, 1)
				} else {
					a.choosePurchase(v.dragIndex, in, w, h)
				}
			}
		}
		return true
	}
	ix, iy := a.inventoryUI.Origin(w, h)
	overInventory := a.inventory.open && panelPointer(in, ix, iy, ui.InventoryWidth, ui.InventoryHeight)
	if !overInventory && a.inventory.dragging && in.Clicked && merchant && !busy && a.inventory.source[0] != 0 &&
		ui.MarketDropHit(in.MouseX-mx, in.MouseY-my) {
		source := a.inventory.source
		a.inventory.dragging, a.inventory.selected = false, false
		if _, ok := a.state.Inventory[source]; ok {
			a.chooseSale(source, in, w, h)
		}
		return true
	}
	return false
}

func (a *App) drawShopDrag() {
	v := &a.shop
	e := a.state.Entities[v.npcID]
	if !v.dragging || e == nil || a.state.Self.Dead {
		return
	}
	def := a.npcDefs[sdata.NPCKey{Type: e.NpcType, ID: e.TypeID}]
	if int(v.dragIndex) >= len(def.Items) {
		return
	}
	it := def.Items[v.dragIndex]
	if d := a.itemsItem(it.Type, it.TypeID); d != nil {
		a.drawItemIcon(d, a.lastMouseX-16, a.lastMouseY-16, 32, [4]float32{1, 1, 1, .85})
	}
}
