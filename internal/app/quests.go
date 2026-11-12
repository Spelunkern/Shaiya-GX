package app

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

// Definitions are loaded on first quest interaction, not for every NPC draw.
func (a *App) loadQuests() error {
	if a.questDataLoaded {
		if a.questDefs == nil {
			return fmt.Errorf("quest definitions unavailable")
		}
		return nil
	}
	a.questDataLoaded = true
	raw, err := a.arc.ReadFile("Npc/NpcQuest.SData")
	if err != nil {
		return err
	}
	npcs, quests, err := sdata.ParseQuests(raw)
	if err != nil {
		return err
	}
	a.npcDefs, a.questDefs = npcs, quests
	return nil
}
func (a *App) questMessage(message string) {
	a.state.Chat = append(a.state.Chat, game.ChatLine{Text: message})
	if len(a.state.Chat) > 200 {
		a.state.Chat = append([]game.ChatLine(nil), a.state.Chat[len(a.state.Chat)-200:]...)
	}
}
func (a *App) questName(id uint16) string {
	if id == 0 || int(id) > len(a.questDefs) {
		return fmt.Sprintf("Quest %d", id)
	}
	return a.questDefs[id-1].Name
}

// Presentation follows confirmed server messages, including automatic progress.
func (a *App) confirmedQuest(p []byte) {
	switch proto.Op(p) {
	case proto.SelectCharacter:
		if a.state.Phase == game.PhaseLoading {
			a.questStartedAt = nil
			a.questSnapshotAt = nil
		}
	case proto.QuestList:
		if quests, err := proto.DecodeQuestList(p); err == nil {
			if a.questSnapshotAt == nil {
				a.questSnapshotAt = make(map[uint16]time.Time)
			}
			at := time.Now()
			for _, q := range quests {
				a.questSnapshotAt[q.ID] = at
			}
		}
	case proto.QuestStart:
		_, id, err := proto.DecodeQuestStart(p)
		if err == nil {
			if a.questStartedAt == nil {
				a.questStartedAt = make(map[uint16]time.Time)
			}
			a.questStartedAt[id] = time.Now()
			delete(a.questSnapshotAt, id)
			a.questMessage("Quest started: " + a.questName(id))
		}
	case proto.QuestCounter:
		id, slot, value, err := proto.DecodeQuestCounter(p)
		if err == nil && slot < 3 {
			a.questMessage(fmt.Sprintf("%s: objective %d = %d", a.questName(id), slot+1, value))
		}
	case proto.QuestChoose:
		id, err := proto.DecodeQuestChoice(p)
		if err == nil {
			a.questMessage(fmt.Sprintf("Choose a reward: /quest choose %d 0-2", id))
		}
	case proto.QuestEnd:
		result, err := proto.DecodeQuestEnd(p)
		if err == nil {
			delete(a.questStartedAt, result.ID)
			delete(a.questSnapshotAt, result.ID)
			// A timed or automatic quest can end before the quest UI has ever
			// loaded definitions. Its repeatability still determines whether
			// Rapture records it as finished.
			if !a.questDataLoaded && a.arc != nil {
				_ = a.loadQuests()
			}
			if result.ID > 0 && int(result.ID) <= len(a.questDefs) {
				if a.state.FinishedQuests == nil {
					a.state.FinishedQuests = make(map[uint16]uint8)
				}
				if a.questDefs[result.ID-1].TickReceive == 0 {
					a.state.FinishedQuests[result.ID] = result.Success
				} else {
					delete(a.state.FinishedQuests, result.ID)
				}
			}
			dialogue := result.Success != 0 && a.shop.npcID != 0 && a.npcQuest.selected == result.ID && a.npcQuest.ending
			if dialogue {
				copy := result
				a.npcQuest.result = &copy
				if a.npcQuestUI != nil {
					a.npcQuestUI.ResetDescriptionScroll()
				}
			}
			status := "ended"
			if result.Success != 0 {
				status = "completed"
			}
			a.questMessage(fmt.Sprintf("Quest %s: %s", status, a.questName(result.ID)))
			if result.Success != 0 && !dialogue {
				var def sdata.Quest
				if result.ID > 0 && int(result.ID) <= len(a.questDefs) {
					def = a.questDefs[result.ID-1]
				}
				a.questMessage(a.questResultText(def, &result))
			}
		}
	}
}

// Commands provide access while the native quest dialogue is being built.
// Requests never change quest state until the server acknowledges them.
func (a *App) questCommand(text string) bool {
	fields := strings.Fields(text)
	if len(fields) == 0 || !strings.EqualFold(fields[0], "/quest") {
		return false
	}
	if err := a.loadQuests(); err != nil {
		a.questMessage("Quest data unavailable.")
		return true
	}
	if len(fields) == 1 || fields[1] == "list" {
		ids := make([]int, 0, len(a.state.Quests))
		for id := range a.state.Quests {
			ids = append(ids, int(id))
		}
		sort.Ints(ids)
		for _, number := range ids {
			id := uint16(number)
			q := a.state.Quests[id]
			a.questMessage(fmt.Sprintf("%d: %s (%d/%d/%d)", id, a.questName(id), q.Counters[0], q.Counters[1], q.Counters[2]))
		}
		if npc := a.state.Entities[a.shop.npcID]; npc != nil {
			for _, entry := range a.npcQuestEntries(npc) {
				if entry.ending {
					a.questMessage(fmt.Sprintf("Hand in at NPC: %d: %s", entry.id, a.questName(entry.id)))
				} else {
					a.questMessage(fmt.Sprintf("NPC quest: %d: %s", entry.id, a.questName(entry.id)))
				}
			}
		}
		a.questMessage("/quest start|end|abandon ID, /quest choose ID 0-2")
		return true
	}
	if len(fields) < 3 {
		a.questMessage("Use /quest list for available commands.")
		return true
	}
	parsed, err := strconv.ParseUint(fields[2], 10, 16)
	if err != nil || parsed == 0 {
		a.questMessage("Invalid quest number.")
		return true
	}
	id := uint16(parsed)
	var packet []byte
	switch strings.ToLower(fields[1]) {
	case "start", "end":
		npc := a.state.Entities[a.shop.npcID]
		if npc == nil || npc.Kind != game.KindNpc {
			a.questMessage("Talk to the quest NPC first.")
			return true
		}
		if strings.EqualFold(fields[1], "start") {
			packet = proto.QuestStartPacket(npc.ID, id)
		} else {
			packet = proto.QuestEndPacket(npc.ID, id)
		}
	case "abandon":
		packet = proto.QuestAbandonPacket(id)
	case "choose":
		if len(fields) != 4 {
			a.questMessage("Choose a reward: /quest choose ID 0-2")
			return true
		}
		choice, e := strconv.ParseUint(fields[3], 10, 8)
		if e != nil || choice > 2 || !a.state.QuestChoosing || a.state.QuestChoice != id {
			a.questMessage("No matching reward choice is pending.")
			return true
		}
		packet = proto.QuestChoosePacket(id, uint8(choice))
	default:
		a.questMessage("Use /quest list for available commands.")
		return true
	}
	if err := a.world.Send(packet); err != nil {
		a.backToLogin("Unable to send quest request.")
	}
	return true
}
