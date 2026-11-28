package app

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"path/filepath"

	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

type windowDrag struct {
	portrait, moved                 bool
	pressX, pressY                  int
	click                           func() // pressed without moving; defaults to selecting self for the portrait
	position                        *ui.WindowPosition
	offsetX, offsetY, width, height int
}

func (a *App) updateWindowDrag(in *platform.Input, w, h int) bool {
	d := &a.windowDrag
	if d.position != nil {
		clickable := d.portrait || d.click != nil
		if !clickable || absInt(in.MouseX-d.pressX) > 4 || absInt(in.MouseY-d.pressY) > 4 {
			d.moved = true
		}
		if d.moved {
			d.position.Move(in.MouseX-d.offsetX, in.MouseY-d.offsetY, w, h, d.width, d.height)
		}
		if in.Clicked || !in.MouseDown {
			if clickable && !d.moved && in.Clicked {
				if d.click != nil {
					d.click()
				} else {
					a.selectSelf()
				}
			}
			a.finishWindowDrag()
		}
		return true
	}
	if !in.Pressed || a.wv == nil || a.inventory.dragging || a.shop.dragging || a.bank.dragging {
		return false
	}
	start := func(position *ui.WindowPosition, x, y, width, height, handleX, handleWidth int) bool {
		if in.MouseX < x+handleX || in.MouseX >= x+handleX+handleWidth || in.MouseY < y+5 || in.MouseY >= y+30 {
			return false
		}
		a.windowDrag = windowDrag{position: position, offsetX: in.MouseX - x, offsetY: in.MouseY - y, width: width, height: height}
		a.wv.uiLeft = -2 // Title owns the entire drag, including release outside the panel.
		return true
	}
	// Native frames (0x526f30) drag from anywhere inside their rectangle once the
	// window's own controls and item cells have declined the press.
	startFrame := func(position *ui.WindowPosition, x, y, width, height int, free func() bool) bool {
		if !pointInside(in, x, y, width, height) || !free() {
			return false
		}
		a.windowDrag = windowDrag{position: position, offsetX: in.MouseX - x, offsetY: in.MouseY - y, width: width, height: height}
		a.wv.uiLeft = -2
		return true
	}
	// frameFree builds a free-area test: not on a control (found by reflection),
	// not inside the listed window-relative rectangles {x, y, w, h}, and not
	// accepted by the optional cell test.
	frameFree := func(owner any, ox, oy int, cell func(rx, ry int) bool, rects ...[4]int) func() bool {
		return func() bool {
			if ui.ButtonHit(owner, in.MouseX, in.MouseY) {
				return false
			}
			rx, ry := in.MouseX-ox, in.MouseY-oy
			if cell != nil && cell(rx, ry) {
				return false
			}
			for _, r := range rects {
				if rx >= r[0] && rx < r[0]+r[2] && ry >= r[1] && ry < r[1]+r[3] {
					return false
				}
			}
			return true
		}
	}
	if a.minimap != nil && a.wv.mapView.full {
		x, y, scale := a.minimap.FullRect(w, h)
		return start(&a.minimap.FullPosition, x, y, int(544*scale), int(579*scale), 8, int(450*scale))
	}
	if a.teleportStone.open && a.recordedPoints != nil {
		x, y := a.recordedPoints.Origin(w, h)
		if startFrame(&a.recordedPoints.Position, x, y, ui.RecordedPointsWidth, ui.RecordedPointsHeight, frameFree(a.recordedPoints, x, y, nil)) {
			return true
		}
		if pointInside(in, x, y, ui.RecordedPointsWidth, ui.RecordedPointsHeight) {
			return false
		}
	}
	// Match drawing order: the quest journal is above the other panels.
	if a.mall.open && a.mallUI != nil {
		if a.mallUI.GiftOpen {
			x, y := a.mallUI.GiftOrigin(w, h)
			if start(&a.mallUI.GiftPosition, x, y, ui.GiftBoxWidth, ui.GiftBoxHeight, 20, 270) {
				return true
			}
			if pointInside(in, x, y, ui.GiftBoxWidth, ui.GiftBoxHeight) {
				return false
			}
		}
		x, y := a.mallUI.Origin(w, h)
		if startFrame(&a.mallUI.Position, x, y, ui.MallWidth, ui.MallHeight, frameFree(a.mallUI, x, y, nil, ui.MallControlRects()...)) {
			return true
		}
		if pointInside(in, x, y, ui.MallWidth, ui.MallHeight) {
			return false
		}
	}
	if a.personalMarketOpen() {
		x, y := a.personalMarketUI.Origin(w, h)
		position, freeAt := &a.personalMarketUI.Position, a.personalMarketUI.FreeAt
		width, height := ui.PersonalMarketWidth, ui.PersonalMarketHeight
		if a.state.ShopView.Open && a.marketUI != nil {
			x, y = a.marketUI.Origin(w, h)
			position, freeAt = &a.marketUI.Position, a.marketUI.FreeAt
			width, height = ui.MarketWidth, ui.MarketHeight
		}
		if pointInside(in, x, y, width, height) {
			return startFrame(position, x, y, width, height, func() bool {
				return freeAt(in.MouseX, in.MouseY, w, h)
			})
		}
	}
	if a.guild.open && a.guildUI != nil {
		x, y := a.guildUI.Origin(w, h)
		if startFrame(&a.guildUI.Position, x, y, ui.GuildWidth, ui.GuildHeight, frameFree(a.guildUI, x, y, nil, [4]int{37, 112, 266, ui.GuildRows * 23}, [4]int{0, 61, 30, 242})) {
			return true
		}
		if pointInside(in, x, y, ui.GuildWidth, ui.GuildHeight) {
			return false
		}
	}
	if a.questLog.open && a.questUI != nil {
		x, y := a.questUI.Origin(w, h)
		if startFrame(&a.questUI.Position, x, y, ui.QuestWidth, ui.QuestHeight, frameFree(a.questUI, x, y, nil, [4]int{15, 54, 219, 228}, [4]int{275, 102, 215, 325}, [4]int{18, 320, 206, 130})) {
			return true
		}
		if pointInside(in, x, y, ui.QuestWidth, ui.QuestHeight) {
			return false
		}
	}
	// Match drawing order: the right-hand panels cover the shop when overlapped.
	if a.status.open && a.statusUI != nil {
		x, y := a.statusUI.Origin(w, h)
		width := ui.StatusWidth
		if a.statusUI.Expanded {
			width += 132
		}
		if startFrame(&a.statusUI.Position, x, y, width, ui.StatusHeight, func() bool {
			return a.statusUI.FreeAt(in.MouseX, in.MouseY, w, h, a.statusState())
		}) {
			return true
		}
		if pointInside(in, x, y, width, ui.StatusHeight) {
			return false
		}
	}
	if a.subActionsOpen && a.subActionsUI != nil {
		x, y := a.subActionsUI.Origin(w, h)
		if startFrame(&a.subActionsUI.Position, x, y, ui.SubActionWidth, ui.SubActionHeight, func() bool { return a.subActionsUI.FreeAt(in.MouseX, in.MouseY, w, h) }) {
			return true
		}
		if pointInside(in, x, y, ui.SubActionWidth, ui.SubActionHeight) {
			return false
		}
	}
	if a.skills.open && a.skillBook != nil {
		x, y := a.skillBook.Origin(w, h)
		if startFrame(&a.skillBook.Position, x, y, ui.SkillBookWidth, ui.SkillBookHeight, func() bool { return a.skillBook.FreeAt(in.MouseX, in.MouseY, w, h) }) {
			return true
		}
		if pointInside(in, x, y, ui.SkillBookWidth, ui.SkillBookHeight) {
			return false
		}
	}
	if a.inspect.open && a.inspectUI != nil {
		x, y := a.inspectUI.Origin(w, h)
		if start(&a.inspectUI.Position, x, y, ui.InspectWidth, ui.InspectHeight, 20, 193) {
			return true
		}
		if pointInside(in, x, y, ui.InspectWidth, ui.InspectHeight) {
			return false
		}
	}
	if a.inventory.open && a.inventoryUI != nil {
		x, y := a.inventoryUI.Origin(w, h)
		position := &a.inventoryUI.Position
		if _, _, docked := a.inventoryDockOrigin(w, h); docked {
			position = &a.inventory.servicePosition
		}
		if startFrame(position, x, y, ui.InventoryWidth, ui.InventoryHeight, func() bool { return a.inventoryUI.FreeAt(in.MouseX, in.MouseY, w, h, a.inventory.bag) }) {
			return true
		}
		if pointInside(in, x, y, ui.InventoryWidth, ui.InventoryHeight) {
			return false
		}
	}
	if a.duelUI != nil && a.state.Duel.Phase == game.DuelWager {
		x, y := a.duelUI.Origin(w, h)
		if startFrame(&a.duelUI.Position, x, y, ui.ExchangeWidth, ui.ExchangeHeight, frameFree(a.duelUI, x, y, func(rx, ry int) bool { _, ok := ui.ExchangeHit(rx, ry); return ok })) {
			return true
		}
		if pointInside(in, x, y, ui.ExchangeWidth, ui.ExchangeHeight) {
			return false
		}
	}
	if a.exchangeUI != nil && a.state.Exchange.Active {
		x, y := a.exchangeUI.Origin(w, h)
		if startFrame(&a.exchangeUI.Position, x, y, ui.ExchangeWidth, ui.ExchangeHeight, frameFree(a.exchangeUI, x, y, func(rx, ry int) bool { _, ok := ui.ExchangeHit(rx, ry); return ok })) {
			return true
		}
		if pointInside(in, x, y, ui.ExchangeWidth, ui.ExchangeHeight) {
			return false
		}
	}
	if e := a.state.Entities[a.shop.npcID]; e != nil && e.NpcType == 4 && a.pvpRewardUI != nil {
		if a.pvpRewardMessage != nil && a.pvpRewardMessage.Visible {
			return false
		}
		x, y := a.pvpRewardUI.Origin(w, h)
		if startFrame(&a.pvpRewardUI.Position, x, y, ui.PvPRewardWidth, ui.PvPRewardHeight, frameFree(a.pvpRewardUI, x, y, nil, [4]int{15, 82, 240, 244})) {
			return true
		}
		if pointInside(in, x, y, ui.PvPRewardWidth, ui.PvPRewardHeight) {
			return false
		}
	}
	if e := a.state.Entities[a.shop.npcID]; e != nil && a.shop.talk && a.npcTalkUI != nil {
		x, y := a.npcTalkUI.Origin(w, h)
		if startFrame(&a.npcTalkUI.Position, x, y, ui.NPCTalkWidth, ui.NPCTalkHeight, frameFree(a.npcTalkUI, x, y, nil, [4]int{16, 82, 227, 225})) {
			return true
		}
		if pointInside(in, x, y, ui.NPCTalkWidth, ui.NPCTalkHeight) {
			return false
		}
	}
	if e := a.state.Entities[a.shop.npcID]; e != nil && a.npcQuestUI != nil && !a.shop.talk && (a.shop.questMode || e.NpcType != 1 && e.NpcType != 2 && e.NpcType != 3 && (e.NpcType != 4 || a.pvpRewardUI == nil) && e.NpcType != 6 && e.NpcType != 11) {
		a.npcQuestUI.Detail = a.npcQuest.selected != 0
		x, y := a.npcQuestUI.Origin(w, h)
		height := 391
		if a.npcQuest.selected != 0 {
			height = 512
		}
		if startFrame(&a.npcQuestUI.Position, x, y, ui.NPCQuestWidth, height, frameFree(a.npcQuestUI, x, y, nil, [4]int{16, 110, 224, 360})) {
			return true
		}
		if pointInside(in, x, y, ui.NPCQuestWidth, height) {
			return false
		}
	}
	if e := a.state.Entities[a.shop.npcID]; e != nil && e.NpcType == 2 && !a.shop.overlay() && a.gateUI != nil {
		x, y := a.gateUI.Origin(w, h)
		if startFrame(&a.gateUI.Position, x, y, ui.GatekeeperWidth, ui.GatekeeperHeight, frameFree(a.gateUI, x, y, nil, [4]int{16, 82, 226, 167})) {
			return true
		}
		if pointInside(in, x, y, ui.GatekeeperWidth, ui.GatekeeperHeight) {
			return false
		}
	}
	if a.guildHouseUI != nil && a.houseWindowOpenNow() {
		x, y := a.guildHouseUI.Origin(w, h)
		if startFrame(&a.guildHouseUI.Position, x, y, ui.GuildHouseWidth, ui.GuildHouseHeight, frameFree(a.guildHouseUI, x, y, nil, [4]int{270, 60, 305, 380})) {
			return true
		}
		if pointInside(in, x, y, ui.GuildHouseWidth, ui.GuildHouseHeight) {
			return false
		}
	}
	if a.appearanceEdit != nil && !a.appearanceEdit.confirming && a.appearanceUI != nil {
		x, y := a.appearanceUI.Origin(w, h)
		if startFrame(&a.appearanceUI.Position, x, y, ui.AppearanceWidth, ui.AppearanceHeight, frameFree(a.appearanceUI, x, y, nil)) {
			return true
		}
		if pointInside(in, x, y, ui.AppearanceWidth, ui.AppearanceHeight) {
			return false
		}
	}
	if a.emblemOpen() {
		x, y := a.emblemUI.Origin(w, h)
		if startFrame(&a.emblemUI.Position, x, y, ui.EmblemWidth, ui.EmblemHeight, frameFree(a.emblemUI, x, y, func(rx, ry int) bool { return ui.EmblemPaletteHit(rx, ry) >= 0 }, [4]int{0x218, 0x3a, 0x50, 0x4e})) {
			return true
		}
		if pointInside(in, x, y, ui.EmblemWidth, ui.EmblemHeight) {
			return false
		}
	}
	if a.blacksmithOpen() {
		x, y := a.blacksmithUI.Origin(w, h)
		bw, bh := a.blacksmithUI.Size()
		if startFrame(&a.blacksmithUI.Position, x, y, bw, bh, frameFree(a.blacksmithUI, x, y, func(rx, ry int) bool { return a.blacksmithUI.Hit(rx, ry) >= 0 })) {
			return true
		}
		if pointInside(in, x, y, bw, bh) {
			return false
		}
	}
	if a.auctionBoard(a.shop.npcID) {
		x, y, width := a.auctionPanel(w, h)
		// The strip above the tabs is the only free part of the native frame.
		if in.MouseY < y+0x14 && start(&a.auction.position, x, y, width, auctionHeight(h), 8, width-64) {
			return true
		}
		if pointInside(in, x, y, width, auctionHeight(h)) {
			return false
		}
	}
	if a.merchantOpen() {
		x, y := a.marketUI.Origin(w, h)
		if startFrame(&a.marketUI.MerchantPosition, x, y, ui.MarketWidth, ui.MarketHeight, func() bool {
			return a.marketUI.FreeAt(in.MouseX, in.MouseY, w, h)
		}) {
			return true
		}
		if pointInside(in, x, y, ui.MarketWidth, ui.MarketHeight) {
			return false
		}
	}
	if e := a.state.Entities[a.shop.npcID]; e != nil && e.NpcType == 11 && a.guildDirectoryUI != nil {
		x, y := a.guildDirectoryUI.Origin(w, h)
		if startFrame(&a.guildDirectoryUI.Position, x, y, ui.GuildDirectoryWidth, ui.GuildDirectoryHeight, frameFree(a.guildDirectoryUI, x, y, nil, [4]int{0x112, 0x6e, 0x11e, ui.GuildDirectoryRows * 0x1b})) {
			return true
		}
		if pointInside(in, x, y, ui.GuildDirectoryWidth, ui.GuildDirectoryHeight) {
			return false
		}
	}
	if a.warehouseOpen() {
		x, y := a.bankUI.Origin(w, h)
		if startFrame(&a.bankUI.Position, x, y, ui.WarehouseWidth, ui.WarehouseHeight, func() bool { return a.bankUI.FreeAt(in.MouseX, in.MouseY, w, h) }) {
			return true
		}
		if pointInside(in, x, y, ui.WarehouseWidth, ui.WarehouseHeight) {
			return false
		}
	}
	if a.questTracker.open && a.questTrackerUI != nil {
		q := a.questTrackerUI
		x, y := q.Origin(w, h)
		// 0x4dc540 falls through to the native frame drag (0x526f30).
		if startFrame(&q.Position, x, y, ui.QuestTrackerWidth, ui.QuestTrackerHeight, frameFree(q, x, y, func(rx, ry int) bool { return q.TitleAt(rx, ry) != 0 }, [4]int{224, 38, 13, 77})) {
			return true
		}
		if pointInside(in, x, y, ui.QuestTrackerWidth, ui.QuestTrackerHeight) {
			return false
		}
	}
	xq, yq := ui.QuickPotionOrigin(a.quickPotions.Position, w, h)
	// The frame and any empty cell move the bar; a filled cell is grabbed as a potion.
	if cell := ui.QuickPotionCell(in.MouseX, in.MouseY, xq, yq); pointInside(in, xq, yq, 94, 54) && (cell < 0 || a.quickPotions.bindings[cell] == (sdata.ItemKey{})) && !a.barCovered(in, w, h) {
		a.windowDrag = windowDrag{position: &a.quickPotions.Position, offsetX: in.MouseX - xq, offsetY: in.MouseY - yq, width: 94, height: 54}
		a.wv.uiLeft = -2
		return true
	}
	if a.hud != nil {
		if a.targetFrameShown() {
			x, y := a.hud.TargetOrigin(w, h)
			if pointInside(in, x, y, 256, 64) {
				a.windowDrag = windowDrag{position: &a.hud.TargetPosition, offsetX: in.MouseX - x, offsetY: in.MouseY - y, width: 256, height: 64}
				a.wv.uiLeft = -2
				return true
			}
		}
		x, y := a.hud.Origin(w, h)
		if pointInside(in, x+17, y+26, 55, 53) {
			a.windowDrag = windowDrag{position: &a.hud.Position, offsetX: in.MouseX - x, offsetY: in.MouseY - y, width: 213, height: 114, portrait: true, pressX: in.MouseX, pressY: in.MouseY}
			a.wv.uiLeft = -2
			return true
		}
		if startFrame(&a.hud.Position, x, y, 213, 114, func() bool { return true }) {
			return true
		}
	}
	if a.minimap != nil && !a.wv.mapView.compactHidden {
		x, y := a.minimap.Origin(w, h)
		// Drag the rim or lower information plate; leave buttons and map zoom free.
		if a.minimap.DragHandleAt(in.MouseX, in.MouseY, w, h) {
			a.windowDrag = windowDrag{position: &a.minimap.Position, offsetX: in.MouseX - x, offsetY: in.MouseY - y, width: ui.MinimapWidth, height: ui.MinimapHeight}
			a.wv.uiLeft = -2
			return true
		}
	}
	for group, count := range a.buffCounts() {
		if count == 0 {
			continue
		}
		x, y, width, height := a.buffLayout.Rect(w, h, group, count)
		if pointInside(in, x, y, width, height) {
			a.windowDrag = windowDrag{position: &a.buffLayout.Position[group], offsetX: in.MouseX - x, offsetY: in.MouseY - y, width: width, height: height}
			if i := a.buffHit(w, h, in.MouseX, in.MouseY); i >= 0 && a.buffs.rows[i].buff.Skill == warehouseCallSkill {
				a.windowDrag.pressX, a.windowDrag.pressY = in.MouseX, in.MouseY
				instance := a.buffs.rows[i].buff.Instance
				a.windowDrag.click = func() {
					if i < len(a.buffs.rows) && a.buffs.rows[i].buff.Instance == instance && a.buffs.rows[i].buff.Skill == warehouseCallSkill {
						a.openRemoteWarehouse()
					}
				}
			}
			a.wv.uiLeft = -2
			return true
		}
	}
	if members := a.state.Party.Members; a.partyUI != nil && !a.state.Party.Raid && len(members) > 0 {
		// Rows stay clickable: a press that never moves selects the member,
		// moving more than a few pixels drags the whole list.
		x, y := a.partyUI.Origin(w, h, len(members))
		if pointInside(in, x, y, ui.PartyWidth, len(members)*ui.PartyRowHeight) {
			id := members[(in.MouseY-y)/ui.PartyRowHeight].ID
			a.windowDrag = windowDrag{position: &a.partyUI.Position, offsetX: in.MouseX - x, offsetY: in.MouseY - y, width: ui.PartyWidth, height: ui.PartyRowHeight, portrait: true, pressX: in.MouseX, pressY: in.MouseY,
				click: func() { a.state.Target = id }}
			a.wv.uiLeft = -2
			return true
		}
	}
	if a.partyUI != nil && a.raidGridShown() && !a.unionConfigCovers(in, w, h) {
		x, y := a.partyUI.RaidGridOrigin(w, h)
		if pointInside(in, x, y, ui.RaidWindowWidth, ui.RaidGridHeight) {
			slot := a.partyUI.RaidGridSlotAt(w, h, in.MouseX, in.MouseY)
			id := uint32(0)
			if slot >= 0 {
				id = a.state.Party.RaidSlots[uint16(slot)]
			}
			a.windowDrag = windowDrag{position: &a.partyUI.RaidGridPosition, offsetX: in.MouseX - x, offsetY: in.MouseY - y, width: ui.RaidWindowWidth, height: ui.RaidGridHeight, portrait: true, pressX: in.MouseX, pressY: in.MouseY,
				click: func() {
					// A plain click either finishes Move Group (empty cells too) or selects.
					if a.party.moving {
						a.finishMove(slot)
					} else if id != 0 {
						a.state.Target = id
					}
				}}
			a.wv.uiLeft = -2
			return true
		}
	}
	if a.partyUI != nil && a.state.Party.Raid && !a.unionConfigCovers(in, w, h) {
		// The raid's side list moves like the party list: a press that never
		// moves selects the member, a drag takes the whole list.
		x, y := a.partyUI.RaidCompactOrigin(w, h)
		height := 6 * ui.PartyRowHeight
		if a.partyUI.RaidCompactGroups() {
			height += 24
		}
		body := y
		if a.partyUI.RaidCompactGroups() {
			body += 24
		}
		onTabs := a.partyUI.RaidCompactGroups() && in.MouseY < body && in.MouseX >= x+20 && in.MouseX < x+140
		if pointInside(in, x, y, ui.RaidCompactWidth, height) && !onTabs {
			id := a.partyUI.RaidCompactMemberAt(w, h, in.MouseX, in.MouseY, a.state.SelfID, a.state.Party.RaidSlots)
			slot := a.partyUI.RaidCompactSlotAt(w, h, in.MouseX, in.MouseY, a.state.SelfID, a.state.Party.RaidSlots)
			a.windowDrag = windowDrag{position: &a.partyUI.RaidCompactPosition, offsetX: in.MouseX - x, offsetY: in.MouseY - y, width: ui.RaidCompactWidth, height: height, portrait: true, pressX: in.MouseX, pressY: in.MouseY,
				click: func() {
					if a.party.moving {
						a.finishMove(slot)
					} else if id != 0 {
						a.state.Target = id
					}
				}}
			a.wv.uiLeft = -2
			return true
		}
	}
	if a.partyUI != nil && a.unionConfigOpen() {
		x, y := a.partyUI.RaidOrigin(w, h)
		if start(&a.partyUI.RaidPosition, x, y, ui.RaidWidth, ui.RaidHeight, 20, 520) {
			return true
		}
	}
	return false
}
func pointInside(in *platform.Input, x, y, w, h int) bool {
	return in.MouseX >= x && in.MouseX < x+w && in.MouseY >= y && in.MouseY < y+h
}
func (a *App) finishWindowDrag() {
	if position := a.windowDrag.position; position != nil {
		a.windowDrag = windowDrag{}
		if position != &a.inventory.servicePosition && (a.marketUI == nil || position != &a.marketUI.MerchantPosition) {
			a.saveUILayout()
		}
	}
}
func (a *App) layoutPositions() map[string]*ui.WindowPosition {
	out := map[string]*ui.WindowPosition{"quick_potions": &a.quickPotions.Position, "revolver": &a.revolver.Position, "buffs": &a.buffLayout.Position[0], "debuffs": &a.buffLayout.Position[1]}
	if a.hud != nil {
		out["portrait_hud"] = &a.hud.Position
		out["target_hud"] = &a.hud.TargetPosition
	}
	if a.minimap != nil {
		out["minimap"] = &a.minimap.Position
		out["full_map"] = &a.minimap.FullPosition
	}
	if a.gameMenu != nil {
		out["game_menu"] = &a.gameMenu.Position
	}
	if a.actionBar != nil {
		out["action_bar_0"] = &a.actionBar.Position[0]
		out["action_bar_1"] = &a.actionBar.Position[1]
		out["action_bar_2"] = &a.actionBar.Position[2]
	}
	if a.subActionsUI != nil {
		out["sub_actions"] = &a.subActionsUI.Position
	}
	if a.personalMarketUI != nil {
		out["personal_market"] = &a.personalMarketUI.Position
	}
	if a.partyUI != nil {
		out["party"] = &a.partyUI.Position
		out["raid"] = &a.partyUI.RaidPosition
		out["raid_list"] = &a.partyUI.RaidCompactPosition
		out["raid_grid"] = &a.partyUI.RaidGridPosition
	}
	if a.mallUI != nil {
		out["mall"] = &a.mallUI.Position
		out["mall_gifts"] = &a.mallUI.GiftPosition
	}
	if a.guildUI != nil {
		out["guild"] = &a.guildUI.Position
	}
	if a.guildDirectoryUI != nil {
		out["guild_directory"] = &a.guildDirectoryUI.Position
	}
	if a.duelUI != nil {
		out["duel"] = &a.duelUI.Position
	}
	if a.exchangeUI != nil {
		out["exchange"] = &a.exchangeUI.Position
	}
	if a.questUI != nil {
		out["quest"] = &a.questUI.Position
	}
	if a.questTrackerUI != nil {
		out["quest_tracker"] = &a.questTrackerUI.Position
	}
	if a.inspectUI != nil {
		out["inspect"] = &a.inspectUI.Position
	}
	if a.inventoryUI != nil {
		out["inventory"] = &a.inventoryUI.Position
	}
	if a.skillBook != nil {
		out["skills"] = &a.skillBook.Position
	}
	if a.statusUI != nil {
		out["status"] = &a.statusUI.Position
	}
	if a.npcQuestUI != nil {
		out["npc_quest"] = &a.npcQuestUI.Position
	}
	if a.marketUI != nil {
		out["shop"] = &a.marketUI.Position
	}
	if a.pvpRewardUI != nil {
		out["pvp_rewards"] = &a.pvpRewardUI.Position
	}
	if a.gateUI != nil {
		out["gatekeeper"] = &a.gateUI.Position
	}
	if a.guildHouseUI != nil {
		out["guildHouse"] = &a.guildHouseUI.Position
	}
	if a.appearanceUI != nil {
		out["appearance"] = &a.appearanceUI.Position
	}
	if a.emblemUI != nil {
		out["emblem"] = &a.emblemUI.Position
	}
	if a.blacksmithUI != nil {
		out["blacksmith"] = &a.blacksmithUI.Position
	}
	if a.bankUI != nil {
		out["warehouse"] = &a.bankUI.Position
	}
	return out
}
func uiLayoutPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "shaiya-go", "ui-layout.json"), nil
}

