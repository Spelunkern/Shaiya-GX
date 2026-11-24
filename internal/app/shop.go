package app

import (
	"fmt"
	"math"

	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

type shopView struct {
	inventoryPosition ui.WindowPosition
	quantity          *tradeQuantity
	npcID             uint32
	page              int
	pending           bool
	deadline          float64
	dragging          bool
	questMode         bool
	talk              bool // the NPC conversation window precedes the service
	dragIndex         uint8
}

func (a *App) merchantOpen() bool {
	e := a.state.Entities[a.shop.npcID]
	if a.marketUI == nil || e == nil || e.NpcType != 1 || a.shop.overlay() || a.state.Self.Dead || a.auctionBoard(a.shop.npcID) || a.emblemNPC(e) {
		return false
	}
	_, house := a.houseKind(e)
	return !house
}

func (a *App) updateShop(in *platform.Input, w, h int) bool {
	v := &a.shop
	if a.state.Self.Dead {
		v.npcID = 0
		return false
	}
	if a.state.Entities[v.npcID] == nil {
		v.npcID = 0
	}
	if v.npcID == 0 {
		if a.remoteWarehouseOpen() {
			return a.updateWarehouse(in, w, h)
		}
		return false
	}
	e := a.state.Entities[v.npcID]
	if a.auctionBoard(v.npcID) {
		return a.updateAuction(in, w, h)
	}
	if v.talk && a.npcTalkUI != nil {
		return a.updateNPCTalk(in, w, h, e)
	}
	if v.questMode && a.npcQuestUI != nil {
		return a.updateNPCQuest(in, w, h, e)
	}
	if kind, ok := a.houseKind(e); ok {
		return a.updateHouse(in, w, h, e, kind)
	}
	if a.emblemNPC(e) {
		return a.updateEmblem(in, w, h, e)
	}
	if e.NpcType == 4 && a.pvpRewardUI != nil {
		return a.updatePvPRewards(in, w, h)
	}
	if e.NpcType == 11 && a.guildDirectoryUI != nil {
		return a.updateGuildDirectory(in, w, h, e)
	}
	if e.NpcType == 2 && a.gateUI != nil {
		return a.updateGatekeeper(in, w, h, e)
	}
	if e.NpcType == 3 && a.blacksmithUI != nil {
		return a.updateBlacksmith(in, w, h)
	}
	if e.NpcType == 6 && a.bankUI != nil {
		return a.updateWarehouse(in, w, h)
	}
	if e.NpcType != 1 && e.NpcType != 6 && a.npcQuestUI != nil {
		return a.updateNPCQuest(in, w, h, e)
	}
	def := a.npcDefs[sdata.NPCKey{Type: e.NpcType, ID: e.TypeID}]
	x, y := a.marketUI.Origin(w, h)
	width, height := ui.MarketWidth, ui.MarketHeight
	if e.NpcType != 1 {
		x, y, width, height = 32, 82, 400, 460
	}
	captured := panelPointer(in, x, y, width, height)
	busy := v.pending || a.inventory.pending
	closed := in.HasKey(platform.KeyEscape)
	if e.NpcType == 1 {
		var clickedClose bool
		v.page, clickedClose = a.marketUI.Update(in, w, h, v.page, len(def.Items), busy)
		closed = closed || clickedClose
	}
	if closed {
		if err := a.world.Send(proto.NpcTalkPacket(0)); err != nil {
			a.backToLoginID(12011)
			return true
		}
		v.npcID = 0
		return true
	}
	if busy || e.NpcType != 1 {
		return captured
	}
	pages := max(1, (len(def.Items)+ui.MarketPageSize-1)/ui.MarketPageSize)
	if captured && in.Wheel < 0 {
		v.page++
	}
	if captured && in.Wheel > 0 {
		v.page--
	}
	v.page = max(0, min(v.page, pages-1))
	if cell, ok := ui.MarketHit(in.MouseX-x, in.MouseY-y); ok && in.Pressed {
		index := v.page*ui.MarketPageSize + cell
		if index < len(def.Items) && index < 256 && def.Items[index].Type != 0 {
			v.dragging, v.dragIndex = true, uint8(index)
			a.inventory.selected = false
		}
	}

	return captured
}

func (a *App) openNPC(id uint32) {
	if a.state == nil || a.wv == nil || a.state.Self.Dead || a.world == nil || a.inventory.pending || a.shop.pending {
		return
	}
	e := a.state.Entities[id]
	if e == nil || e.Kind != game.KindNpc {
		return
	}
	pos := a.visualEntityPosition(e)
	dx, dz := pos[0]-a.wv.pos[0], pos[2]-a.wv.pos[2]
	serverX, serverZ := e.X-a.wv.pos[0], e.Z-a.wv.pos[2]
	if dx*dx+dz*dz > 2*2 || serverX*serverX+serverZ*serverZ >= 9.5*9.5 {
		if dx*dx+dz*dz < 150*150 {
			a.wv.autoRun = false
			a.wv.approachTarget, a.wv.approachNPC, a.wv.approachLoot = id, true, false
			a.wv.approachSkill = nil
			a.updateActionApproach(&platform.Input{})
		}
		return
	}
	a.loadNPCs()
	a.wv.approachTarget, a.wv.approachSkill = 0, nil
	a.wv.approachNPC, a.wv.travel, a.wv.autoRun = false, false, false
	if err := a.world.Send(proto.NpcTalkPacket(id)); err != nil {
		a.backToLoginID(12184)
		return
	}
	a.shop.npcID, a.shop.page, a.shop.questMode, a.shop.talk = id, 0, false, false
	a.shop.inventoryPosition = ui.WindowPosition{}
	if a.auctionBoard(id) {
		a.beginAuction(id)
		return
	}
	a.auction = auctionView{}
	if a.npcTalkUI != nil {
		a.shop.talk = true
		a.npcTalkUI.Reset()
		return
	}
	a.enterNPCService(e)
}

func (a *App) drawShop(w, h int) {
	v := &a.shop
	e := a.state.Entities[v.npcID]
	if e == nil {
		if v.npcID == 0 && a.remoteWarehouseOpen() {
			a.drawWarehouse(w, h, nil)
		}
		return
	}
	if a.auctionBoard(v.npcID) {
		a.drawAuction(w, h)
		return
	}
	if v.talk && a.npcTalkUI != nil {
		a.drawNPCTalk(w, h, e)
		return
	}
	if v.questMode && a.npcQuestUI != nil {
		a.drawNPCQuest(w, h, e)
		return
	}
	if kind, ok := a.houseKind(e); ok {
		a.drawHouse(w, h, e, kind)
		return
	}
	if a.emblemNPC(e) {
		a.drawEmblem(w, h)
		return
	}
	if e.NpcType == 4 && a.pvpRewardUI != nil {
		a.drawPvPRewards(w, h)
		return
	}
	if e.NpcType == 11 && a.guildDirectoryUI != nil {
		a.drawGuildDirectory(w, h)
		return
	}
	def := a.npcDefs[sdata.NPCKey{Type: e.NpcType, ID: e.TypeID}]
	if e.NpcType == 2 && a.gateUI != nil {
		a.drawGatekeeper(w, h, e, def)
		return
	}
	if e.NpcType == 3 && a.blacksmithUI != nil {
		a.drawBlacksmith(w, h, def.Name, questText(def.Welcome))
		return
	}
	if e.NpcType == 6 && a.bankUI != nil {
		a.drawWarehouse(w, h, a.npcPortrait(def))
		return
	}
	if e.NpcType != 1 && e.NpcType != 6 && a.npcQuestUI != nil {
		a.drawNPCQuest(w, h, e)
		return
	}
	if e.NpcType == 1 {
		a.drawMarket(w, h, def)
		return
	}
	a.sprites.Draw(a.white, 32, 82, 400, 460, 0, 0, 1, 1, [4]float32{0.05, 0.05, 0.05, 0.95})
	a.font.Draw(a.sprites, "Esc: close", 40, 510, white, 380)
}

func (a *App) drawMarket(w, h int, def sdata.NPC) {
	v := &a.shop
	x, y := a.marketUI.Origin(w, h)
	a.clearTooltipUnderPanel(x, y, ui.MarketWidth, ui.MarketHeight)
	portrait := a.npcPortrait(def)
	a.marketUI.Draw(a.sprites, a.font, w, h, v.page, len(def.Items), v.pending || a.inventory.pending, def.Name, portrait)
	start := v.page * ui.MarketPageSize
	for i := start; i < min(start+ui.MarketPageSize, len(def.Items)); i++ {
		it := def.Items[i]
		if d := a.itemsItem(it.Type, it.TypeID); d != nil {
			cx, cy, _ := ui.MarketCell(i - start)
			a.drawItemIcon(d, x+cx, y+cy, 32, white)
		}
	}
	message := ""
	switch a.state.LastShop.Result {
	case 1:
		message = a.tr(12258)
		if a.state.LastShop.Bag != 0 {
			// Sale refusals identify the source stack. Buy failures use bag zero.
			message = a.tr(467)
		}
	case 2:
		message = a.tr(238)
	}
	if v.pending {
		message = a.tr(12259)
	}
	a.font.Draw(a.sprites, message, x, y+ui.MarketHeight+4, white, 256)
	if cell, ok := ui.MarketHit(a.lastMouseX-x, a.lastMouseY-y); ok && !v.dragging && start+cell < len(def.Items) {
		it := def.Items[start+cell]
		if d := a.itemsItem(it.Type, it.TypeID); d != nil {
			a.drawItemDetails(d, nil, 1, min(x+ui.MarketWidth+8, max(0, w-296)), min(y+112, max(0, h-106)), w, h, a.shopPriceText(def, d))
		}
	}
}

func (a *App) buyItem(index uint8, count uint16) {
	if err := a.world.Send(proto.ShopBuyPacket(a.shop.npcID, index, count)); err != nil {
		a.backToLoginID(12185)
		return
	}
	a.shop.pending, a.shop.deadline = true, a.clock+10
}

func (a *App) sellItem(source [2]uint8, count uint16) {
	if err := a.world.Send(proto.ShopSellPacket(source[0], source[1], count)); err != nil {
		a.backToLoginID(12186)
		return
	}
	a.inventory.pending, a.inventory.deadline, a.inventory.selected = true, a.clock+10, false
}

// overlay reports a conversation or quest window covering the NPC's service.
func (v *shopView) overlay() bool { return v.questMode || v.talk }

// shopPriceText is the price line of a merchant tooltip. In a guild house the
// guild's discount for that kind of shop applies (the server charges the same).
func (a *App) shopPriceText(def sdata.NPC, d *sdata.Item) string {
	g := &a.state.Guild
	if !houseMaps[a.state.MapID] || g.ID == 0 || def.MerchantType < 1 || def.MerchantType > 4 || g.HouseShopDiscount[def.MerchantType-1] == 0 {
		return fmt.Sprintf("Buy: %d gold", d.BuyPrice)
	}
	price := float64(max(d.BuyPrice, 0))
	reduced := uint32(math.Floor(price*float64(100-g.HouseShopDiscount[def.MerchantType-1])/100 + 0.5))
	state := "ok"
	if a.state.Details.Money < reduced {
		state = "short"
	}
	return fmt.Sprintf("HouseBuy:%s|%d|%d", state, reduced, max(d.BuyPrice, 0))
}
