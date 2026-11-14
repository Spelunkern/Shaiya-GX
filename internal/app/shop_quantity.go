package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

type tradeQuantity struct {
	npcID  uint32
	buy    bool
	index  uint8
	source [2]uint8
	item   proto.InventoryItem
}

func (a *App) choosePurchase(index uint8, in *platform.Input, w, h int) {
	e := a.state.Entities[a.shop.npcID]
	if e == nil || e.NpcType != 1 {
		return
	}
	def := a.npcDefs[sdata.NPCKey{Type: e.NpcType, ID: e.TypeID}]
	if int(index) >= len(def.Items) {
		return
	}
	it := def.Items[index]
	d := a.itemsItem(it.Type, it.TypeID)
	if d == nil {
		return
	}
	if d.Count <= 1 {
		a.buyItem(index, 1)
		return
	}
	a.shop.quantity = &tradeQuantity{npcID: a.shop.npcID, buy: true, index: index}
	a.quantityUI.Open(in.MouseX, in.MouseY, w, h, int(d.Count))
}
func (a *App) chooseSale(source [2]uint8, in *platform.Input, w, h int) {
	it, ok := a.state.Inventory[source]
	if !ok {
		return
	}
	if it.Count <= 1 {
		a.sellItem(source, 1)
		return
	}
	a.shop.quantity = &tradeQuantity{npcID: a.shop.npcID, source: source, item: it}
	a.quantityUI.Open(in.MouseX, in.MouseY, w, h, int(it.Count))
}
func (a *App) updateTradeQuantity(in *platform.Input) {
	q := a.shop.quantity
	e := a.state.Entities[q.npcID]
	// Never apply a stale dialog to another merchant or a replaced bag item.
	if e == nil || e.NpcType != 1 || a.state.Self.Dead || a.shop.npcID != q.npcID || a.shop.pending || a.inventory.pending ||
		(!q.buy && a.state.Inventory[q.source] != q.item) {
		a.shop.quantity = nil
		return
	}
	count := a.quantityUI.Update(in)
	if count == 0 {
		return
	}
	a.shop.quantity = nil
	if count < 0 {
		return
	}
	if q.buy {
		a.buyItem(q.index, uint16(count))
	} else {
		a.sellItem(q.source, uint16(count))
	}
}
