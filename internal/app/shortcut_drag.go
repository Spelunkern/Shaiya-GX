package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

type shortcutDrag struct {
	tracking, active bool
	x, y             int
	from             int // -1: learned skill, otherwise flattened bar slot
	reference        proto.QuickSlot
	local            uint8
	skill            proto.LearnedSkill
}

func (a *App) shortcutAt(index int) (*proto.QuickSlot, uint8) {
	for _, q := range a.state.QuickSlots {
		if int(q.Bar)*ui.ActionSlots+int(q.Slot) == index {
			cp := q
			return &cp, 0
		}
	}
	if a.wv != nil && index >= 0 && index < ui.TotalActionSlots {
		return nil, a.wv.localActions[index]
	}
	return nil, 0
}
func (a *App) saveShortcut(index int, source *proto.QuickSlot, local uint8, from int) bool {
	if a.world == nil || !a.state.SkillBarReady || index < 0 || index >= ui.TotalActionSlots {
		return false
	}
	slots := replaceAction(a.state.QuickSlots, uint8(index), source)
	var previous *proto.QuickSlot
	var oldLocal uint8
	if from >= 0 && from != index {
		previous, oldLocal = a.shortcutAt(index)
		slots = replaceAction(slots, uint8(from), previous)
	}
	packet, err := proto.SkillBarPacket(slots)
	if err == nil {
		err = a.world.Send(packet)
	}
	if err != nil {
		a.backToLogin("Unable to save action bar.")
		return false
	}
	a.state.QuickSlots = slots
	if a.wv != nil {
		a.wv.localActions[index] = local
		if from >= 0 && from != index {
			a.wv.localActions[from] = oldLocal
		}
		a.saveLocalActions()
	}
	a.playUI()
	return true
}

