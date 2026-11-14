package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/itemtype"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

func (a *App) forgeDefinition(it proto.InventoryItem) *sdata.Item {
	if a.items == nil && a.arc == nil {
		return nil
	}
	return a.itemsItem(it.Type, it.TypeID)
}
func (a *App) isForgeHammer(it proto.InventoryItem, special uint8) bool {
	d := a.forgeDefinition(it)
	return d != nil && itemtype.Real(it.Type) == 26 && d.Special == special
}
func lapFitsItem(lap *sdata.Item, t uint8) bool {
	if lap == nil {
		return false
	}
	switch rt := itemtype.Real(t); {
	case rt == 24:
		return true
	case rt >= 1 && rt <= 15:
		return lap.MinLevel != 0
	case rt == 16:
		return lap.Country != 0
	case rt >= 17 && rt <= 22:
		return lap.Classes[rt-17] != 0
	case rt == 23:
		return lap.Grow != 0
	case rt == 25:
		return lap.RequiredStats[0] != 0
	}
	return false
}
func (a *App) gemPlacementFailure(item, lap proto.InventoryItem) int {
	target, g := a.forgeDefinition(item), a.forgeDefinition(lap)
	if target == nil || g == nil || item.Count != 1 || lap.Type != 30 || !lapFitsItem(g, item.Type) {
		return 62
	}
	used, free := 0, false
	for _, id := range item.Gems {
		if id == 0 {
			free = true
			continue
		}
		if id == lap.TypeID {
			return 63
		}
		if def := a.forgeDefinition(proto.InventoryItem{Type: 30, TypeID: id}); def != nil {
			used += int(def.Slot)
		}
	}
	if !free || int(g.Slot) > int(target.Slot)-used {
		return 64
	}
	return 0
}
func (a *App) placeBlacksmithItem(slot int, key [2]uint8, it proto.InventoryItem) bool {
	v := &a.blacksmith
	accepted := false
	mode := a.blacksmithUI.Mode
	switch slot {
	case 0:
		rt := itemtype.Real(it.Type)
		accepted = rt >= 1 && rt <= 25 && it.Count == 1
		if accepted && mode == ui.ForgeLapis && v.selected[1] {
			if id := a.gemPlacementFailure(it, v.items[1]); id != 0 {
				a.systemMessage(3, id, nil)
				return false
			}
		}
	case 1:
		if key[0] == 0 {
			return false
		}
		if mode == ui.ForgeLapis {
			if !v.selected[0] {
				a.systemMessage(3, 62, nil)
				return false
			}
			if id := a.gemPlacementFailure(v.items[0], it); id != 0 {
				a.systemMessage(3, id, nil)
				return false
			}
			accepted = true
		} else if mode == ui.ForgeEnchant {
			accepted = it.Type == 95 && v.selected[0] && lapFitsItem(a.forgeDefinition(it), v.items[0].Type)
			if !accepted {
				a.systemMessage(3, 62, nil)
				return false
			}
		}
	case 2:
		d := a.forgeDefinition(it)
		if mode == ui.ForgeRemake {
			accepted = key[0] != 0 && d != nil && d.Special == 62
		} else if mode == ui.ForgeEnchant {
			accepted = key[0] != 0 && it.Count > 0
			if v.quote.known && v.quote.value.Material != 0 {
				accepted = accepted && uint32(it.Type)*1000+uint32(it.TypeID) == v.quote.value.Material
			}
		}
	case 3:
		accepted = key[0] != 0 && (a.isForgeHammer(it, 36) || a.isForgeHammer(it, 58))
	}
	if !accepted {
		return false
	}
	v.keys[slot], v.items[slot], v.selected[slot] = key, it, true
	v.quote = forgeQuoteView{}
	return true
}
func (a *App) blacksmithExtractionSlot() int {
	v := &a.blacksmith
	if !v.selected[0] {
		return -1
	}
	it := a.state.Inventory[v.keys[0]]
	if v.extractSlot >= 0 && v.extractSlot < 6 && it.Gems[v.extractSlot] != 0 {
		return v.extractSlot
	}
	for i, g := range it.Gems {
		if g != 0 {
			return i
		}
	}
	return -1
}
func (a *App) nearBlacksmith() bool {
	if !a.blacksmithOpen() || a.wv == nil {
		return false
	}
	e := a.state.Entities[a.shop.npcID]
	dx, dz := e.X-a.wv.pos[0], e.Z-a.wv.pos[2]
	if dx*dx+dz*dz >= 100 {
		a.questMessage("Move closer to the blacksmith.")
		return false
	}
	return true
}
