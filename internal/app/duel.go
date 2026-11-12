package app

import (
	"fmt"
	"math"
	"strings"

	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

type duelView struct {
	requestSerial, resultSerial uint64
	inviteUntil, countdownUntil float64
	quantity                    *exchangeQuantity
	moneyOpen, cancelSent       bool
	mutation, slot              uint8
	deadline                    float64
}

func (a *App) confirmedDuel(p []byte) {
	if a.state == nil {
		return
	}
	d, v := &a.state.Duel, &a.duel
	switch proto.Op(p) {
	case proto.DuelWindow:
		if d.Phase == game.DuelWager {
			a.inventory.open = true
			a.inventory.bag = max(a.inventory.bag, 1)
			a.skills.open = false
			a.status.open = false
			if a.duelMessage != nil {
				a.duelMessage.Visible = false
			}
		}
	case proto.DuelCountdown:
		v.countdownUntil = a.clock + 5
		v.quantity = nil
		v.moneyOpen = false
		v.mutation = 0
	case proto.DuelStart:
		if d.Phase == game.DuelFighting {
			if e := a.state.Entities[d.Opponent]; e != nil && e.Kind == game.KindUser {
				a.state.Target = d.Opponent
			}
			a.questMessage("Duel started.")
		}
	case proto.DuelResult:
		if d.Outcome == 1 {
			a.questMessage("Duel won.")
		} else if d.Outcome == 2 {
			a.questMessage("Duel lost.")
		}
	case proto.DuelEnd:
		a.questMessage("Duel ended.")
	case proto.DuelWindowEnd:
		v.quantity = nil
		v.moneyOpen = false
		v.mutation = 0
		v.cancelSent = false
	case proto.DuelReady:
		if u, err := proto.DecodeExchangeUpdate(p); err == nil && (u.Kind == 1 || u.Kind == 3 && u.Value == 1) && v.mutation == 4 {
			v.mutation = 0
		}
	case proto.DuelAdd:
		if item, err := proto.DecodeExchangeOwnItem(p); err == nil && v.mutation == 1 && item.Destination == v.slot {
			v.mutation = 0
		}
	case proto.DuelRemove:
		if u, err := proto.DecodeExchangeUpdate(p); err == nil && u.Kind == 1 && v.mutation == 2 && u.Value == v.slot {
			v.mutation = 0
		}
	case proto.DuelMoney:
		if u, err := proto.DecodeExchangeMoney(p); err == nil && u.Kind == 1 && v.mutation == 3 {
			v.mutation = 0
		}
	}
}
func (a *App) duelCommand(text string) bool {
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return false
	}
	command := strings.ToLower(fields[0])
	if command != "/duel" && command != "/forfeit" {
		return false
	}
	if command == "/forfeit" {
		if a.state.Duel.Phase == game.DuelFighting {
			if err := a.world.Send(proto.DuelGiveUpPacket()); err != nil {
				a.backToLogin("Unable to surrender duel.")
			}
		}
		return true
	}
	e := a.state.Entities[a.state.Target]
	if len(fields) > 1 {
		name := strings.Join(fields[1:], " ")
		e = nil
		for _, p := range a.state.Entities {
			if p.Kind == game.KindUser && p.Shape != nil && strings.EqualFold(p.Shape.Name, name) {
				e = p
				break
			}
		}
	}
	if e == nil || e.Kind != game.KindUser || e.ID == a.state.SelfID || a.state.Duel.Phase != game.DuelNone || a.state.GuildDuel.Phase != game.DuelNone || a.state.Exchange.Active {
		a.questMessage("Select an available player to duel.")
		return true
	}
	if err := a.world.Send(proto.DuelRequestPacket(e.ID)); err != nil {
		a.backToLogin("Unable to request duel.")
	}
	return true
}
func (a *App) updateDuelInvite(in *platform.Input, w, h int) bool {
	if a.duelMessage == nil || a.state == nil {
		return false
	}
	d, v := &a.state.Duel, &a.duel
	if d.RequestSerial != v.requestSerial {
		v.requestSerial = d.RequestSerial
		if d.Phase == game.DuelIncoming {
			name := fmt.Sprintf("Player %d", d.Opponent)
			if e := a.state.Entities[d.Opponent]; e != nil && e.Shape != nil && e.Shape.Name != "" {
				name = e.Shape.Name
			}
			a.duelMessage.Open(name+" challenges you to a duel. Accept?", true)
			v.inviteUntil = a.clock + 10
		}
	}
	if d.ResultSerial != v.resultSerial {
		v.resultSerial = d.ResultSerial
		if d.Phase == game.DuelNone {
			a.duelMessage.Visible = false
		}
	}
	if !a.duelMessage.Visible {
		return false
	}
	if d.Phase != game.DuelIncoming || a.clock >= v.inviteUntil {
		a.duelMessage.Visible = false
		return false
	}
	result := a.duelMessage.Update(in, w, h)
	if result != ui.MessageNone {
		accept := result == ui.MessageOK
		if accept && (a.state.Exchange.Active || a.state.GuildDuel.Phase != game.DuelNone) {
			accept = false
			a.questMessage("Finish the current trade or guild duel first.")
		}
		if err := a.world.Send(proto.DuelAnswerPacket(accept)); err != nil {
			a.backToLogin("Unable to answer duel.")
			return true
		}
		if !accept {
			d.Phase = game.DuelNone
		}
	}
	return true
}
func (a *App) duelCanEdit() bool {
	if a.state == nil {
		return false
	}
	d, v := &a.state.Duel, &a.duel
	if d.Phase != game.DuelWager || a.state.Exchange.Active || a.state.Self.Dead || v.mutation != 0 || v.cancelSent || a.inventory.pending || a.shop.pending || a.shop.quantity != nil {
		return false
	}
	if d.OwnReady {
		a.questMessage("Undo your ready status before editing the wager.")
		return false
	}
	return true
}
func (a *App) sendDuelOffer(source [2]uint8, item proto.InventoryItem, slot, count uint8) {
	if !a.duelCanEdit() || a.state.Inventory[source] != item || count == 0 || count > item.Count {
		return
	}
	if err := a.world.Send(proto.DuelAddPacket(source[0], source[1], count, slot)); err != nil {
		a.backToLogin("Unable to offer duel item.")
		return
	}
	a.duel.mutation, a.duel.slot, a.duel.deadline = 1, slot, a.clock+10
}
func (a *App) updateDuelDrag(in *platform.Input, w, h int) bool {
	if a.duelUI == nil || a.state == nil || a.state.Duel.Phase != game.DuelWager || !a.inventory.dragging || !in.Clicked {
		return false
	}
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
	x, y := a.duelUI.Origin(w, h)
	slot, hit := ui.ExchangeHit(in.MouseX-x, in.MouseY-y)
	if !hit {
		return false
	}
	source := a.inventory.source
	item, ok := a.state.Inventory[source]
	a.inventory.dragging, a.inventory.selected = false, false
	if !ok || source[0] == 0 || !a.duelCanEdit() {
		return true
	}
	if item.Type == 27 || item.Type == 28 || item.Type == 29 {
		a.questMessage("Quest items cannot be wagered.")
		return true
	}
	if a.state.Duel.Own[slot].Item.Type != 0 {
		a.questMessage("Choose an empty wager slot.")
		return true
	}
	for _, offer := range a.state.Duel.Own {
		if offer.Item.Type != 0 && offer.Source == source {
			a.questMessage("That item is already wagered.")
			return true
		}
	}
	if item.Count > 1 {
		a.duel.quantity = &exchangeQuantity{partner: a.state.Duel.Opponent, source: source, item: item, slot: uint8(slot)}
		a.quantityUI.Open(in.MouseX, in.MouseY, w, h, int(item.Count))
	} else {
		a.sendDuelOffer(source, item, uint8(slot), 1)
	}
	return true
}
func (a *App) updateDuelModal(in *platform.Input) {
	v := &a.duel
	if a.state == nil || a.state.Duel.Phase != game.DuelWager {
		v.quantity = nil
		v.moneyOpen = false
		return
	}
	if v.quantity != nil {
		q := v.quantity
		if a.state.Duel.Opponent != q.partner || a.state.Inventory[q.source] != q.item {
			v.quantity = nil
			return
		}
		n := a.quantityUI.Update(in)
		if n == 0 {
			return
		}
		v.quantity = nil
		if n > 0 {
			a.sendDuelOffer(q.source, q.item, q.slot, uint8(n))
		}
		return
	}
	if v.moneyOpen && a.exchangeMoney != nil {
		value, done, cancel := a.exchangeMoney.Update(in)
		if !done {
			return
		}
		v.moneyOpen = false
		if cancel || !a.duelCanEdit() {
			return
		}
		money := uint32(min(value, uint64(a.state.Details.Money)))
		if err := a.world.Send(proto.DuelMoneyPacket(money)); err != nil {
			a.backToLogin("Unable to wager gold.")
			return
		}
		v.mutation, v.deadline = 3, a.clock+10
	}
}
func (a *App) updateDuel(in *platform.Input, w, h int) bool {
	if a.duelUI == nil || a.state == nil || a.state.Duel.Phase != game.DuelWager {
		return false
	}
	d, v := &a.state.Duel, &a.duel
	confirm, close := a.duelUI.Update(in, w, h)
	if close || in.HasKey(platform.KeyEscape) {
		if !v.cancelSent {
			if err := a.world.Send(proto.DuelReadyPacket(2)); err != nil {
				a.backToLogin("Unable to cancel duel.")
				return true
			}
			v.cancelSent = true
		}
		return true
	}
	x, y := a.duelUI.Origin(w, h)
	if in.RightPressed {
		if slot, hit := ui.ExchangeHit(in.MouseX-x, in.MouseY-y); hit && d.Own[slot].Item.Type != 0 {
			if !a.duelCanEdit() {
				return true
			}
			if err := a.world.Send(proto.DuelRemovePacket(uint8(slot))); err != nil {
				a.backToLogin("Unable to remove wager.")
				return true
			}
			v.mutation, v.slot, v.deadline = 2, uint8(slot), a.clock+10
			return true
		}
	}
	if in.Clicked && in.MouseX >= x+87 && in.MouseX < x+182 && in.MouseY >= y+175 && in.MouseY < y+197 && a.exchangeMoney != nil {
		if a.duelCanEdit() {
			v.moneyOpen = true
			a.exchangeMoney.Open(w, h, d.OwnMoney)
		}
		return true
	}
	if confirm && v.mutation == 0 && !v.cancelSent {
		ready := uint8(0)
		if d.OwnReady {
			ready = 1
		}
		if err := a.world.Send(proto.DuelReadyPacket(ready)); err != nil {
			a.backToLogin("Unable to update duel readiness.")
			return true
		}
		v.mutation, v.deadline = 4, a.clock+10
		return true
	}
	return panelPointer(in, x, y, ui.ExchangeWidth, ui.ExchangeHeight)
}
func (a *App) drawDuel(w, h int) {
	if a.duelUI == nil || a.state == nil {
		return
	}
	d := &a.state.Duel
	if d.Phase == game.DuelCounting {
		seconds := max(0, int(math.Ceil(a.duel.countdownUntil-a.clock)))
		a.font.Draw(a.sprites, fmt.Sprintf("Duel begins in %d", seconds), w/2-60, 100, white, 160)
	}
	if d.Phase != game.DuelWager {
		return
	}
	name := fmt.Sprintf("Player %d", d.Opponent)
	if e := a.state.Entities[d.Opponent]; e != nil && e.Shape != nil && e.Shape.Name != "" {
		name = e.Shape.Name
	}
	a.duelUI.Draw(a.sprites, a.font, w, h, name, d.OwnMoney, d.OtherMoney, d.OwnReady, d.OtherReady)
	x, y := a.duelUI.Origin(w, h)
	for side := 0; side < 2; side++ {
		for i := 0; i < proto.ExchangeSlots; i++ {
			it := d.Other[i]
			if side == 0 {
				it = d.Own[i].Item
			}
			if it.Type == 0 || it.Count == 0 {
				continue
			}
			cx, cy, _ := ui.ExchangeCell(side, i)
			if def := a.itemsItem(it.Type, it.TypeID); def != nil {
				a.drawItemIcon(def, x+cx, y+cy, 32, white)
			}
			if it.Count > 1 {
				text := fmt.Sprint(it.Count)
				a.font.Draw(a.sprites, text, x+cx+32-a.font.Width(text), y+cy+18, white, 32)
			}
		}
	}
}
