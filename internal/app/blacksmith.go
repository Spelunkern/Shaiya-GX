package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

type blacksmithView struct {
	npcID         uint32
	keys          [4][2]uint8
	items         [4]proto.InventoryItem
	selected      [4]bool
	confirm       uint8
	quantity      bool
	materialCount uint8
}

func (a *App) blacksmithOpen() bool {
	e := a.state.Entities[a.shop.npcID]
	return a.blacksmithUI != nil && e != nil && e.NpcType == 3 && !a.shop.questMode && !a.state.Self.Dead
}

func (a *App) blacksmithReady() bool {
	v := &a.blacksmith
	return v.selected[0] && v.selected[1] && v.items[1].Type == 30 && v.keys[0] != v.keys[1] &&
		a.state.Inventory[v.keys[0]] == v.items[0] && a.state.Inventory[v.keys[1]] == v.items[1] &&
		(!v.selected[3] || (v.keys[3] != v.keys[0] && v.keys[3] != v.keys[1] && a.state.Inventory[v.keys[3]] == v.items[3]))
}

func (a *App) blacksmithEnchantReady() bool {
	v := &a.blacksmith
	return v.selected[0] && v.selected[1] && v.selected[2] && v.items[1].Type == 95 &&
		v.keys[0] != v.keys[1] && v.keys[0] != v.keys[2] && v.keys[1] != v.keys[2] &&
		a.state.Inventory[v.keys[0]] == v.items[0] && a.state.Inventory[v.keys[1]] == v.items[1] && a.state.Inventory[v.keys[2]] == v.items[2]
}

func (a *App) blacksmithComposeReady() bool {
	v := &a.blacksmith
	return v.selected[0] && v.selected[2] && v.keys[0] != v.keys[2] &&
		a.state.Inventory[v.keys[0]] == v.items[0] && a.state.Inventory[v.keys[2]] == v.items[2]
}

func (a *App) blacksmithRepairReady() bool {
	v := &a.blacksmith
	return v.selected[0] && a.state.Inventory[v.keys[0]] == v.items[0]
}

func (a *App) blacksmithExtractReady() bool {
	v := &a.blacksmith
	return v.selected[0] && a.state.Inventory[v.keys[0]] == v.items[0] && v.items[0].Gems != [6]uint8{}
}

func (a *App) updateBlacksmith(in *platform.Input, w, h int) bool {
	v := &a.blacksmith
	if v.npcID != a.shop.npcID {
		*v = blacksmithView{npcID: a.shop.npcID}
		if !a.inventory.open {
			a.toggleInventory()
		}
	}
	x, y := a.blacksmithUI.Origin(w, h)
	captured := panelPointer(in, x, y, ui.BlacksmithWidth, ui.BlacksmithHeight)
	busy := a.shop.pending || a.inventory.pending
	closed, link, extract, repair, repairAll, enchant, compose := a.blacksmithUI.Update(in, w, h, busy, a.blacksmithReady(), a.blacksmithExtractReady(), a.blacksmithRepairReady(), a.blacksmithEnchantReady(), a.blacksmithComposeReady())
	if (closed || in.HasKey(platform.KeyEscape)) && !busy {
		if err := a.world.Send(proto.NpcTalkPacket(0)); err != nil {
			a.backToLogin("Unable to close NPC conversation.")
			return true
		}
		a.shop.npcID = 0
		return true
	}
	if !busy && in.Clicked && !a.inventory.dragging && ui.BlacksmithHit(in.MouseX-x, in.MouseY-y) == 3 && v.selected[3] {
		v.selected[3] = false
		return true
	}
	if link && a.gemConfirm != nil {
		v.confirm = 1
		a.gemConfirm.Open("Linking can fail, consume the lapis, hammer and gold, and destroy the equipment. Continue?", true)
		return true
	}
	if extract && a.gemConfirm != nil {
		v.confirm = 2
		a.gemConfirm.Open("Extraction can fail, consume the lapis and gold, and destroy the equipment. Continue?", true)
		return true
	}
	if (repair || repairAll) && a.gemConfirm != nil {
		v.confirm = 3
		if repairAll {
			v.confirm = 4
		}
		a.gemConfirm.Open("Repair equipment? The server determines the gold cost.", true)
		return true
	}
	if enchant && a.quantityUI != nil {
		v.quantity = true
		a.quantityUI.Open(in.MouseX, in.MouseY, w, h, min(int(v.items[2].Count), 100))
		return true
	}
	if compose && a.gemConfirm != nil {
		v.confirm = 6
		a.gemConfirm.Open("Recreate equipment attributes? The rune will be consumed and current attributes replaced.", true)
		return true
	}
	return captured
}

