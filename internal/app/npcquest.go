package app

import (
	"fmt"

	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

type npcQuestEntry struct {
	id     uint16
	ending bool
}
type npcQuestView struct {
	npc      uint32
	selected uint16
	ending   bool
	offset   int
	result   *proto.QuestEndResult
}

// Reward values mirror Rapture's awardQuest; the result packet remains the
// authoritative record of what was actually granted.
func (a *App) questRewardLabel(q sdata.Quest, i int) string {
	r := q.Results[i]
	experience := uint64(r.Experience)
	if q.Mode <= 2 {
		experience *= 2
	}
	reward := fmt.Sprintf("%d XP, %d gold", experience, r.Money)
	if r.ItemType == 0 {
		return fmt.Sprintf("%d: %s", i+1, reward)
	}
	name := fmt.Sprintf("Item %d/%d", r.ItemType, r.ItemID)
	if a.arc != nil || a.items != nil {
		if item := a.itemsItem(r.ItemType, r.ItemID); item != nil && item.Name != "" {
			name = item.Name
		}
	}
	return fmt.Sprintf("%d: %s\n%s", i+1, name, reward)
}

func (a *App) questResultText(q sdata.Quest, result *proto.QuestEndResult) string {
	description := questText(q.Summary)
	if variant := int(result.Variant); variant < len(q.Completion) {
		if text := questText(q.Completion[variant]); text != "" {
			description = text
		}
	}
	if description != "" {
		description += "\n\n"
	}
	description += fmt.Sprintf("Received: %d XP, %d gold", result.Experience, result.Money)
	if result.Item[2] != 0 {
		name := fmt.Sprintf("Item %d/%d", result.Item[2], result.Item[3])
		if a.arc != nil || a.items != nil {
			if item := a.itemsItem(result.Item[2], result.Item[3]); item != nil && item.Name != "" {
				name = item.Name
			}
		}
		description += "\n" + name
	}
	return description
}

// questStartVisible mirrors Rapture's eligibility checks for presentation.
// Missing snapshots are treated as unknown; the server makes the decision.
func (a *App) questStartVisible(q sdata.Quest, e *game.Entity) bool {
	if q.StartType > 2 {
		return false
	}
	if (q.StartType != 0 || q.StartNPCType != 0) && (q.StartNPCType != e.NpcType || q.StartNPCID != e.TypeID) {
		return false
	}
	if slot, ok := a.selfSlot(); ok {
		if slot.Level < q.MinLevel || slot.Level > q.MaxLevel || slot.Class >= 6 || q.Jobs[slot.Class] == 0 {
			return false
		}
		allowed := true
		switch q.Faction {
		case 0, 1:
			allowed = slot.Race == q.Faction
		case 2:
			allowed = slot.Race <= 1
		case 3:
			allowed = slot.Race == 3
		case 4:
			allowed = slot.Race == 2
		case 5:
			allowed = slot.Race == 2 || slot.Race == 3
		}
		if !allowed {
			return false
		}
		switch slot.Mode {
		case 0:
			if q.Mode != 0 {
				return false
			}
		case 1:
			if q.Mode > 1 {
				return false
			}
		case 2:
			if q.Mode == 3 {
				return false
			}
		}
	}
	if a.state.FinishedQuests != nil {
		ids := [3]uint16{q.PrevQuest, q.HG, uint16(q.VG)}
		if ids != [3]uint16{} {
			met := false
			for _, id := range ids {
				if id != 0 && a.state.FinishedQuests[id] != 0 {
					met = true
					break
				}
			}
			if !met {
				return false
			}
		}
	}
	if q.StartType == 2 && a.state.InventoryReady {
		found := false
		for key, item := range a.state.Inventory {
			if key[0] >= 1 && key[0] <= 5 && item.Type == q.StartItemType && item.TypeID == q.StartItemID && item.Count > 0 {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func (a *App) serviceQuestOrigin(w, h int, e *game.Entity) (int, int) {
	if e.NpcType == 2 && a.gateUI != nil {
		x, y := a.gateUI.Origin(w, h)
		return x + 145, y + 300
	}
	if e.NpcType == 3 && a.blacksmithUI != nil {
		x, y := a.blacksmithUI.Origin(w, h)
		return x + 145, y + 300
	}
	if e.NpcType == 6 && a.bankUI != nil {
		x, y := a.bankUI.Origin(w, h)
		return x + 145, y + 75
	}
	x, y := a.marketUI.Origin(w, h)
	return x + 145, y + 75
}
func (a *App) drawServiceQuestButton(w, h int, e *game.Entity, def sdata.NPC) {
	if a.npcQuestUI == nil || len(def.StartQuests)+len(def.EndQuests) == 0 {
		return
	}
	x, y := a.serviceQuestOrigin(w, h, e)
	a.npcQuestUI.DrawServiceButton(a.sprites, a.font, x, y, false, "")
}

func (a *App) npcQuestEntries(e *game.Entity) []npcQuestEntry {
	def := a.npcDefs[sdata.NPCKey{Type: e.NpcType, ID: e.TypeID}]
	out := make([]npcQuestEntry, 0, len(def.StartQuests)+len(def.EndQuests))
	for _, id := range def.EndQuests {
		if _, ok := a.state.Quests[id]; ok {
			out = append(out, npcQuestEntry{id, true})
		}
	}
	for _, id := range def.StartQuests {
		if _, active := a.state.Quests[id]; active {
			continue
		}
		if _, finished := a.state.FinishedQuests[id]; finished {
			continue
		}
		if id > 0 && int(id) <= len(a.questDefs) && a.questStartVisible(a.questDefs[id-1], e) {
			out = append(out, npcQuestEntry{id, false})
		}
	}
	return out
}
func (a *App) updateNPCQuest(in *platform.Input, w, h int, e *game.Entity) bool {
	v := &a.npcQuest
	if v.npc != e.ID {
		*v = npcQuestView{npc: e.ID}
		a.npcQuestUI.ResetDescriptionScroll()
		if err := a.loadQuests(); err != nil {
			a.questMessage("Quest data unavailable.")
		}
	}
	entries := a.npcQuestEntries(e)
	if a.shop.questMode && v.selected == 0 {
		x, y := a.npcQuestUI.Origin(w, h)
		if a.npcQuestUI.ServiceButton(in, x+145, y+336, true) {
			a.shop.questMode = false
			return true
		}
	}
	if v.selected != 0 && v.result == nil {
		found := false
		for _, entry := range entries {
			if entry.id == v.selected && entry.ending == v.ending {
				found = true
				break
			}
		}
		if !found {
			v.selected = 0
		}
	}
	choices := 0
	if v.result == nil && v.selected != 0 && a.state.QuestChoosing && a.state.QuestChoice == v.selected && int(v.selected) <= len(a.questDefs) {
		choices = min(int(a.questDefs[v.selected-1].UserSelect), 3)
	}
	action := a.npcQuestUI.Update(in, w, h, len(entries), v.offset, v.selected != 0, v.ending, choices)
	v.offset = action.Offset
	if action.Close || in.HasKey(platform.KeyEscape) {
		if err := a.world.Send(proto.NpcTalkPacket(0)); err != nil {
			a.backToLogin("Unable to close NPC conversation.")
		} else {
			a.shop.npcID = 0
		}
		return true
	}
	if action.Row >= 0 && choices > 0 {
		if err := a.world.Send(proto.QuestChoosePacket(v.selected, uint8(action.Row))); err != nil {
			a.backToLogin("Unable to choose quest reward.")
		}
		return true
	}
	if action.Row >= 0 {
		entry := entries[action.Row]
		v.selected, v.ending = entry.id, entry.ending
		a.npcQuestUI.ResetDescriptionScroll()
	}
	if action.Back {
		v.selected = 0
		v.result = nil
		a.npcQuestUI.ResetDescriptionScroll()
	}
	if action.Accept && v.result != nil {
		v.selected, v.result = 0, nil
		a.npcQuestUI.ResetDescriptionScroll()
		return true
	}
	if action.Accept && v.selected != 0 {
		packet := proto.QuestStartPacket(e.ID, v.selected)
		if v.ending {
			packet = proto.QuestEndPacket(e.ID, v.selected)
		}
		if err := a.world.Send(packet); err != nil {
			a.backToLogin("Unable to send quest request.")
		}
		return true
	}
	x, y := a.npcQuestUI.Origin(w, h)
	height := 391
	if v.selected != 0 {
		height = 512
	}
	return panelPointer(in, x, y, ui.NPCQuestWidth, height)
}
func (a *App) drawNPCQuest(w, h int, e *game.Entity) {
	v := &a.npcQuest
	def := a.npcDefs[sdata.NPCKey{Type: e.NpcType, ID: e.TypeID}]
	entries := a.npcQuestEntries(e)
	names := make([]string, len(entries))
	for i, entry := range entries {
		prefix := "! "
		if entry.ending {
			prefix = "? "
		}
		names[i] = prefix + a.questName(entry.id)
	}
	title, description := "", ""
	var choices []string
	if v.selected > 0 && int(v.selected) <= len(a.questDefs) {
		q := a.questDefs[v.selected-1]
		title = q.Name
		description = questText(q.Summary)
		if !v.ending {
			if text := questText(q.Descriptions[0]); text != "" {
				description = text
			}
			description = fmt.Sprintf("Level %d-%d. %s", q.MinLevel, q.MaxLevel, description)
		}
		if v.result != nil {
			description = a.questResultText(q, v.result)
		} else if v.ending {
			active := a.state.Quests[v.selected]
			text := questText(q.Descriptions[2])
			if questMapReady(q, active, a.state) {
				text = questText(q.Completion[0])
			}
			if text != "" {
				description = text
			}
			description += "\n\n" + a.questObjective(q, active)
		}
		if a.state.QuestChoosing && a.state.QuestChoice == v.selected {
			for i := 0; i < min(int(q.UserSelect), 3); i++ {
				choices = append(choices, a.questRewardLabel(q, i))
			}
		}
	}
	a.npcQuestUI.Draw(a.sprites, a.font, w, h, def.Name, questText(def.Welcome), a.npcPortrait(def), names, v.offset, v.selected != 0, v.ending, title, description, choices)
	if a.shop.questMode && v.selected == 0 {
		x, y := a.npcQuestUI.Origin(w, h)
		label := "Shop"
		if e.NpcType == 6 {
			label = "Bank"
		}
		a.npcQuestUI.DrawServiceButton(a.sprites, a.font, x+145, y+336, true, label)
	}
}
