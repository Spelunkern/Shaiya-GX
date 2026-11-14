package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

type gemLinkView struct {
	gem, item                    [2]uint8
	itemName, gemName            string
	pending, success, refreshing bool
	hammer                       [2]uint8
	withHammer                   bool
	refresh                      gemRefresh
}

// sendGemLink reserves the inventory until the result and any necessary
// hammer refresh have been acknowledged. No item is changed locally.
func (a *App) sendGemLink(gemKey, itemKey [2]uint8, hammer ...[2]uint8) bool {
	if a.state == nil || a.world == nil || a.state.Self.Dead || a.shop.pending || a.inventory.pending ||
		a.state.Exchange.Active || a.state.Duel.Phase != game.DuelNone || gemKey == itemKey {
		return false
	}
	gem, gemOK := a.state.Inventory[gemKey]
	item, itemOK := a.state.Inventory[itemKey]
	if !gemOK || !itemOK || gem.Type != 30 || gem.Count == 0 || item.Count != 1 {
		return false
	}
	var hammerKey [2]uint8
	if len(hammer) > 1 {
		return false
	}
	if len(hammer) == 1 {
		hammerKey = hammer[0]
		h, ok := a.state.Inventory[hammerKey]
		if !ok || hammerKey[0] == 0 || hammerKey == gemKey || hammerKey == itemKey || h.Count == 0 || !a.isForgeHammer(h, 36) {
			return false
		}
	}
	v := &a.gemLink
	*v = gemLinkView{gem: gemKey, item: itemKey, pending: true, hammer: hammerKey, withHammer: len(hammer) == 1}
	if d := a.gemItemName(item.Type, item.TypeID); d != "" {
		v.itemName = d
	}
	if d := a.gemItemName(gem.Type, gem.TypeID); d != "" {
		v.gemName = d
	}
	a.shop.pending, a.shop.deadline = true, a.clock+10
	packet := proto.GemAddPacket(gemKey[0], gemKey[1], itemKey[0], itemKey[1])
	if len(hammer) == 1 {
		packet = proto.GemAddHammerPacket(gemKey[0], gemKey[1], itemKey[0], itemKey[1], hammerKey[0], hammerKey[1])
	}
	if err := a.world.Send(packet); err != nil {
		a.backToLogin("Unable to request gem link.")
		return false
	}
	return true
}

func (a *App) confirmedGemLink(p []byte) {
	v := &a.gemLink
	if !v.pending {
		return
	}
	switch proto.Op(p) {
	case proto.GemAdd:
		result, err := proto.DecodeGemAddResult(p)
		if err != nil || v.gem != [2]uint8{result.GemBag, result.GemSlot} || v.item != [2]uint8{result.ItemBag, result.ItemSlot} {
			return
		}
		if v.refreshing {
			return
		}
		var slots [][2]uint8
		if v.withHammer {
			slots = append(slots, v.hammer)
		}
		v.success, v.refreshing = result.Result == 1, true
		v.refresh.start(slots)
		for key := range v.refresh.pending {
			if err := a.world.Send(proto.ItemMovePacket(key[0], key[1], key[0], key[1])); err != nil {
				a.backToLogin("Unable to refresh items after gem link.")
				return
			}
		}
		if len(v.refresh.pending) != 0 {
			a.shop.deadline = a.clock + 10
			return
		}
		a.finishGemLink(result.Result == 1)
	case proto.ItemBagToBag:
		if v.refreshing && v.refresh.acknowledge(p) && len(v.refresh.pending) == 0 {
			a.finishGemLink(v.success)
		}
	}
}

func (a *App) finishGemLink(success bool) {
	a.forgeResult(success)
	a.gemLink.pending = false
	a.syncBlacksmithItems()
	a.shop.pending = false
	id := 56
	if success {
		id = 55
	}
	a.systemMessage(3, id, map[string]string{"i": a.gemLink.itemName, "r": a.gemLink.gemName})
}

func (a *App) gemItemName(typ, id uint8) string {
	if a.items == nil && a.arc == nil {
		return ""
	}
	if d := a.itemsItem(typ, id); d != nil {
		return d.Name
	}
	return ""
}
