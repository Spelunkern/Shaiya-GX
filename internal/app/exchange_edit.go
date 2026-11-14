package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

type exchangeQuantity struct {
	partner uint32
	source  [2]uint8
	item    proto.InventoryItem
	slot    uint8
}

func (a *App) confirmedExchange(p []byte) {
	if a.state == nil {
		return
	}
	x, v := &a.state.Exchange, &a.exchange
	switch proto.Op(p) {
	case proto.ExchangeReady:
		if update, err := proto.DecodeExchangeUpdate(p); err == nil && update.Kind == 3 && update.Value != 0 {
			v.pending = false
		}
	case proto.ExchangeStart:
		if x.Active {
			a.inventory.open = true
			a.inventory.bag = max(a.inventory.bag, 1)
			a.skills.open = false
			a.status.open = false
		}
	case proto.ExchangeEnd:
		v.quantity = nil
		v.moneyOpen = false
		v.mutation = 0
		v.cancelSent = false
	case proto.ExchangeAdd:
		if offer, err := proto.DecodeExchangeOwnItem(p); err == nil && v.mutation == 1 && offer.Destination == v.slot {
			v.mutation = 0
		}
	case proto.ExchangeAddDest:
		if slot, _, err := proto.DecodeExchangePartnerItem(p); err == nil && v.mutation == 2 && slot == v.slot && !x.PendingRemove[slot] {
			v.mutation = 0
		}
	case proto.ExchangeAddMoney:
		if money, err := proto.DecodeExchangeMoney(p); err == nil && v.mutation == 3 && money.Kind == 1 {
			v.mutation = 0
		}
	}
	if x.Desync && x.Active && !v.cancelSent {
		v.cancelSent = true
		v.mutation = 0
		a.questMessage("Trade cancelled: simultaneous offer changes could not be distinguished.")
		if err := a.world.Send(proto.ExchangeReadyPacket(2)); err != nil {
			a.backToLogin("Unable to cancel ambiguous trade.")
		}
	}
}
func (a *App) exchangeCanEdit() bool {
	x, v := &a.state.Exchange, &a.exchange
	if !x.Active || a.state.Duel.Phase != game.DuelNone || a.state.Self.Dead || v.mutation != 0 || v.pending || v.cancelSent || a.inventory.pending || a.shop.pending || a.shop.quantity != nil {
		return false
	}
	if x.OwnReady || x.OwnConfirmed {
		a.questMessage("Undo your trade confirmation before editing the offer.")
		return false
	}
	return true
}
func (a *App) sendExchangeOffer(source [2]uint8, item proto.InventoryItem, slot uint8, count uint16) {
	if !a.exchangeCanEdit() || a.state.Inventory[source] != item || count == 0 || count > item.Count {
		return
	}
	if err := a.world.Send(proto.ExchangeAddPacket(source[0], source[1], count, slot)); err != nil {
		a.backToLogin("Unable to offer item.")
		return
	}
	a.exchange.mutation, a.exchange.slot, a.exchange.deadline = 1, slot, a.clock+10
}
func (a *App) updateExchangeDrag(in *platform.Input, w, h int) bool {
	if a.exchangeUI == nil || a.state == nil || !a.state.Exchange.Active || !a.inventory.dragging || !in.Clicked {
		return false
	}
	// These panels are drawn above trade and retain ownership when overlapping.
	if a.wv != nil && a.wv.mapView.full {
		return false
	}
	if a.questLog.open && a.questUI != nil {
		x, y := a.questUI.Origin(w, h)
		if pointInside(in, x, y, ui.QuestWidth, ui.QuestHeight) {
			return false
		}
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
	x, y := a.exchangeUI.Origin(w, h)
	slot, hit := ui.ExchangeHit(in.MouseX-x, in.MouseY-y)
	if !hit {
		return false
	}
	source := a.inventory.source
	item, ok := a.state.Inventory[source]
	a.inventory.dragging, a.inventory.selected = false, false
	if !ok || source[0] == 0 || !a.exchangeCanEdit() {
		return true
	}
	if item.Type == 27 || item.Type == 28 || item.Type == 29 {
		a.localizedNotice(1, 231, nil, "Quest items cannot be traded.")
		return true
	}
	if a.state.Exchange.Own[slot].Item.Type != 0 {
		a.questMessage("Choose an empty trade slot.")
		return true
	}
	for _, offer := range a.state.Exchange.Own {
		if offer.Item.Type != 0 && offer.Source == source {
			a.questMessage("That item is already offered.")
			return true
		}
	}
	if item.Count > 1 {
		a.exchange.quantity = &exchangeQuantity{partner: a.state.Exchange.PartnerID, source: source, item: item, slot: uint8(slot)}
		a.quantityUI.Open(in.MouseX, in.MouseY, w, h, int(item.Count))
	} else {
		a.sendExchangeOffer(source, item, uint8(slot), 1)
	}
	return true
}
func (a *App) updateExchangeModal(in *platform.Input) {
	v := &a.exchange
	if a.state == nil || !a.state.Exchange.Active {
		v.quantity = nil
		v.moneyOpen = false
		return
	}
	if v.quantity != nil {
		q := v.quantity
		if a.state.Exchange.PartnerID != q.partner || a.state.Inventory[q.source] != q.item {
			v.quantity = nil
			return
		}
		count := a.quantityUI.Update(in)
		if count == 0 {
			return
		}
		v.quantity = nil
		if count > 0 {
			a.sendExchangeOffer(q.source, q.item, q.slot, uint16(count))
		}
		return
	}
	if v.moneyOpen && a.exchangeMoney != nil {
		money, done, cancel := a.exchangeMoney.Update(in)
		if !done {
			return
		}
		v.moneyOpen = false
		if cancel || !a.exchangeCanEdit() {
			return
		}
		amount := uint32(min(money, uint64(a.state.Details.Money)))
		if err := a.world.Send(proto.ExchangeMoneyPacket(amount)); err != nil {
			a.backToLogin("Unable to offer gold.")
			return
		}
		v.mutation, v.deadline = 3, a.clock+10
	}
}
