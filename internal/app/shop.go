package app

import (
	"fmt"
	"strings"

	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

type shopView struct {
	quantity  *tradeQuantity
	npcID     uint32
	page      int
	pending   bool
	deadline  float64
	dragging  bool
	questMode bool
	dragIndex uint8
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
	for _, r := range in.Chars {
		if (r == 'n' || r == 'N') && !a.state.Self.Dead {
			a.loadNPCs()
			best := float32(50 * 50)
			id := uint32(0)
			for k, e := range a.state.Entities {
				if e.Kind != game.KindNpc {
					continue
				}
				dx, dz := e.X-a.wv.pos[0], e.Z-a.wv.pos[2]
				d := dx*dx + dz*dz
				if d >= 50*50 {
					continue
				}
				if d < best || d == best && (id == 0 || k < id) {
					best, id = d, k
				}
			}
			if id != 0 {
				a.openNPC(id)
				return true
			}
		}
	}
	if v.npcID == 0 {
		return false
	}
	e := a.state.Entities[v.npcID]
	if e.NpcType == 11 && a.guildDirectoryUI != nil {
		return a.updateGuildDirectory(in, w, h, e)
	}
	if (e.NpcType == 1 || e.NpcType == 2 || e.NpcType == 3 || e.NpcType == 6) && a.npcQuestUI != nil {
		if v.questMode {
			return a.updateNPCQuest(in, w, h, e)
		}
		def := a.npcDefs[sdata.NPCKey{Type: e.NpcType, ID: e.TypeID}]
		if len(def.StartQuests)+len(def.EndQuests) > 0 && !v.pending && !a.inventory.pending {
			x, y := a.serviceQuestOrigin(w, h, e)
			if a.npcQuestUI.ServiceButton(in, x, y, false) {
				if err := a.loadQuests(); err != nil {
					a.questMessage("Quest data unavailable.")
				} else {
					v.questMode = true
					a.npcQuest = npcQuestView{npc: e.ID}
				}
				return true
			}
		}
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
			a.backToLogin("Unable to close NPC conversation.")
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
		a.backToLogin("Unable to talk to NPC.")
		return
	}
	a.shop.npcID, a.shop.page, a.shop.questMode = id, 0, false
	if e.NpcType == 2 && a.gateUI != nil {
		a.gateUI.ResetWelcomeScroll()
	}
}

func (a *App) drawShop(w, h int) {
	v := &a.shop
	e := a.state.Entities[v.npcID]
	if e == nil {
		return
	}
	if e.NpcType == 11 && a.guildDirectoryUI != nil {
		a.drawGuildDirectory(w, h)
		return
	}
	def := a.npcDefs[sdata.NPCKey{Type: e.NpcType, ID: e.TypeID}]
	if v.questMode && a.npcQuestUI != nil {
		a.drawNPCQuest(w, h, e)
		return
	}
	if e.NpcType == 2 && a.gateUI != nil {
		a.drawGatekeeper(w, h, e, def)
		a.drawServiceQuestButton(w, h, e, def)
		return
	}
	if e.NpcType == 3 && a.blacksmithUI != nil {
		a.drawBlacksmith(w, h, def.Name)
		a.drawServiceQuestButton(w, h, e, def)
		return
	}
	if e.NpcType == 6 && a.bankUI != nil {
		a.drawWarehouse(w, h, def)
		a.drawServiceQuestButton(w, h, e, def)
		return
	}
	if e.NpcType != 1 && e.NpcType != 6 && a.npcQuestUI != nil {
		a.drawNPCQuest(w, h, e)
		return
	}
	if e.NpcType == 1 {
		a.drawMarket(w, h, def)
		a.drawServiceQuestButton(w, h, e, def)
		return
	}
	a.sprites.Draw(a.white, 32, 82, 400, 460, 0, 0, 1, 1, [4]float32{0.05, 0.05, 0.05, 0.95})
	a.font.Draw(a.sprites, def.Name+"  [Esc: close]", 40, 90, white, 380)
	words := strings.Fields(def.Welcome)
	line := ""
	y := 115
	for _, word := range words {
		if a.font.Width(line+" "+word) > 380 {
			a.font.Draw(a.sprites, line, 40, y, white, 380)
			y += 16
			line = ""
		}
		if y >= 180 {
			break
		}
		if line != "" {
			line += " "
		}
		line += word
	}
	if y < 180 {
		a.font.Draw(a.sprites, line, 40, y, white, 380)
	}
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
		message = "Not enough gold."
	case 2:
		message = "Inventory is full."
	}
	if v.pending {
		message = "Waiting for server..."
	}
	a.font.Draw(a.sprites, message, x, y+ui.MarketHeight+4, white, 256)
	if cell, ok := ui.MarketHit(a.lastMouseX-x, a.lastMouseY-y); ok && !v.dragging && start+cell < len(def.Items) {
		it := def.Items[start+cell]
		if d := a.itemsItem(it.Type, it.TypeID); d != nil {
			a.drawItemDetails(d, nil, 1, min(x+ui.MarketWidth+8, max(0, w-296)), min(y+112, max(0, h-106)), w, h, fmt.Sprintf("Buy: %d gold", d.BuyPrice))
		}
	}
}

func (a *App) buyItem(index, count uint8) {
	if err := a.world.Send(proto.ShopBuyPacket(a.shop.npcID, index, count)); err != nil {
		a.backToLogin("Unable to request purchase.")
		return
	}
	a.shop.pending, a.shop.deadline = true, a.clock+10
}

func (a *App) sellItem(source [2]uint8, count uint8) {
	if err := a.world.Send(proto.ShopSellPacket(source[0], source[1], count)); err != nil {
		a.backToLogin("Unable to request sale.")
		return
	}
	a.inventory.pending, a.inventory.deadline, a.inventory.selected = true, a.clock+10, false
}
