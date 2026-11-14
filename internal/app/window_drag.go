package app

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"path/filepath"

	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

type windowDrag struct {
	portrait, moved                 bool
	pressX, pressY                  int
	position                        *ui.WindowPosition
	offsetX, offsetY, width, height int
}

func (a *App) updateWindowDrag(in *platform.Input, w, h int) bool {
	d := &a.windowDrag
	if d.position != nil {
		if !d.portrait || absInt(in.MouseX-d.pressX) > 4 || absInt(in.MouseY-d.pressY) > 4 {
			d.moved = true
		}
		if d.moved {
			d.position.Move(in.MouseX-d.offsetX, in.MouseY-d.offsetY, w, h, d.width, d.height)
		}
		if in.Clicked || !in.MouseDown {
			if d.portrait && !d.moved && in.Clicked {
				a.selectSelf()
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
	if a.minimap != nil && a.wv.mapView.full {
		x, y, scale := a.minimap.FullRect(w, h)
		return start(&a.minimap.FullPosition, x, y, int(544*scale), int(579*scale), 8, int(450*scale))
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
		if start(&a.mallUI.Position, x, y, ui.MallWidth, ui.MallHeight, 20, 520) {
			return true
		}
		if pointInside(in, x, y, ui.MallWidth, ui.MallHeight) {
			return false
		}
	}
	if a.guild.open && a.guildUI != nil {
		x, y := a.guildUI.Origin(w, h)
		if start(&a.guildUI.Position, x, y, ui.GuildWidth, ui.GuildHeight, 30, 270) {
			return true
		}
		if pointInside(in, x, y, ui.GuildWidth, ui.GuildHeight) {
			return false
		}
	}
	if a.questLog.open && a.questUI != nil {
		x, y := a.questUI.Origin(w, h)
		if start(&a.questUI.Position, x, y, ui.QuestWidth, ui.QuestHeight, 8, 460) {
			return true
		}
		if pointInside(in, x, y, ui.QuestWidth, ui.QuestHeight) {
			return false
		}
	}
	// Match drawing order: the right-hand panels cover the shop when overlapped.
	if a.subActionsOpen && a.subActionsUI != nil {
		x, y := a.subActionsUI.Origin(w, h)
		if start(&a.subActionsUI.Position, x, y, ui.SubActionWidth, ui.SubActionHeight, 25, 225) {
			return true
		}
		if pointInside(in, x, y, ui.SubActionWidth, ui.SubActionHeight) {
			return false
		}
	}
	if a.status.open && a.statusUI != nil {
		x, y := a.statusUI.Origin(w, h)
		width, handleX := ui.StatusWidth, 0
		if a.statusUI.Expanded {
			width += 132
			handleX = 132
		}
		if start(&a.statusUI.Position, x, y, width, ui.StatusHeight, handleX, 245) {
			return true
		}
		if pointInside(in, x, y, width, ui.StatusHeight) {
			return false
		}
	}
	if a.skills.open && a.skillBook != nil {
		x, y := a.skillBook.Origin(w, h)
		if start(&a.skillBook.Position, x, y, ui.SkillBookWidth, ui.SkillBookHeight, 8, 288) {
			return true
		}
		if pointInside(in, x, y, ui.SkillBookWidth, ui.SkillBookHeight) {
			return false
		}
	}
	if a.inventory.open && a.inventoryUI != nil {
		x, y := a.inventoryUI.Origin(w, h)
		if start(&a.inventoryUI.Position, x, y, ui.InventoryWidth, ui.InventoryHeight, 8, 215) {
			return true
		}
		if pointInside(in, x, y, ui.InventoryWidth, ui.InventoryHeight) {
			return false
		}
	}
	if a.duelUI != nil && a.state.Duel.Phase == game.DuelWager {
		x, y := a.duelUI.Origin(w, h)
		if start(&a.duelUI.Position, x, y, ui.ExchangeWidth, ui.ExchangeHeight, 48, 120) {
			return true
		}
		if pointInside(in, x, y, ui.ExchangeWidth, ui.ExchangeHeight) {
			return false
		}
	}
	if a.exchangeUI != nil && a.state.Exchange.Active {
		x, y := a.exchangeUI.Origin(w, h)
		if start(&a.exchangeUI.Position, x, y, ui.ExchangeWidth, ui.ExchangeHeight, 48, 120) {
			return true
		}
		if pointInside(in, x, y, ui.ExchangeWidth, ui.ExchangeHeight) {
			return false
		}
	}
	if e := a.state.Entities[a.shop.npcID]; e != nil && a.npcQuestUI != nil && (a.shop.questMode || e.NpcType != 1 && e.NpcType != 2 && e.NpcType != 3 && e.NpcType != 6 && e.NpcType != 11) {
		a.npcQuestUI.Detail = a.npcQuest.selected != 0
		x, y := a.npcQuestUI.Origin(w, h)
		height := 391
		if a.npcQuest.selected != 0 {
			height = 512
		}
		if start(&a.npcQuestUI.Position, x, y, ui.NPCQuestWidth, height, 75, 145) {
			return true
		}
		if pointInside(in, x, y, ui.NPCQuestWidth, height) {
			return false
		}
	}
	if e := a.state.Entities[a.shop.npcID]; e != nil && e.NpcType == 2 && !a.shop.questMode && a.gateUI != nil {
		x, y := a.gateUI.Origin(w, h)
		if start(&a.gateUI.Position, x, y, ui.GatekeeperWidth, ui.GatekeeperHeight, 265, 190) {
			return true
		}
		if pointInside(in, x, y, ui.GatekeeperWidth, ui.GatekeeperHeight) {
			return false
		}
	}
	if a.blacksmithOpen() {
		x, y := a.blacksmithUI.Origin(w, h)
		if start(&a.blacksmithUI.Position, x, y, ui.BlacksmithWidth, ui.BlacksmithHeight, 270, 190) {
			return true
		}
		if pointInside(in, x, y, ui.BlacksmithWidth, ui.BlacksmithHeight) {
			return false
		}
	}
	if e := a.state.Entities[a.shop.npcID]; e != nil && e.NpcType == 1 && !a.shop.questMode && a.marketUI != nil {
		x, y := a.marketUI.Origin(w, h)
		if start(&a.marketUI.Position, x, y, ui.MarketWidth, ui.MarketHeight, 8, 212) {
			return true
		}
		if pointInside(in, x, y, ui.MarketWidth, ui.MarketHeight) {
			return false
		}
	}
	if e := a.state.Entities[a.shop.npcID]; e != nil && e.NpcType == 11 && a.guildDirectoryUI != nil {
		x, y := a.guildDirectoryUI.Origin(w, h)
		if start(&a.guildDirectoryUI.Position, x, y, ui.GuildDirectoryWidth, ui.GuildDirectoryHeight, 268, 280) {
			return true
		}
		if pointInside(in, x, y, ui.GuildDirectoryWidth, ui.GuildDirectoryHeight) {
			return false
		}
	}
	if a.warehouseOpen() {
		x, y := a.bankUI.Origin(w, h)
		if start(&a.bankUI.Position, x, y, ui.WarehouseWidth, ui.WarehouseHeight, 75, 226) {
			return true
		}
		if pointInside(in, x, y, ui.WarehouseWidth, ui.WarehouseHeight) {
			return false
		}
	}
	xq, yq := ui.QuickPotionOrigin(a.quickPotions.Position, w, h)
	if pointInside(in, xq, yq, 94, 54) && ui.QuickPotionCell(in.MouseX, in.MouseY, xq, yq) < 0 && !a.barCovered(in, w, h) {
		a.windowDrag = windowDrag{position: &a.quickPotions.Position, offsetX: in.MouseX - xq, offsetY: in.MouseY - yq, width: 94, height: 54}
		a.wv.uiLeft = -2
		return true
	}
	if a.hud != nil {
		if a.state.Target != 0 {
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
		if start(&a.hud.Position, x, y, 213, 114, 80, 125) {
			return true
		}
	}
	if a.minimap != nil {
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
			a.wv.uiLeft = -2
			return true
		}
	}
	if a.partyUI != nil && a.state.Party.Raid && !a.party.raidHidden {
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
	if a.windowDrag.position != nil {
		a.windowDrag = windowDrag{}
		a.saveUILayout()
	}
}
func (a *App) layoutPositions() map[string]*ui.WindowPosition {
	out := map[string]*ui.WindowPosition{"quick_potions": &a.quickPotions.Position, "buffs": &a.buffLayout.Position[0], "debuffs": &a.buffLayout.Position[1]}
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
	}
	if a.subActionsUI != nil {
		out["sub_actions"] = &a.subActionsUI.Position
	}
	if a.partyUI != nil {
		out["raid"] = &a.partyUI.RaidPosition
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
	if a.gateUI != nil {
		out["gatekeeper"] = &a.gateUI.Position
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
	values := map[string]ui.WindowPosition{}
	if err == nil {
		err = json.Unmarshal(data, &values)
	}
	if err != nil {
		log.Printf("load UI layout: %v", err)
		return
	}
	for key, position := range a.layoutPositions() {
		if saved, ok := values[key]; ok && saved.Valid() {
			*position = saved
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
		data, err = json.Marshal(a.layoutPositions())
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