// Panels are above the bars. Hidden bars never accept a click or a drop
// through a visible window; modal owners are handled by worldControls first.
func (a *App) barCovered(in *platform.Input, w, h int) bool {
	if a.wv != nil && a.wv.mapView.full {
		return true
	}
	if a.questLog.open && a.questUI != nil {
		x, y := a.questUI.Origin(w, h)
		if pointInside(in, x, y, ui.QuestWidth, ui.QuestHeight) {
			return true
		}
	}
	if a.guild.open && a.guildUI != nil {
		x, y := a.guildUI.Origin(w, h)
		if pointInside(in, x, y, ui.GuildWidth, ui.GuildHeight) {
			return true
		}
	}
	if a.mall.open && a.mallUI != nil {
		x, y := a.mallUI.Origin(w, h)
		if pointInside(in, x, y, ui.MallWidth, ui.MallHeight) {
			return true
		}
	}
	if a.subActionsOpen && a.subActionsUI != nil {
		x, y := a.subActionsUI.Origin(w, h)
		if pointInside(in, x, y, ui.SubActionWidth, ui.SubActionHeight) {
			return true
		}
	}

	if a.exchangeUI != nil && a.state.Exchange.Active {
		x, y := a.exchangeUI.Origin(w, h)
		if pointInside(in, x, y, ui.ExchangeWidth, ui.ExchangeHeight) {
			return true
		}
	}
	if a.blacksmithOpen() && a.blacksmithUI != nil {
		x, y := a.blacksmithUI.Origin(w, h)
		if pointInside(in, x, y, ui.BlacksmithWidth, ui.BlacksmithHeight) {
			return true
		}
	}
	if a.warehouseOpen() && a.bankUI != nil {
		x, y := a.bankUI.Origin(w, h)
		if pointInside(in, x, y, ui.WarehouseWidth, ui.WarehouseHeight) {
			return true
		}
	}
	if e := a.state.Entities[a.shop.npcID]; e != nil {
		x, y, width, height := 0, 0, 0, 0
		switch {
		case a.shop.questMode || e.NpcType != 1 && e.NpcType != 2 && e.NpcType != 3 && e.NpcType != 6 && e.NpcType != 11:
			if a.npcQuestUI != nil {
				x, y = a.npcQuestUI.Origin(w, h)
				width, height = ui.NPCQuestWidth, 391
				if a.npcQuest.selected != 0 {
					height = 512
				}
			}
		case e.NpcType == 2 && a.gateUI != nil:
			x, y = a.gateUI.Origin(w, h)
			width, height = ui.GatekeeperWidth, ui.GatekeeperHeight
		case e.NpcType == 11 && a.guildDirectoryUI != nil:
			x, y = a.guildDirectoryUI.Origin(w, h)
			width, height = ui.GuildDirectoryWidth, ui.GuildDirectoryHeight
		case e.NpcType == 1 && a.marketUI != nil:
			x, y = a.marketUI.Origin(w, h)
			width, height = ui.MarketWidth, ui.MarketHeight
		}
		if pointInside(in, x, y, width, height) {
			return true
		}
	}

	if a.skills.open && a.skillBook != nil {
		x, y := a.skillBook.Origin(w, h)
		if pointInside(in, x, y, ui.SkillBookWidth, ui.SkillBookHeight) {
			return true
		}
	}
	if a.inventory.open && a.inventoryUI != nil {
		x, y := a.inventoryUI.Origin(w, h)
		if pointInside(in, x, y, ui.InventoryWidth, ui.InventoryHeight) {
			return true
		}
	}
	if a.status.open && a.statusUI != nil {
		x, y := a.statusUI.Origin(w, h)
		bw := ui.StatusWidth
		if a.statusUI.Expanded {
			bw += 132
		}
		if pointInside(in, x, y, bw, ui.StatusHeight) {
			return true
		}
	}

	return false
}
func (a *App) updateShortcutDrag(in *platform.Input, w, h int) bool {
	d := &a.skills.drag
	if a.inventory.dragging && a.inventory.selected && in.Clicked && !a.barCovered(in, w, h) {
		if index := a.actionBar.SlotAt(in.MouseX, in.MouseY, w, h); index >= 0 {
			key := a.inventory.source
			if _, ok := a.state.Inventory[key]; ok {
				a.saveShortcut(index, &proto.QuickSlot{Bag: key[0], Number: uint16(key[1])}, 0, -1)
			}
			a.inventory.dragging, a.inventory.selected = false, false
			return true
		}
	}
	if !d.tracking {
		return false
	}
	if a.state.Self.Dead || in.HasKey(platform.KeyEscape) || d.from < 0 && !a.skills.open {
		*d = shortcutDrag{}
		a.actionBar.CancelHeld()
		return true
	}
	if d.from < 0 {
		if current, ok := a.state.Skills[d.skill.Number]; !ok || current.ID != d.skill.ID || current.Level != d.skill.Level {
			*d = shortcutDrag{}
			return true
		}
	} else {
		current, local := a.shortcutAt(d.from)
		if local != d.local || d.local == 0 && (current == nil || *current != d.reference) {
			*d = shortcutDrag{}
			a.actionBar.CancelHeld()
			return true
		}
	}
	if !d.active && (absInt(in.MouseX-d.x) > 4 || absInt(in.MouseY-d.y) > 4) && (in.MouseDown || in.Clicked) {
		d.active = true
		a.actionBar.CancelHeld()
		a.skillBook.CancelHeld()
		a.wv.uiLeft = -2
	}
	if !d.active {
		if in.Clicked || !in.MouseDown {
			*d = shortcutDrag{}
		}
		return false
	}
	if in.Clicked || !in.MouseDown {
		if in.Clicked {
			if index := a.actionBar.SlotAt(in.MouseX, in.MouseY, w, h); index >= 0 {
				if a.barCovered(in, w, h) {
					*d = shortcutDrag{}
					return true
				}
				var source *proto.QuickSlot
				if d.local == 0 {
					cp := d.reference
					source = &cp
				}
				a.saveShortcut(index, source, d.local, d.from)
			} else if d.from >= 0 {
				a.saveShortcut(d.from, nil, 0, -1)
			}
		}
		*d = shortcutDrag{}
	}
	return true
}
func (a *App) drawShortcutDrag() {
	d := a.skills.drag
	if !d.active {
		return
	}
	if d.local > 0 {
		if tex := a.assets.Texture("interface/icon/icon_sub.png"); tex != nil {
			u, v := float32((d.local-1)%4)/4, float32((d.local-1)/4)/4
			a.sprites.Draw(tex, float32(a.lastMouseX-16), float32(a.lastMouseY-16), 32, 32, u, v, u+.25, v+.25, white)
		}
		return
	}
	if d.reference.Bag == 100 {
		if k, ok := a.state.Skills[uint8(d.reference.Number)]; ok {
			a.drawSkillIcon(k, a.lastMouseX-16, a.lastMouseY-16)
		}
	} else {
		it := a.state.Inventory[[2]uint8{d.reference.Bag, uint8(d.reference.Number)}]
		if def := a.itemsItem(it.Type, it.TypeID); def != nil {
			a.drawItemIcon(def, a.lastMouseX-16, a.lastMouseY-16, 32, white, &it)
		}
	}
}
