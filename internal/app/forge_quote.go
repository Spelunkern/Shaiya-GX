package app

import (
	"fmt"
	"github.com/Spelunkern/Shaiya-GX/internal/data/text"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
	"strconv"
	"strings"
)

type forgeQuoteView struct {
	request        proto.ForgeQuoteRequest
	items          [4]proto.InventoryItem
	value          proto.ForgeQuote
	known, pending bool
	next           float64
	confirm        uint8
}

func (a *App) forgeRequest(kind uint8) proto.ForgeQuoteRequest {
	v := &a.blacksmith
	q := proto.ForgeQuoteRequest{Kind: kind, Count: 1}
	if kind == proto.ForgeQuoteExtract {
		q.Count = uint8(max(a.blacksmithExtractionSlot(), 0))
	}
	for slot, k := range []*[2]uint8{&q.Item, &q.Lapis, &q.Material, &q.Hammer} {
		*k = [2]uint8{255, 255}
		if v.selected[slot] {
			*k = v.keys[slot]
		}
	}
	return q
}
func (a *App) requestForgeQuote(kind, confirm uint8, force bool) {
	v := &a.blacksmith
	q := a.forgeRequest(kind)
	old := v.quote.request
	old.Serial = 0
	if !force && q == old && v.items == v.quote.items && (v.quote.pending && a.clock < v.quote.next || v.quote.known && a.clock < v.quote.next) {
		return
	}
	a.forgeQuoteSerial++
	q.Serial = a.forgeQuoteSerial
	v.quote = forgeQuoteView{request: q, items: v.items, pending: true, next: a.clock + 2, confirm: confirm}
	if a.world == nil {
		return
	}
	if err := a.world.Send(proto.ForgeQuotePacket(q)); err != nil {
		a.backToLogin("Unable to request blacksmith details.")
	}
}
func (a *App) updateForgeQuote() {
	v := &a.blacksmith
	if v.quote.pending && v.quote.confirm != 0 && a.clock < v.quote.next {
		return
	}
	if !v.selected[0] || a.blacksmithUI.Mode != ui.ForgeEnchant {
		v.quote = forgeQuoteView{}
		return
	}
	kind := proto.ForgeQuoteLink
	if a.blacksmithUI.Mode == ui.ForgeEnchant {
		kind = proto.ForgeQuoteEnchant
	} else if !v.selected[1] {
		if v.items[0].Gems == [6]uint8{} {
			return
		}
		kind = proto.ForgeQuoteExtract
	}
	a.requestForgeQuote(kind, 0, false)
}
func (a *App) confirmedForgeQuote(p []byte) {
	q, err := proto.DecodeForgeQuote(p)
	v := &a.blacksmith
	expected := v.quote.request
	expected.Serial = 0
	if err != nil || !a.blacksmithOpen() || !v.quote.pending || q.Serial != v.quote.request.Serial || q.Kind != v.quote.request.Kind || a.forgeRequest(q.Kind) != expected || v.items != v.quote.items {
		return
	}
	for slot, selected := range v.selected {
		if selected && a.state.Inventory[v.keys[slot]] != v.quote.items[slot] {
			return
		}
	}

	confirm := v.quote.confirm
	v.quote.pending = false
	v.quote.known = true
	v.quote.value = q
	if confirm == 0 || a.gemConfirm == nil {
		return
	}
	if !q.Valid {
		a.questMessage("The selected items cannot be used for this action.")
		return
	}
	v.confirm = confirm
	a.gemConfirm.OpenFull(a.forgeConfirmation(q), true, a.font)
}
func forgePercent(rate uint32) string { return strconv.FormatFloat(float64(rate)/10000, 'f', -1, 64) }
func (a *App) forgeConfirmation(q proto.ForgeQuote) string {
	id := 10011
	if q.Kind == proto.ForgeQuoteExtract {
		id = 840
	}
	if q.Kind != proto.ForgeQuoteEnchant && a.msg.Get(id) != "" {
		template := a.msg.Get(id)
		// Native 10011 starts with an obsolete tendency-dependent cost explanation.
		// Keep its original cost/rate/question wording, omitting only that intro.
		if q.Kind == proto.ForgeQuoteLink {
			if nl := strings.IndexByte(template, '\n'); nl >= 0 && !strings.Contains(template[:nl], "<b>") && !strings.Contains(template[:nl], "<v>") {
				template = template[nl+1:]
			}
		}
		value := text.SysMsg{id: template}.Format(id, map[string]string{"b": strconv.FormatUint(uint64(q.Gold), 10), "v": forgePercent(q.Rate)})
		for strings.Contains(value, "%%") {
			value = strings.ReplaceAll(value, "%%", "%")
		}
		return value
	}
	action := "Link lapis"
	if q.Kind == proto.ForgeQuoteExtract {
		action = "Extract lapis"
	}
	if q.Kind == proto.ForgeQuoteEnchant {
		action = "Enchant equipment"
	}
	return fmt.Sprintf("%s?\nSuccess: %s%%\nGold: %d", action, forgePercent(q.Rate), q.Gold)
}
func (a *App) syncBlacksmithItems() {
	v := &a.blacksmith
	for i, selected := range v.selected {
		if !selected {
			continue
		}
		it, ok := a.state.Inventory[v.keys[i]]
		if !ok || it.Type != v.items[i].Type || it.TypeID != v.items[i].TypeID || it.Count == 0 {
			v.selected[i] = false
			continue
		}
		v.items[i] = it
	}
}