func (a *App) updateBlacksmithDrag(in *platform.Input, w, h int) bool {
	if !a.blacksmithOpen() || !a.inventory.dragging || !a.inventory.selected || !in.Clicked || a.shop.pending || a.inventory.pending {
		return false
	}
	if a.wv != nil && a.wv.mapView.full {
		return false
	}
	if a.inventory.open && a.inventoryUI != nil {
		x, y := a.inventoryUI.Origin(w, h)
		if pointInside(in, x, y, ui.InventoryWidth, ui.InventoryHeight) {
			return false
		}
	}
	if a.skills.open && a.skillBook != nil {
		x, y := a.skillBook.Origin(w, h)
		if pointInside(in, x, y, ui.SkillBookWidth, ui.SkillBookHeight) {
			return false
		}
	}
	if a.status.open && a.statusUI != nil {
		x, y := a.statusUI.Origin(w, h)
		width := ui.StatusWidth
		if a.statusUI.Expanded {
			width += 132
		}
		if pointInside(in, x, y, width, ui.StatusHeight) {
			return false
		}
	}
	if a.exchangeUI != nil && a.state.Exchange.Active {
		x, y := a.exchangeUI.Origin(w, h)
		if pointInside(in, x, y, ui.ExchangeWidth, ui.ExchangeHeight) {
			return false
		}
	}
	if a.duelUI != nil && a.state.Duel.Phase == game.DuelWager {
		x, y := a.duelUI.Origin(w, h)
		if pointInside(in, x, y, ui.ExchangeWidth, ui.ExchangeHeight) {
			return false
		}
	}
	if a.guild.open && a.guildUI != nil {
		x, y := a.guildUI.Origin(w, h)
		if pointInside(in, x, y, ui.GuildWidth, ui.GuildHeight) {
			return false
		}
	}
	if a.questLog.open && a.questUI != nil {
		x, y := a.questUI.Origin(w, h)
		if pointInside(in, x, y, ui.QuestWidth, ui.QuestHeight) {
			return false
		}
	}
	x, y := a.blacksmithUI.Origin(w, h)
	slot := ui.BlacksmithHit(in.MouseX-x, in.MouseY-y)
	if slot < 0 {
		return false
	}
	key := a.inventory.source
	item, ok := a.state.Inventory[key]
	if ok && ((slot == 1 && (item.Type == 30 || item.Type == 95) && key[0] != 0) || (slot == 0 && item.Type != 30 && item.Type != 95 && item.Count == 1) || (slot == 2 && key[0] != 0 && item.Count > 0) || (slot == 3 && key[0] != 0 && item.Count > 0 && (item.Type == 25 || item.Type == 44 || item.Type == 100))) {
		a.blacksmith.keys[slot], a.blacksmith.items[slot], a.blacksmith.selected[slot] = key, item, true
	} else {
		a.questMessage("Place equipment, lapis or lapisia, and material in their slots.")
	}
	a.inventory.dragging, a.inventory.selected = false, false
	return true
}

func (a *App) updateGemConfirm(in *platform.Input, w, h int) {
	if a.gemConfirm.Update(in, w, h) != ui.MessageOK {
		return
	}
	if !a.blacksmithOpen() || (a.blacksmith.confirm == 1 && !a.blacksmithReady()) || (a.blacksmith.confirm == 2 && !a.blacksmithExtractReady()) || (a.blacksmith.confirm == 3 && !a.blacksmithRepairReady()) || (a.blacksmith.confirm == 5 && !a.blacksmithEnchantReady()) || (a.blacksmith.confirm == 6 && !a.blacksmithComposeReady()) {
		a.questMessage("The selected items are no longer available.")
		return
	}
	e := a.state.Entities[a.shop.npcID]
	dx, dz := e.X-a.wv.pos[0], e.Z-a.wv.pos[2]
	if dx*dx+dz*dz >= 100 {
		a.questMessage("Move closer to the blacksmith.")
		return
	}
	if a.blacksmith.confirm == 2 {
		a.sendGemRemove(a.blacksmith.keys[0])
	} else if a.blacksmith.confirm == 1 {
		if a.blacksmith.selected[3] {
			a.sendGemLink(a.blacksmith.keys[1], a.blacksmith.keys[0], a.blacksmith.keys[3])
		} else {
			a.sendGemLink(a.blacksmith.keys[1], a.blacksmith.keys[0])
		}
	} else if a.blacksmith.confirm == 3 {
		a.sendRepair(a.blacksmith.keys[0])
	} else if a.blacksmith.confirm == 4 {
		a.sendRepair([2]uint8{255, 255})
	} else if a.blacksmith.confirm == 5 {
		a.sendEnchant(a.blacksmith.keys[1], a.blacksmith.keys[2], a.blacksmith.keys[0], a.blacksmith.materialCount)
	} else if a.blacksmith.confirm == 6 {
		a.sendCompose(a.blacksmith.keys[2], a.blacksmith.keys[0])
	}
}

func (a *App) drawBlacksmith(w, h int, name string) {
	a.blacksmithUI.Draw(a.sprites, a.font, w, h, name, a.shop.pending || a.inventory.pending, a.blacksmithReady(), a.blacksmithExtractReady(), a.blacksmithRepairReady(), a.blacksmithEnchantReady(), a.blacksmithComposeReady())
	x, y := a.blacksmithUI.Origin(w, h)
	if e := a.state.Entities[a.shop.npcID]; e != nil {
		def := a.npcDefs[sdata.NPCKey{Type: e.NpcType, ID: e.TypeID}]
		if portrait := a.npcPortrait(def); portrait != nil {
			a.sprites.Draw(portrait, float32(x+7), float32(y+7), 64, 64, 0, 0, 1, 1, white)
		}
	}
	for slot := 0; slot < 4; slot++ {
		if !a.blacksmith.selected[slot] {
			continue
		}
		it, ok := a.state.Inventory[a.blacksmith.keys[slot]]
		if !ok || it != a.blacksmith.items[slot] {
			continue
		}
		if d := a.itemsItem(it.Type, it.TypeID); d != nil {
			sx, sy := ui.BlacksmithSlot(slot)
			a.drawItemIcon(d, x+sx, y+sy, 32, white)
		}
	}
}
