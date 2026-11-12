package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

type repairView struct {
	request           [2]uint8
	pending, accepted bool
	refresh           gemRefresh
}

func (a *App) sendRepair(key [2]uint8) bool {
	if a.state == nil || a.world == nil || a.state.Self.Dead || a.shop.pending || a.inventory.pending ||
		a.state.Exchange.Active || a.state.Duel.Phase != game.DuelNone {
		return false
	}
	all := key == [2]uint8{255, 255}
	if !all {
		if _, ok := a.state.Inventory[key]; !ok {
			return false
		}
	}
	slots := [][2]uint8{key}
	if all {
		slots = nil
		for slot := uint8(0); slot < 13; slot++ {
			if _, ok := a.state.Inventory[[2]uint8{0, slot}]; ok {
				slots = append(slots, [2]uint8{0, slot})
			}
		}
		if len(slots) == 0 {
			slots = append(slots, [2]uint8{1, 0})
		}
	}
	a.repair = repairView{request: key, pending: true}
	a.repair.refresh.start(slots)
	a.shop.pending, a.shop.deadline = true, a.clock+10
	if err := a.world.Send(proto.ItemRepairPacket(key[0], key[1])); err != nil {
		a.backToLogin("Unable to request repair.")
		return false
	}
	// Rapture sends no rejection. A same-slot move after the request is an
	// ordered barrier and returns the actual durability whether repair worked
	// or not; any successful 0x0554 precedes these echoes on the connection.
	for _, slot := range slots {
		if err := a.world.Send(proto.ItemMovePacket(slot[0], slot[1], slot[0], slot[1])); err != nil {
			a.backToLogin("Unable to refresh repaired equipment.")
			return false
		}
	}
	return true
}

func (a *App) confirmedRepair(p []byte) {
	v := &a.repair
	if !v.pending {
		return
	}
	switch proto.Op(p) {
	case proto.ItemRepair:
		result, err := proto.DecodeItemRepair(p)
		if err == nil && v.request == [2]uint8{result.Bag, result.Slot} {
			v.accepted = true
		}
	case proto.ItemBagToBag:
		if v.refresh.acknowledge(p) && len(v.refresh.pending) == 0 {
			v.pending, a.shop.pending = false, false
			a.blacksmith.selected = [4]bool{}
			if v.accepted {
				a.questMessage("Equipment repaired.")
			} else {
				a.questMessage("Repair was not performed.")
			}
		}
	}
}
