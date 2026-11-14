package app

import (
	"fmt"
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
	materialCount uint16
	quote         forgeQuoteView
	extractSlot   int
	shineUntil    float64
}

func (a *App) blacksmithOpen() bool {
	e := a.state.Entities[a.shop.npcID]
	return a.blacksmithUI != nil && e != nil && e.NpcType == 3 && !a.shop.questMode && !a.state.Self.Dead
}

func (a *App) blacksmithReady() bool {
	v := &a.blacksmith
	return v.selected[0] && v.selected[1] && v.items[1].Type == 30 && a.gemPlacementFailure(v.items[0], v.items[1]) == 0 && v.keys[0] != v.keys[1] &&
		a.state.Inventory[v.keys[0]] == v.items[0] && a.state.Inventory[v.keys[1]] == v.items[1] &&
		(!v.selected[3] || (a.isForgeHammer(v.items[3], 36) && v.keys[3] != v.keys[0] && v.keys[3] != v.keys[1] && a.state.Inventory[v.keys[3]] == v.items[3]))
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

func (a *App) blacksmithExtractReady() bool {
	v := &a.blacksmith
	return v.selected[0] && a.state.Inventory[v.keys[0]] == v.items[0] && v.items[0].Gems != [6]uint8{} && v.selected[3] && a.isForgeHammer(v.items[3], 58)
}

func (a *App) updateBlacksmith(in *platform.Input, w, h int) bool {
	v := &a.blacksmith
	if v.npcID != a.shop.npcID {
		*v = blacksmithView{npcID: a.shop.npcID}
		if !a.inventory.open {
			a.toggleInventory()
		}
	}
	a.syncBlacksmithItems()
	if !a.shop.pending && !a.inventory.pending {
		a.updateForgeQuote()
	}
	x, y := a.blacksmithUI.Origin(w, h)
	captured := panelPointer(in, x, y, ui.BlacksmithWidth, ui.BlacksmithHeight)
	busy := a.shop.pending || a.inventory.pending
	previousMode := a.blacksmithUI.Mode
	closed, link, extract, enchant, compose := a.blacksmithUI.Update(in, w, h, busy, a.blacksmithReady(), a.blacksmithExtractReady(), a.blacksmithEnchantReady(), a.blacksmithComposeReady())
	if previousMode != a.blacksmithUI.Mode {
		*v = blacksmithView{npcID: a.shop.npcID}
		return true
	}
	if (closed || in.HasKey(platform.KeyEscape)) && !busy {
		if err := a.world.Send(proto.NpcTalkPacket(0)); err != nil {
			a.backToLogin("Unable to close NPC conversation.")
			return true
		}
		v.quote = forgeQuoteView{}
		a.shop.npcID = 0
		return true
	}
	if !busy && in.Clicked && !a.inventory.dragging {
		if a.blacksmithUI.Mode == ui.ForgeLapis && v.selected[0] {
			it := a.state.Inventory[v.keys[0]]
			for i, g := range it.Gems {
				sx, sy := x+270+i*39, y+206
				if g != 0 && in.MouseX >= sx && in.MouseX < sx+32 && in.MouseY >= sy && in.MouseY < sy+32 {
					v.extractSlot = i
					v.quote = forgeQuoteView{}
					return true
				}
			}
		}
		if slot := a.blacksmithUI.Hit(in.MouseX-x, in.MouseY-y); slot >= 0 && v.selected[slot] {
			v.selected[slot] = false
			v.quote = forgeQuoteView{}
			return true
		}
	}
	if link && a.gemConfirm != nil {
		a.requestForgeQuote(proto.ForgeQuoteLink, 1, true)
		return true
	}
	if extract && a.gemConfirm != nil {
		a.requestForgeQuote(proto.ForgeQuoteExtract, 2, true)
		return true
	}
	if enchant && a.nearBlacksmith() {
		v.materialCount = 1
		a.sendEnchant(v.keys[1], v.keys[2], v.keys[0], 1)
		return true
	}
	if compose && a.nearBlacksmith() {
		a.sendCompose(v.keys[2], v.keys[0])
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
	slot := a.blacksmithUI.Hit(in.MouseX-x, in.MouseY-y)
	if slot < 0 {
		return false
	}
	key := a.inventory.source
	item, ok := a.state.Inventory[key]
	if ok {
		a.placeBlacksmithItem(slot, key, item)
	}
	a.inventory.dragging, a.inventory.selected = false, false
	return true
}

func (a *App) updateGemConfirm(in *platform.Input, w, h int) {
	if a.gemConfirm.Update(in, w, h) != ui.MessageOK {
		return
	}
	ready := false
	switch a.blacksmith.confirm {
	case 1:
		ready = a.blacksmithReady()
	case 2:
		ready = a.blacksmithExtractReady()
	case 5:
		ready = a.blacksmithEnchantReady()
	default:
		return
	}
	if !a.blacksmithOpen() || !ready {
		a.questMessage("The selected items are no longer available.")
		return
	}
	if !a.blacksmith.quote.known || !a.blacksmith.quote.value.Valid || !a.nearBlacksmith() {
		return
	}
	switch a.blacksmith.confirm {
	case 1:
		if a.blacksmith.selected[3] {
			a.sendGemLink(a.blacksmith.keys[1], a.blacksmith.keys[0], a.blacksmith.keys[3])
		} else {
			a.sendGemLink(a.blacksmith.keys[1], a.blacksmith.keys[0])
		}
	case 2:
		a.sendGemRemove(a.blacksmith.keys[0], a.blacksmith.keys[3])
	case 5:
		a.sendEnchant(a.blacksmith.keys[1], a.blacksmith.keys[2], a.blacksmith.keys[0], a.blacksmith.materialCount)
	}
}

func (a *App) drawBlacksmith(w, h int, name string) {
	a.blacksmithUI.GemSlot = 0
	if a.blacksmithUI.Mode == ui.ForgeLapis && a.blacksmith.selected[0] {
		it := a.state.Inventory[a.blacksmith.keys[0]]
		if d := a.itemsItem(it.Type, it.TypeID); d != nil {
			for i := 0; i < min(int(d.Slot), 6); i++ {
				if it.Gems[i] == 0 {
					a.blacksmithUI.GemSlot = i
					break
				}
			}
		}
	}
	a.blacksmithUI.Draw(a.sprites, a.font, w, h, name, a.shop.pending || a.inventory.pending, a.blacksmithReady(), a.blacksmithExtractReady(), a.blacksmithEnchantReady(), a.blacksmithComposeReady())
	x, y := a.blacksmithUI.Origin(w, h)
	a.clearTooltipUnderPanel(x, y, ui.BlacksmithWidth, ui.BlacksmithHeight)
	if e := a.state.Entities[a.shop.npcID]; e != nil {
		def := a.npcDefs[sdata.NPCKey{Type: e.NpcType, ID: e.TypeID}]
		if portrait := a.npcPortrait(def); portrait != nil {
			a.sprites.Draw(portrait, float32(x+7), float32(y+7), 64, 64, 0, 0, 1, 1, white)
		}
	}
	if a.blacksmithUI.Mode == ui.ForgeLapis {
		sockets := 0
		var gems [6]uint8
		var inactive [6]bool
		if a.blacksmith.selected[0] {
			it := a.state.Inventory[a.blacksmith.keys[0]]
			gems = it.Gems
			_, inactive = a.itemElementState(a.itemsItem(it.Type, it.TypeID), &it)
			if d := a.itemsItem(it.Type, it.TypeID); d != nil {
				sockets = min(int(d.Slot), 6)
			}
		}
		for i := 0; i < 6; i++ {
			sx, sy := x+270+i*39, y+206
			if i >= sockets {
				a.sprites.Draw(a.white, float32(sx), float32(sy), 32, 32, 0, 0, 1, 1, [4]float32{1, 0, 0, .35})
			} else if gems[i] != 0 {
				if i == a.blacksmithExtractionSlot() {
					a.sprites.Draw(a.white, float32(sx), float32(sy), 32, 32, 0, 0, 1, 1, [4]float32{1, .8, .2, .25})
				}
				if d := a.itemsItem(30, gems[i]); d != nil {
					a.drawSocketLapis(d, sx, sy, 32, inactive[i])
				}
			}
		}
	}
	for slot := 0; slot < 4; slot++ {
		if !a.blacksmith.selected[slot] {
			continue
		}
		it, ok := a.state.Inventory[a.blacksmith.keys[slot]]
		if !ok || it.Type != a.blacksmith.items[slot].Type || it.TypeID != a.blacksmith.items[slot].TypeID {
			continue
		}
		if d := a.itemsItem(it.Type, it.TypeID); d != nil {
			sx, sy := a.blacksmithUI.Slot(slot)
			if sx < 0 {
				continue
			}
			a.drawItemIcon(d, x+sx, y+sy, 32, white, &it)
			if it.Count > 1 {
				count := fmt.Sprint(it.Count)
				a.font.DrawOutlined(a.sprites, count, x+sx+32-a.font.Width(count), y+sy+18, white)
			}
		}
	}
	a.drawForgeShine(x, y)
	a.drawBlacksmithDetails(w, h)
}
