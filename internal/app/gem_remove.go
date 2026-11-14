package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

type gemRemoveView struct {
	item                         [2]uint8
	itemName                     string
	hammer                       [2]uint8
	withHammer                   bool
	pending, success, refreshing bool
	refresh                      gemRefresh
}

func (a *App) sendGemRemove(key [2]uint8, tool ...[2]uint8) bool {
	if a.state == nil || a.world == nil || a.state.Self.Dead || a.shop.pending || a.inventory.pending ||
		a.state.Exchange.Active || a.state.Duel.Phase != game.DuelNone {
		return false
	}
	item, ok := a.state.Inventory[key]
	if !ok || item.Count != 1 || item.Gems == [6]uint8{} {
		return false
	}
	if len(tool) != 1 {
		return false
	}
	hammer, ok := a.state.Inventory[tool[0]]
	if !ok || hammer.Count == 0 || tool[0][0] == 0 || !a.isForgeHammer(hammer, 58) {
		return false
	}
	socket := a.blacksmithExtractionSlot()
	if socket < 0 {
		return false
	}
	a.gemRemove = gemRemoveView{item: key, pending: true, hammer: tool[0], withHammer: true, itemName: a.gemItemName(item.Type, item.TypeID)}
	packet := proto.GemRemoveHammerPacket(key, tool[0], uint8(socket))
	a.shop.pending, a.shop.deadline = true, a.clock+10
	if err := a.world.Send(packet); err != nil {
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
		// A failure can mean rejection or loss of the selected lapis.
		// Query the source slot even on success to resolve partial extraction.
		v.success, v.refreshing = result.Result == 1, true
		keys := [][2]uint8{v.item}
		if v.withHammer {
			keys = append(keys, v.hammer)
		}
		v.refresh.start(keys)
		a.shop.deadline = a.clock + 10
		for _, key := range keys {
			if err := a.world.Send(proto.ItemMovePacket(key[0], key[1], key[0], key[1])); err != nil {
				a.backToLogin("Unable to refresh extraction items.")
				return
			}
		}
	case proto.ItemBagToBag:
		if v.refreshing && v.refresh.acknowledge(p) && len(v.refresh.pending) == 0 {
			v.pending, a.shop.pending = false, false
			a.syncBlacksmithItems()
			a.forgeResult(v.success)
			id := 58
			if v.success {
				id = 57
			}
			a.systemMessage(3, id, map[string]string{"i": v.itemName})
		}
	}
}