type savedWindowLayout struct {
	ui.WindowPosition
	Hidden bool `json:",omitempty"`
	// Raid carries the raid window options (transparency, list and tabs) under "raid_view".
	Raid *savedRaidView `json:",omitempty"`
}

type savedRaidView struct {
	Opacity         float32
	Compact, Groups bool
}

func (a *App) loadUILayout() {
	if !a.cfg.RememberLayout {
		return
	}
	path, err := uiLayoutPath()
	var data []byte
	if err == nil {
		data, err = os.ReadFile(path)
	}
	if errors.Is(err, os.ErrNotExist) {
		return
	}
	values := map[string]savedWindowLayout{}
	if err == nil {
		err = json.Unmarshal(data, &values)
	}
	if err != nil {
		log.Printf("load UI layout: %v", err)
		return
	}
	for key, position := range a.layoutPositions() {
		if saved, ok := values[key]; ok && saved.Valid() {
			*position = saved.WindowPosition
		}
	}
	if v, ok := values["raid_view"]; ok && v.Raid != nil && a.partyUI != nil {
		a.partyUI.SetRaidViewSettings(v.Raid.Opacity, v.Raid.Compact, v.Raid.Groups)
	}
	if a.actionBar != nil {
		a.actionBar.HideSecond = values["action_bar_1"].Hidden
		if third, ok := values["action_bar_2"]; ok {
			a.actionBar.ShowThird = !third.Hidden && !a.actionBar.HideSecond
		}
	}
}
func (a *App) saveUILayout() {
	if !a.cfg.RememberLayout {
		return
	}
	path, err := uiLayoutPath()
	if err == nil {
		err = os.MkdirAll(filepath.Dir(path), 0700)
	}
	if err == nil {
		var data []byte
		positions := a.layoutPositions()
		values := make(map[string]savedWindowLayout, len(positions))
		for key, position := range positions {
			values[key] = savedWindowLayout{WindowPosition: *position, Hidden: key == "action_bar_1" && a.actionBar.HideSecond || key == "action_bar_2" && !a.actionBar.ShowThird}
		}
		if a.partyUI != nil {
			opacity, compact, groups := a.partyUI.RaidViewSettings()
			values["raid_view"] = savedWindowLayout{Raid: &savedRaidView{Opacity: opacity, Compact: compact, Groups: groups}}
		}
		data, err = json.Marshal(values)
		if err == nil {
			var file *os.File
			file, err = os.CreateTemp(filepath.Dir(path), ".ui-layout-*")
			if err == nil {
				defer os.Remove(file.Name())
				_, err = file.Write(data)
				closeErr := file.Close()
				if err == nil {
					err = closeErr
				}
				if err == nil {
					err = os.Rename(file.Name(), path)
				}
			}
		}
	}
	if err != nil {
		log.Printf("save UI layout: %v", err)
	}
}

// unionConfigCovers: the raid window sits over the side list when they overlap.
func (a *App) unionConfigCovers(in *platform.Input, w, h int) bool {
	return a.unionConfigOpen() && a.partyUI.RaidInside(w, h, in.MouseX, in.MouseY)
}
