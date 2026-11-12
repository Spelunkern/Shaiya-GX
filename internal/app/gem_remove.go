package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

type gemRemoveView struct {
	item                         [2]uint8
	pending, success, refreshing bool
	refresh                      gemRefresh
}

func (a *App) sendGemRemove(key [2]uint8) bool {
	if a.state == nil || a.world == nil || a.state.Self.Dead || a.shop.pending || a.inventory.pending ||
		a.state.Exchange.Active || a.state.Duel.Phase != game.DuelNone {
		return false
	}
	item, ok := a.state.Inventory[key]
	if !ok || item.Count != 1 || item.Gems == [6]uint8{} {
		return false
	}
	a.gemRemove = gemRemoveView{item: key, pending: true}
	a.shop.pending, a.shop.deadline = true, a.clock+10
	if err := a.world.Send(proto.GemRemovePacket(key[0], key[1])); err != nil {
		a.backToLogin("Unable to request lapis extraction.")
		return false
	}
	return true
}

func (a *App) confirmedGemRemove(p []byte) {
	v := &a.gemRemove
	if !v.pending {
		return
	}
	switch proto.Op(p) {
	case proto.GemRemove:
		result, err := proto.DecodeGemRemoveResult(p)
		if err != nil || v.refreshing || v.item != [2]uint8{result.ItemBag, result.ItemSlot} {
			return
		}
		// A failure can mean rejection, loss of all gems, or item destruction.
		// Query the source slot even on success to resolve partial extraction.
		v.success, v.refreshing = result.Result == 1, true
		v.refresh.start([][2]uint8{v.item})
		a.shop.deadline = a.clock + 10
		if err := a.world.Send(proto.ItemMovePacket(v.item[0], v.item[1], v.item[0], v.item[1])); err != nil {
			a.backToLogin("Unable to refresh equipment after extraction.")
		}
	case proto.ItemBagToBag:
		if v.refreshing && v.refresh.acknowledge(p) && len(v.refresh.pending) == 0 {
			v.pending, a.shop.pending = false, false
			a.blacksmith.selected = [4]bool{}
			if v.success {
				a.questMessage("Lapis extracted.")
			} else {
				a.questMessage("Lapis extraction failed or was refused.")
			}
		}
	}
}
