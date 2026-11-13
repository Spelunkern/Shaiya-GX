package app

import (
	"fmt"

	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

type warehouseView struct {
	npcID                       uint32
	page                        int
	source, hover               uint8
	selected, dragging, hovered bool
	guild, guildPending         bool
	sourceMove, destMove        [2]uint8
}

func (a *App) warehouseBag() uint8 {
	if a.bank.guild {
		return proto.GuildBankBag
	}
	return proto.WarehouseBag
}

func (a *App) warehouseItems() map[uint8]proto.InventoryItem {
	if a.bank.guild {
		return a.state.Guild.Bank
	}
	return a.state.Warehouse
}

func (a *App) guildBankAllowed(source, destination [2]uint8) bool {
	g := &a.state.Guild
	if g.ID == 0 || g.Rank == 0 || source == destination {
		return false
	}
	if source[0] == proto.GuildBankBag {
		if g.Rank > 2 {
			a.questMessage("Only guild administrators can withdraw or rearrange items.")
			return false
		}
		item, ok := g.Bank[source[1]]
		if !ok {
			return false
		}
		if source[1] >= 40 || destination[0] == proto.GuildBankBag && destination[1] >= 40 {
			a.questMessage("This guild bank page is unavailable.")
			return false
		}
		if destination[0] != proto.GuildBankBag && item.Type != 94 {
			def := a.itemsItem(item.Type, item.TypeID)
			if def == nil {
				return false
			}
			fee := uint32(float32(uint32(item.Count)*def.SellPrice) * .05)
			if a.state.Details.Money < fee {
				a.questMessage("Not enough gold for the guild bank withdrawal fee.")
				return false
			}
		}
		return true
	}
	if destination[0] != proto.GuildBankBag || source[0] == 0 || destination[1] >= 40 {
		return false
	}
	if g.Rank == 9 {
		a.questMessage("Your guild rank cannot deposit items.")
		return false
	}
	if _, occupied := g.Bank[destination[1]]; occupied && g.Rank > 2 {
		a.questMessage("Only guild administrators can replace stored items.")
		return false
	}
	item, ok := a.state.Inventory[source]
	if !ok {
		return false
	}
	if item.Type == 27 || item.Type == 28 || item.Type == 29 {
		a.questMessage("Quest items cannot be stored in the guild bank.")
		return false
	}
	def := a.itemsItem(item.Type, item.TypeID)
	if def == nil || def.ReqOG != 0 {
		a.questMessage("This item cannot be stored in the guild bank.")
		return false
	}
	return true
}

func (a *App) warehouseOpen() bool {
	e := a.state.Entities[a.shop.npcID]
	return a.bankUI != nil && e != nil && e.NpcType == 6 && !a.shop.questMode && !a.state.Self.Dead
}
func (a *App) closeWarehouse() {
	a.bank.selected, a.bank.dragging = false, false
	if err := a.world.Send(proto.NpcTalkPacket(0)); err != nil {
		a.backToLogin("Unable to close warehouse.")
		return
	}
	a.shop.npcID = 0
}
func (a *App) updateWarehouse(in *platform.Input, w, h int) bool {
	v := &a.bank
	if v.npcID != a.shop.npcID {
		*v = warehouseView{npcID: a.shop.npcID}
		if !a.inventory.open {
			a.toggleInventory()
		}
	}
	x, y := a.bankUI.Origin(w, h)
	captured := panelPointer(in, x, y, ui.WarehouseWidth, ui.WarehouseHeight)
	if a.bankUI.UpdateGuildSwitch(in, w, h, a.state.Guild.ID != 0 && !a.inventory.pending && !a.shop.pending) {
		v.guild = !v.guild
		v.page = 0
		v.selected = false
		v.dragging = false
		return true
	}
	if v.guild && a.state.Guild.ID == 0 {
		v.guild = false
		v.page = 0
		v.selected = false
		v.dragging = false
	}
	page, close := a.bankUI.Update(in, w, h, v.page)
	if page != v.page {
		v.page = page
		v.selected, v.dragging = false, false
	}
	if close || in.HasKey(platform.KeyEscape) {
		a.closeWarehouse()
		return true
	}
	v.hover, v.hovered = ui.WarehouseHit(in.MouseX-x, in.MouseY-y, v.page)
	if (!v.guild && !a.state.WarehouseReady) || a.inventory.pending || a.shop.pending {
		return captured
	}
	if in.Pressed && v.hovered && !v.selected {
		if _, ok := a.warehouseItems()[v.hover]; ok {
			v.source, v.selected, v.dragging = v.hover, true, true
		}
	}
	if in.Clicked {
		wasDrag := v.dragging
		v.dragging = false
		if v.hovered && v.selected {
			if v.source != v.hover {
				bag := a.warehouseBag()
				a.moveWarehouseItem([2]uint8{bag, v.source}, [2]uint8{bag, v.hover})
				v.selected = false
			} else if !wasDrag {
				v.selected = false
			}
		} else if wasDrag {
			v.selected = false
		}
	}
	return captured
}
func (a *App) moveWarehouseItem(source, destination [2]uint8) {
	if !a.warehouseOpen() || (!a.bank.guild && !a.state.WarehouseReady) || a.inventory.pending || a.shop.pending {
		return
	}
	if a.bank.guild {
		if !a.guildBankAllowed(source, destination) {
			return
		}
		a.bank.guildPending = true
		a.bank.sourceMove, a.bank.destMove = source, destination
	}
	if err := a.world.Send(proto.ItemMovePacket(source[0], source[1], destination[0], destination[1])); err != nil {
		a.backToLogin("Unable to move warehouse item.")
		return
	}
	a.inventory.pending, a.inventory.deadline = true, a.clock+10
	a.inventory.selected, a.inventory.dragging = false, false
	a.bank.selected, a.bank.dragging = false, false
}

// Cross-panel release belongs to its source before the destination handles it.
func (a *App) updateWarehouseDrag(in *platform.Input, w, h int) bool {
	v := &a.bank
	if !a.warehouseOpen() {
		v.selected, v.dragging = false, false
		return false
	}
	if v.dragging && (in.HasKey(platform.KeyEscape) || (!in.MouseDown && !in.Clicked)) {
		v.dragging, v.selected = false, false
		return true
	}
	bx, by := a.bankUI.Origin(w, h)
	ix, iy := a.inventoryUI.Origin(w, h)
	overInventory := a.inventory.open && pointInside(in, ix, iy, ui.InventoryWidth, ui.InventoryHeight)
	if in.Clicked && v.selected && overInventory {
		key, ok := ui.InventoryHit(in.MouseX-ix, in.MouseY-iy, a.inventory.bag)
		if ok && key[0] != 0 {
			a.moveWarehouseItem([2]uint8{a.warehouseBag(), v.source}, key)
		} else {
			v.selected, v.dragging = false, false
		}
		return true
	}
	if in.Clicked && a.inventory.selected && !overInventory {
		slot, ok := ui.WarehouseHit(in.MouseX-bx, in.MouseY-by, v.page)
		if ok && a.inventory.source[0] != 0 {
			a.moveWarehouseItem(a.inventory.source, [2]uint8{a.warehouseBag(), slot})
			return true
		}
	}
	return false
}
func (a *App) drawWarehouse(w, h int, def sdata.NPC) {
	v := &a.bank
	x, y := a.bankUI.Origin(w, h)
	a.clearTooltipUnderPanel(x, y, ui.WarehouseWidth, ui.WarehouseHeight)
	a.bankUI.Draw(a.sprites, a.font, w, h, v.page, def.Name, def.Welcome, a.assets.Texture("interface/NpcFace/55.png"))
	a.bankUI.DrawGuildSwitch(a.sprites, a.font, w, h, a.state.Guild.ID != 0, v.guild)
	if v.guild {
		a.font.Draw(a.sprites, "Guild Bank", x+90, y+84, white, 135)
	}
	for slot, it := range a.warehouseItems() {
		cx, cy, ok := ui.WarehouseCell(v.page, slot)
		if !ok {
			continue
		}
		cx, cy = cx+x, cy+y
		if v.selected && slot == v.source {
			a.sprites.Draw(a.white, float32(cx), float32(cy), 32, 32, 0, 0, 1, 1, [4]float32{1, .8, .2, .3})
		}
		if d := a.itemsItem(it.Type, it.TypeID); d != nil {
			a.drawItemIcon(d, cx, cy, 32, white)
		}
		if it.Count > 1 {
			text := fmt.Sprint(it.Count)
			a.font.Draw(a.sprites, text, cx+32-a.font.Width(text), cy+18, white, 32)
		}
	}
	if v.dragging && v.selected {
		if it, ok := a.warehouseItems()[v.source]; ok {
			if d := a.itemsItem(it.Type, it.TypeID); d != nil {
				a.drawItemIcon(d, a.lastMouseX-16, a.lastMouseY-16, 32, [4]float32{1, 1, 1, .85})
			}
		}
		return
	}
	if hover, hovered := ui.WarehouseHit(a.lastMouseX-x, a.lastMouseY-y, v.page); hovered {
		if it, ok := a.warehouseItems()[hover]; ok {
			if d := a.itemsItem(it.Type, it.TypeID); d != nil {
				feeText := "Withdrawal fee: 5% of the stack's sell value."
				if it.Type == 94 {
					feeText = "Free withdrawal."
				}
				a.drawItemDetails(d, &it, it.Count, min(x+ui.WarehouseWidth+4, max(0, w-304)), min(y+140, max(0, h-110)), w, h, feeText)
			}
		}
	}
}
