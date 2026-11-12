package app

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

type questLogView struct {
	open     bool
	selected uint16
	offset   int
	confirm  uint16
}

// Rapture sends whole remaining minutes, rounded down. Keep an upper bound
// rather than displaying a fabricated seconds-accurate countdown.
func (a *App) questTimeHint(def sdata.Quest, q proto.ActiveQuest) string {
	if def.MaxMinutes == 0 {
		return ""
	}
	var seconds float64
	if at, ok := a.questStartedAt[q.ID]; ok {
		seconds = float64(def.MaxMinutes)*60 - time.Since(at).Seconds()
	} else if at, ok := a.questSnapshotAt[q.ID]; ok {
		seconds = float64(uint32(q.MinutesLeft)+1)*60 - time.Since(at).Seconds()
	} else {
		return ""
	}
	if seconds <= 0 {
		return "Time limit reached; awaiting server."
	}
	return fmt.Sprintf("Time remaining: under %d min", int(math.Ceil(seconds/60)))
}

func (a *App) questIDs() []uint16 {
	ids := make([]uint16, 0, len(a.state.Quests))
	for id := range a.state.Quests {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}
func (a *App) updateQuestLog(in *platform.Input, w, h int) bool {
	if a.questUI == nil || a.questConfirm == nil {
		return false
	}
	v := &a.questLog
	if a.questConfirm.Visible {
		result := a.questConfirm.Update(in, w, h)
		if result == ui.MessageOK {
			if _, ok := a.state.Quests[v.confirm]; ok {
				if err := a.world.Send(proto.QuestAbandonPacket(v.confirm)); err != nil {
					a.backToLogin("Unable to abandon quest.")
				}
			}
		}
		return true
	}
	for _, r := range in.Chars {
		if (r == 'u' || r == 'U') && !(a.inventory.open && a.inventory.selected) {
			v.open = !v.open
			if v.open {
				if err := a.loadQuests(); err != nil {
					v.open = false
					a.questMessage("Quest data unavailable.")
				} else {
					a.inventory.open, a.skills.open, a.status.open = false, false, false
					a.skills.assign = nil
				}
			}
			return true
		}
	}
	if !v.open {
		return false
	}
	if in.HasKey(platform.KeyEscape) {
		v.open = false
		return true
	}
	ids := a.questIDs()
	previous := v.selected
	v.offset = min(v.offset, max(0, len(ids)-ui.QuestRows))
	if _, ok := a.state.Quests[v.selected]; !ok {
		v.selected = 0
		if len(ids) > 0 {
			v.selected = ids[0]
		}
	}
	row, offset, close, quit := a.questUI.Update(in, w, h, len(ids), v.offset, v.selected != 0)
	v.offset = offset
	if row >= 0 {
		v.selected = ids[row]
	}
	if previous != v.selected {
		a.questUI.ResetDescriptionScroll()
	}
	if close {
		v.open = false
		return true
	}
	if quit && v.selected != 0 {
		v.confirm = v.selected
		a.questConfirm.Open("Abandon "+a.questName(v.selected)+"?", true)
		return true
	}
	x, y := a.questUI.Origin(w, h)
	return panelPointer(in, x, y, ui.QuestWidth, ui.QuestHeight)
}
func questText(s string) string {
	var b strings.Builder
	inTag := false
	for _, r := range s {
		if r == '{' {
			inTag = true
			continue
		}
		if r == '}' && inTag {
			inTag = false
			continue
		}
		if !inTag {
			b.WriteRune(r)
		}
	}
	var lines []string
	for _, line := range strings.Split(strings.ReplaceAll(b.String(), "\r\n", "\n"), "\n") {
		line = strings.Join(strings.Fields(line), " ")
		if line != "" || len(lines) > 0 && lines[len(lines)-1] != "" {
			lines = append(lines, line)
		}
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

// Item hand-in checks current bag contents; the server's counters track
// pickups and may be larger after an item is spent or moved to storage.
func (a *App) questObjective(def sdata.Quest, q proto.ActiveQuest) string {
	var lines []string
	switch def.EndType {
	case 1:
		for i, item := range def.FarmItems {
			if item.Type == 0 || item.Count == 0 {
				continue
			}
			name := fmt.Sprintf("Item %d/%d", item.Type, item.ID)
			if a.arc != nil || a.items != nil {
				if d := a.itemsItem(item.Type, item.ID); d != nil {
					name = d.Name
				}
			}
			count := int(q.Counters[i])
			label := "collected"
			if a.state.InventoryReady {
				count = 0
				label = "in bags"
				for key, it := range a.state.Inventory {
					if key[0] >= 1 && key[0] <= 5 && it.Type == item.Type && it.TypeID == item.ID {
						count += int(it.Count)
					}
				}
			}
			lines = append(lines, fmt.Sprintf("%s: %d / %d (%s)", name, count, item.Count, label))
		}
	case 2:
		for i, id := range def.MobIDs {
			if id == 0 || def.MobCounts[i] == 0 {
				continue
			}
			name := fmt.Sprintf("Creature %d", id)
			if a.assets != nil {
				if m, err := a.assets.MonsterDefinition(id); err == nil && m.Name != "" {
					name = m.Name
				}
			}
			lines = append(lines, fmt.Sprintf("%s: %d / %d", name, q.Counters[i], def.MobCounts[i]))
		}
	case 3:
		lines = append(lines, fmt.Sprintf("Defeat players: %d / %d", q.Counters[0], def.PVPKills))
	}
	if def.EndNPCType != 0 {
		name := a.npcDefs[sdata.NPCKey{Type: def.EndNPCType, ID: def.EndNPCID}].Name
		if name == "" {
			name = fmt.Sprintf("NPC %d/%d", def.EndNPCType, def.EndNPCID)
		}
		if def.EndType == 0 {
			lines = append(lines, "Speak with "+name)
		} else {
			lines = append(lines, "Return to "+name)
		}
	}
	if len(lines) == 0 {
		return fmt.Sprintf("Progress: %d / %d / %d", q.Counters[0], q.Counters[1], q.Counters[2])
	}
	return strings.Join(lines, "\n")
}
func (a *App) drawQuestLog(w, h int) {
	if a.questUI == nil || !a.questLog.open {
		return
	}
	x, y := a.questUI.Origin(w, h)
	a.clearTooltipUnderPanel(x, y, ui.QuestWidth, ui.QuestHeight)
	ids := a.questIDs()
	names := make([]string, len(ids))
	selected := -1
	for i, id := range ids {
		names[i] = a.questName(id)
		if id == a.questLog.selected {
			selected = i
		}
	}
	var title, description, objective string
	if selected >= 0 {
		id := ids[selected]
		title = a.questName(id)
		q := a.state.Quests[id]
		objective = fmt.Sprintf("Progress: %d / %d / %d", q.Counters[0], q.Counters[1], q.Counters[2])
		if int(id) <= len(a.questDefs) {
			def := a.questDefs[id-1]
			description = questText(def.Summary)
			if description == "" {
				description = questText(def.Descriptions[0])
			}
			objective = a.questObjective(def, q)
			if hint := a.questTimeHint(def, q); hint != "" {
				objective = hint + "\n" + objective
			}
		}
	}
	a.questUI.Draw(a.sprites, a.font, w, h, names, a.questLog.offset, selected, title, description, objective)
}
