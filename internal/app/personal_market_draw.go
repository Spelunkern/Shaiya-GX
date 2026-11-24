package app

import (
	"fmt"
	"strconv"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

func (a *App) drawPersonalMarket(w, h int) {
	if !a.personalMarketOpen() {
		return
	}
	if a.state.ShopView.Open {
		a.drawPersonalMarketBuyer(w, h)
		return
	}
	v, p := &a.personalMarket, a.personalMarketUI
	s, b := &a.state.MyShop, &a.state.ShopView
	title := v.title.Text
	if b.Open {
		if e := a.state.Entities[b.Seller]; e != nil {
			title = e.ShopTitle
		}
	} else if s.Phase == proto.MyShopSelling {
		title = a.state.Self.ShopTitle
	} else if v.editing {
		title += "|"
	} else if title == "" {
		title = a.tr(12257)
	}
	var items [proto.MyShopSlots]proto.InventoryItem
	var prices [proto.MyShopSlots]uint32
	for i := range items {
		if b.Open && b.Present[i] {
			items[i], prices[i] = b.Items[i].Item, b.Items[i].Price
		} else if !b.Open && s.Slots[i].Used {
			entry := s.Slots[i]
			items[i], prices[i] = a.state.Inventory[[2]uint8{entry.Bag, entry.Slot}], entry.Price
		}
	}
	x, y := p.Origin(w, h)
	a.clearTooltipUnderPanel(x, y, ui.PersonalMarketWidth, ui.PersonalMarketHeight)
	p.Draw(a.sprites, a.font, w, h, title, b.Open, s.Phase == proto.MyShopSelling, v.pending != 0)
	hover := ui.PersonalMarketCellAt(a.lastMouseX-x, a.lastMouseY-y)
	for i, it := range items {
		if it.Type == 0 {
			continue
		}
		def := a.itemsItem(it.Type, it.TypeID)
		if def == nil {
			continue
		}
		cx, cy := ui.PersonalMarketCell(i)
		a.drawItemIcon(def, x+cx, y+cy, 32, white, &it)
		if count := a.itemCount(it.Type, it.TypeID, it.Count); count > 1 {
			text := strconv.Itoa(int(count))
			a.font.Draw(a.sprites, text, x+cx+31-a.font.Width(text), y+cy+18, white, 32)
		}
		if hover == i && !v.modal() && !a.inventory.dragging {
			a.drawItemDetails(def, &it, it.Count, x+ui.PersonalMarketWidth+5, y+cy, w, h, fmt.Sprintf("%d gold per unit", prices[i]))
		}
	}
}

func (a *App) drawPersonalMarketModal(w int) {
	if !a.personalMarketOpen() {
		return
	}
	d := a.personalMarket.dialog
	switch d.op {
	case proto.MyShopBuy:
		a.setTooltip(tooltipContent{})
		q := a.quantityUI
		q.Draw(a.sprites, a.font)
	case proto.MyShopAdd, proto.MyShopPrice:
		a.setTooltip(tooltipContent{})
		m := a.exchangeMoney
		m.Draw(a.sprites, a.font)
	}
}

func (a *App) drawPersonalMarketSigns(w, h int) {
	if len(a.state.ShopSigns) == 0 || a.wv == nil || a.scene3d == nil {
		return
	}
	f := a.chatFont()
	vp := a.scene3d.View.Mul(a.scene3d.Proj)
	for id, title := range a.state.ShopSigns {
		pos, ok := a.effectPosition(id)
		if !ok || pos.Sub(a.wv.pos).Len() > 50 {
			continue
		}
		pos = pos.Add(render.V3{0, 1.5, 0})
		if actor := a.playerActor(id); actor != nil {
			if anchor, ok := actor.NameAnchor(); ok {
				pos = anchor
			}
		}
		x, y, _, ok := projectLabel(pos, vp, w, h)
		if !ok {
			continue
		}
		// 0x44c12a: the common bubble routine 0x41c5c0 with its own colours.
		textWidth := min(292, f.Width(title))
		width, height := textWidth+20, 30
		left, top := x-width/2, y-height-40
		if a.scene3d.OccludedLabel(pos, left, top, width, height, w, h) {
			continue
		}
		a.sprites.Draw(a.white, float32(left), float32(top), float32(width), float32(height), 0, 0, 1, 1, [4]float32{29.0 / 255, 31.0 / 255, 36.0 / 255, 154.0 / 255}) // 0x44c186: 9a1d1f24
		f.Draw(a.sprites, title, left+10, top+(height-f.Height)/2, ui.RGBColor(0xccff66), textWidth)                                                                   // 0x44c181: ffccff66
	}
}

func (a *App) drawPersonalMarketBuyer(w, h int) {
	b := &a.state.ShopView
	x, y := a.marketUI.Origin(w, h)
	a.clearTooltipUnderPanel(x, y, ui.MarketWidth, ui.MarketHeight)
	name := a.chatEntityName(b.Seller)
	a.marketUI.Draw(a.sprites, a.font, w, h, 0, proto.MyShopSlots, a.personalMarket.pending != 0, name, nil)
	hover, hovered := ui.MarketHit(a.lastMouseX-x, a.lastMouseY-y)
	for i := range proto.MyShopSlots {
		if !b.Present[i] {
			continue
		}
		it := b.Items[i].Item
		def := a.itemsItem(it.Type, it.TypeID)
		if def == nil {
			continue
		}
		cx, cy, _ := ui.MarketCell(i)
		a.drawItemIcon(def, x+cx, y+cy, 32, white, &it)
		if count := a.itemCount(it.Type, it.TypeID, it.Count); count > 1 {
			text := strconv.Itoa(int(count))
			a.font.Draw(a.sprites, text, x+cx+31-a.font.Width(text), y+cy+18, white, 32)
		}
		if hovered && hover == i && !a.personalMarket.modal() {
			a.drawItemDetails(def, &it, it.Count, min(x+ui.MarketWidth+8, max(0, w-296)), min(y+112, max(0, h-106)), w, h, fmt.Sprintf("%d gold per unit", b.Items[i].Price))
		}
	}
}
