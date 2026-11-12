package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

type appearanceEdit struct {
	source    [2]uint8
	item      proto.InventoryItem
	character uint32
}

func (a *App) openAppearanceEditor(source [2]uint8) bool {
	item, ok := a.state.Inventory[source]
	if !ok || source[0] == 0 {
		return false
	}
	def := a.itemsItem(item.Type, item.TypeID)
	if def == nil || (def.Special != 44 && def.Special != 45) {
		return false
	}
	switch item.Type {
	case 25, 44, 100, 101, 102:
	default:
		return false
	}
	slot, ok := a.selfSlot()
	if !ok || item.Count == 0 {
		return false
	}
	a.appearanceEdit = &appearanceEdit{source: source, item: item, character: slot.ID}
	a.inventory.open = true
	a.appearanceUI.Open([4]uint8{slot.Hair, slot.Face, slot.Height, slot.Gender}, def.Special == 45)
	return true
}

func (a *App) updateAppearanceEditor(in *platform.Input, w, h int) {
	e := a.appearanceEdit
	if a.state == nil || a.state.Self.Dead || a.state.SelfID != e.character || a.state.Inventory[e.source] != e.item || !a.inventory.open {
		a.appearanceEdit = nil
		return
	}
	result := a.appearanceUI.Update(in, w, h)
	if result == ui.MessageNone {
		return
	}
	a.appearanceEdit = nil
	if result == ui.MessageCancel {
		return
	}
	d := a.appearanceUI.Values
	if err := a.world.Send(proto.ChangeAppearancePacket(e.source[0], e.source[1], d[0], d[1], d[2], d[3])); err != nil {
		a.backToLogin("Unable to request appearance change.")
		return
	}
	a.inventory.pending, a.inventory.deadline = true, a.clock+10
}
