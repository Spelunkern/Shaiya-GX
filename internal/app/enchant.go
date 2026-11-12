package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

type enchantView struct {
	lap, material, item          [2]uint8
	pending, success, refreshing bool
	refresh                      gemRefresh
}

func (a *App) sendEnchant(lapKey, materialKey, itemKey [2]uint8, count uint8) bool {
	if a.state == nil || a.world == nil || a.state.Self.Dead || a.shop.pending || a.inventory.pending || a.state.Exchange.Active ||
		a.state.Duel.Phase != game.DuelNone || lapKey == materialKey || lapKey == itemKey || materialKey == itemKey {
		return false
	}
	lap, l := a.state.Inventory[lapKey]
	material, m := a.state.Inventory[materialKey]
	item, i := a.state.Inventory[itemKey]
	if !l || !m || !i || lap.Type != 95 || lap.Count == 0 || item.Count != 1 || count == 0 || count > 100 || count > material.Count {
		return false
	}
	a.enchant = enchantView{lap: lapKey, material: materialKey, item: itemKey, pending: true}
	a.shop.pending, a.shop.deadline = true, a.clock+10
	if err := a.world.Send(proto.EnchantPacket(lapKey[0], lapKey[1], materialKey[0], materialKey[1], count, itemKey[0], itemKey[1])); err != nil {
		a.backToLogin("Unable to request enchantment.")
		return false
	}
	return true
}

func (a *App) confirmedEnchant(p []byte) {
	v := &a.enchant
	if !v.pending {
		return
	}
	switch proto.Op(p) {
	case proto.EnchantAdd:
		r, err := proto.DecodeEnchantResult(p)
		if err != nil || v.refreshing || v.lap != [2]uint8{r.LapisiaBag, r.LapisiaSlot} || v.material != [2]uint8{r.MaterialBag, r.MaterialSlot} || v.item != [2]uint8{r.ItemBag, r.ItemSlot} {
			return
		}
		v.success, v.refreshing = r.Result == 1, true
		v.refresh.start([][2]uint8{v.item})
		a.shop.deadline = a.clock + 10
		if err := a.world.Send(proto.ItemMovePacket(v.item[0], v.item[1], v.item[0], v.item[1])); err != nil {
			a.backToLogin("Unable to refresh enchanted equipment.")
		}
	case proto.ItemBagToBag:
		if v.refreshing && v.refresh.acknowledge(p) && len(v.refresh.pending) == 0 {
			v.pending, a.shop.pending = false, false
			a.blacksmith.selected = [4]bool{}
			if v.success {
				a.questMessage("Equipment enchanted.")
			} else {
				a.questMessage("Enchantment failed or was refused.")
			}
		}
	}
}

func (a *App) updateEnchantQuantity(in *platform.Input) {
	v := &a.blacksmith
	if !a.blacksmithOpen() || !a.blacksmithEnchantReady() || a.shop.pending || a.inventory.pending {
		v.quantity = false
		return
	}
	count := a.quantityUI.Update(in)
	if count == 0 {
		return
	}
	v.quantity = false
	if count < 0 {
		return
	}
	v.materialCount, v.confirm = uint8(count), 5
	if a.gemConfirm != nil {
		a.gemConfirm.Open("Enchanting consumes materials and gold. Failure can break the equipment. Continue?", true)
	}
}
