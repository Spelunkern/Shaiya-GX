//go:build integration

package app

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/testworld"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

func TestQuestStartAndAbandonAgainstRapture(t *testing.T) {
	st, w, _, wait := testworld.Enter(t)
	arc, err := archive.Open(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	a := inputApp()
	a.state = st
	a.world = w
	a.arc = arc
	a.npcQuestUI, _ = ui.NewNPCQuest(inputTextures{})
	a.npcDefs = nil
	a.wv.pos[0], a.wv.pos[1], a.wv.pos[2] = st.Self.X, st.Self.Y, st.Self.Z
	var id uint32
	wait(func() bool {
		for k, e := range st.Entities {
			if e.Kind == game.KindNpc && e.NpcType == 8 && e.TypeID == 83 {
				id = k
				return true
			}
		}
		return false
	})
	a.worldControls(&platform.Input{Chars: []rune{'n'}}, 1024, 768)
	if a.shop.npcID != id {
		t.Fatalf("quest NPC not opened: got %d want %d", a.shop.npcID, id)
	}
	a.worldControls(&platform.Input{}, 1024, 768)
	entries := a.npcQuestEntries(st.Entities[id])
	index := -1
	for i, entry := range entries {
		if entry.id == 1 && !entry.ending {
			index = i
			break
		}
	}
	if index < 0 {
		t.Fatal("quest not offered by NPC")
	}
	a.npcQuest.offset = min(index, max(0, len(entries)-ui.NPCQuestRows))
	x, y := ui.NPCQuestOrigin(1024, 768)
	rowY := y + 146 + (index-a.npcQuest.offset)*26
	click := func(x, y int) {
		a.worldControls(&platform.Input{MouseX: x, MouseY: y, Pressed: true, MouseDown: true}, 1024, 768)
		a.worldControls(&platform.Input{MouseX: x, MouseY: y, Clicked: true}, 1024, 768)
	}
	click(x+30, rowY)
	if a.npcQuest.selected != 1 {
		t.Fatal("quest detail did not open")
	}
	controls := a.worldControls(&platform.Input{MouseX: x + 30, MouseY: y + 160, Wheel: -120}, 1024, 768)
	if controls.Wheel != 0 || a.npcQuest.selected != 1 {
		t.Fatal("quest description scrolling reached the camera or changed selection")
	}
	click(x+60, y+470)
	if _, ok := st.Quests[1]; ok {
		t.Fatal("quest added before server response")
	}
	poll := func(cond func() bool) {
		t.Helper()
		deadline := time.Now().Add(8 * time.Second)
		for !cond() {
			a.pumpWorld()
			if a.state == nil {
				t.Fatal("disconnected")
			}
			if time.Now().After(deadline) {
				t.Fatal("quest response timed out")
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
	poll(func() bool { _, ok := st.Quests[1]; return ok })
	if !a.questCommand("/quest abandon 1") {
		t.Fatal("abandon not handled")
	}
	if _, ok := st.Quests[1]; !ok {
		t.Fatal("quest removed before server response")
	}
	poll(func() bool { _, ok := st.Quests[1]; return !ok })
	if st.LastQuestEnd.ID != 1 || st.LastQuestEnd.Success != 0 {
		t.Fatalf("unexpected abandonment response: %+v", st.LastQuestEnd)
	}
	if err := w.Shutdown(10 * time.Second); err != nil {
		t.Fatal(err)
	}
	st2, _, _, wait2 := testworld.Enter(t)
	wait2(func() bool { _, ok := st2.FinishedQuests[1]; return ok })
	if _, ok := st2.Quests[1]; ok || st2.FinishedQuests[1] != 0 {
		t.Fatal("abandonment not persisted")
	}
}

func TestQuestCompleteAgainstRapture(t *testing.T) {
	st, w, _, wait := testworld.Enter(t)
	arc, err := archive.Open(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	a := inputApp()
	a.state = st
	a.world = w
	a.arc = arc
	a.npcDefs = nil
	a.npcQuestUI, _ = ui.NewNPCQuest(inputTextures{})
	a.wv.pos[0], a.wv.pos[1], a.wv.pos[2] = st.Self.X, st.Self.Y, st.Self.Z
	var npc uint32
	wait(func() bool {
		for id, e := range st.Entities {
			if e.Kind == game.KindNpc && e.NpcType == 8 && e.TypeID == 87 {
				npc = id
				return true
			}
		}
		return false
	})
	wait(func() bool { q, ok := st.Quests[1]; return ok && q.Counters[0] >= 5 })
	a.worldControls(&platform.Input{Chars: []rune{'n'}}, 1024, 768)
	if a.shop.npcID != npc {
		t.Fatalf("end NPC not opened: %d want %d", a.shop.npcID, npc)
	}
	a.worldControls(&platform.Input{}, 1024, 768)
	entries := a.npcQuestEntries(st.Entities[npc])
	index := -1
	for i, e := range entries {
		if e.id == 1 && e.ending {
			index = i
			break
		}
	}
	if index < 0 {
		t.Fatal("active quest missing from end NPC")
	}
	a.npcQuest.offset = min(index, max(0, len(entries)-ui.NPCQuestRows))
	x, y := ui.NPCQuestOrigin(1024, 768)
	click := func(x, y int) {
		a.worldControls(&platform.Input{MouseX: x, MouseY: y, Pressed: true, MouseDown: true}, 1024, 768)
		a.worldControls(&platform.Input{MouseX: x, MouseY: y, Clicked: true}, 1024, 768)
	}
	click(x+30, y+146+(index-a.npcQuest.offset)*26)
	if a.npcQuest.selected != 1 || !a.npcQuest.ending {
		t.Fatal("completion detail not selected")
	}
	controls := a.worldControls(&platform.Input{MouseX: x + 30, MouseY: y + 160, Wheel: -120}, 1024, 768)
	if controls.Wheel != 0 || a.npcQuest.selected != 1 {
		t.Fatal("completion description scrolling changed the quest or camera")
	}
	beforeMoney := st.Details.Money
	click(x+60, y+470)
	if _, ok := st.Quests[1]; !ok {
		t.Fatal("quest completed before server response")
	}
	deadline := time.Now().Add(8 * time.Second)
	for {
		a.pumpWorld()
		if a.state == nil {
			t.Fatal("disconnected")
		}
		if _, ok := st.Quests[1]; !ok {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("completion timed out")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if st.LastQuestEnd.ID != 1 || st.LastQuestEnd.Success != 1 || st.LastQuestEnd.Experience != 100 || st.LastQuestEnd.Money != 330 || st.Details.Money != beforeMoney+330 {
		t.Fatalf("incorrect reward: %+v money=%d before=%d", st.LastQuestEnd, st.Details.Money, beforeMoney)
	}
	if st.FinishedQuests[1] != 1 {
		t.Fatal("completed quest not recorded")
	}
}

func TestQuestRewardChoiceAgainstRapture(t *testing.T) {
	st, w, _, wait := testworld.Enter(t)
	arc, err := archive.Open(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	a := inputApp()
	a.state = st
	a.world = w
	a.arc = arc
	a.npcDefs = nil
	a.npcQuestUI, _ = ui.NewNPCQuest(inputTextures{})
	a.wv.pos[0], a.wv.pos[1], a.wv.pos[2] = st.Self.X, st.Self.Y, st.Self.Z
	var npc uint32
	wait(func() bool {
		for id, e := range st.Entities {
			if e.Kind == game.KindNpc && e.NpcType == 8 && e.TypeID == 53 {
				npc = id
				return true
			}
		}
		return false
	})
	wait(func() bool { q, ok := st.Quests[226]; return ok && q.Counters[0] >= 5 })
	a.worldControls(&platform.Input{Chars: []rune{'n'}}, 1024, 768)
	if a.shop.npcID != npc {
		t.Fatalf("choice NPC not opened: %d want %d", a.shop.npcID, npc)
	}
	a.worldControls(&platform.Input{}, 1024, 768)
	entries := a.npcQuestEntries(st.Entities[npc])
	index := -1
	for i, e := range entries {
		if e.id == 226 && e.ending {
			index = i
			break
		}
	}
	if index < 0 {
		t.Fatal("quest missing from end NPC")
	}
	a.npcQuest.offset = min(index, max(0, len(entries)-ui.NPCQuestRows))
	x, y := ui.NPCQuestOrigin(1024, 768)
	click := func(x, y int) {
		a.worldControls(&platform.Input{MouseX: x, MouseY: y, Pressed: true, MouseDown: true}, 1024, 768)
		a.worldControls(&platform.Input{MouseX: x, MouseY: y, Clicked: true}, 1024, 768)
	}
	poll := func(cond func() bool) {
		t.Helper()
		deadline := time.Now().Add(8 * time.Second)
		for !cond() {
			a.pumpWorld()
			if a.state == nil {
				t.Fatal("disconnected")
			}
			if time.Now().After(deadline) {
				t.Fatal("reward response timed out")
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
	click(x+30, y+146+(index-a.npcQuest.offset)*26)
	if a.npcQuest.selected != 226 || !a.npcQuest.ending {
		t.Fatal("completion detail not selected")
	}
	click(x+60, y+470)
	if st.QuestChoosing {
		t.Fatal("choice displayed before server response")
	}
	poll(func() bool { return st.QuestChoosing && st.QuestChoice == 226 })
	label := a.questRewardLabel(a.questDefs[225], 1)
	if !strings.Contains(label, "Noble Pelt Leggings") || !strings.Contains(label, "200 XP, 990 gold") {
		t.Fatalf("incorrect reward presentation: %q", label)
	}
	controls := a.worldControls(&platform.Input{MouseX: x + 30, MouseY: y + 160, Wheel: -120}, 1024, 768)
	if controls.Wheel != 0 || !st.QuestChoosing {
		t.Fatal("scrolling interfered with reward choice")
	}
	before := st.Details.Money
	click(x+30, y+355)
	if _, ok := st.Quests[226]; !ok {
		t.Fatal("quest removed before reward confirmation")
	}
	poll(func() bool { _, ok := st.Quests[226]; return !ok })
	result := st.LastQuestEnd
	if result.ID != 226 || result.Success != 1 || result.Variant != 1 || result.Experience != 200 || result.Money != 990 || result.Item[2] != 33 || result.Item[3] != 87 || st.Details.Money != before+990 {
		t.Fatalf("incorrect selected reward: %+v money=%d before=%d", result, st.Details.Money, before)
	}
	item, ok := st.Inventory[[2]uint8{result.Item[0], result.Item[1]}]
	if !ok || item.Type != 33 || item.TypeID != 87 {
		t.Fatalf("selected reward missing in inventory: %+v", item)
	}
	if st.QuestChoosing {
		t.Fatal("choice remained pending after confirmation")
	}
	if a.npcQuest.result == nil || a.npcQuest.result.Variant != 1 || a.npcQuest.selected != 226 {
		t.Fatal("confirmed reward did not retain the selected result dialogue")
	}
	click(x+60, y+470)
	if a.npcQuest.result != nil || a.npcQuest.selected != 0 {
		t.Fatal("closing the result did not return to the quest list")
	}
}

func TestMerchantQuestAgainstRapture(t *testing.T) {
	st, w, _, wait := testworld.Enter(t)
	arc, err := archive.Open(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	a := inputApp()
	a.state = st
	a.world = w
	a.arc = arc
	a.npcDefs = nil
	a.npcQuestUI, _ = ui.NewNPCQuest(inputTextures{})
	a.wv.pos[0], a.wv.pos[1], a.wv.pos[2] = st.Self.X, st.Self.Y, st.Self.Z
	var npc uint32
	wait(func() bool {
		for id, e := range st.Entities {
			if e.Kind == game.KindNpc && e.NpcType == 1 && e.TypeID == 6 {
				npc = id
				return true
			}
		}
		return false
	})
	a.worldControls(&platform.Input{Chars: []rune{'n'}}, 1024, 768)
	if a.shop.npcID != npc {
		t.Fatalf("merchant not opened: %d want %d", a.shop.npcID, npc)
	}
	click := func(x, y int) {
		a.worldControls(&platform.Input{MouseX: x, MouseY: y, Pressed: true, MouseDown: true}, 1024, 768)
		a.worldControls(&platform.Input{MouseX: x, MouseY: y, Clicked: true}, 1024, 768)
	}
	sx, sy := a.serviceQuestOrigin(1024, 768, st.Entities[npc])
	click(sx+5, sy+5)
	if !a.shop.questMode {
		t.Fatal("merchant quest button did not open conversation")
	}
	entries := a.npcQuestEntries(st.Entities[npc])
	index := -1
	for i, e := range entries {
		if e.id == 9 && !e.ending {
			index = i
			break
		}
	}
	if index < 0 {
		t.Fatal("merchant quest 9 not offered")
	}
	a.npcQuest.offset = min(index, max(0, len(entries)-ui.NPCQuestRows))
	x, y := ui.NPCQuestOrigin(1024, 768)
	click(x+30, y+146+(index-a.npcQuest.offset)*26)
	if a.npcQuest.selected != 9 {
		t.Fatal("merchant quest detail not selected")
	}
	click(x+60, y+470)
	if _, ok := st.Quests[9]; ok {
		t.Fatal("quest started before acknowledgement")
	}
	deadline := time.Now().Add(8 * time.Second)
	for {
		a.pumpWorld()
		if a.state == nil {
			t.Fatal("disconnected")
		}
		if _, ok := st.Quests[9]; ok {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("merchant quest start timed out")
		}
		time.Sleep(10 * time.Millisecond)
	}
	a.worldControls(&platform.Input{}, 1024, 768)
	click(x+150, y+340)
	if a.shop.questMode || a.shop.npcID != npc {
		t.Fatal("merchant service not restored")
	}
}

func TestQuestProgressFromRealCombatAgainstRapture(t *testing.T) {
	st, w, send, wait := testworld.Enter(t)
	arc, err := archive.Open(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	a := inputApp()
	a.state = st
	a.world = w
	a.arc = arc
	a.npcDefs = nil
	if err := a.loadQuests(); err != nil {
		t.Fatal(err)
	}
	wait(func() bool { q, ok := st.Quests[1]; return ok && q.Counters[0] == 0 })
	poll := func(cond func() bool) {
		t.Helper()
		deadline := time.Now().Add(20 * time.Second)
		for !cond() {
			a.pumpWorld()
			if a.state == nil {
				t.Fatal("disconnected")
			}
			if time.Now().After(deadline) {
				t.Fatal("quest combat response timed out")
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
	var target *game.Entity
	excluded := map[uint32]bool{}
	for target == nil {
		wait(func() bool {
			for id, e := range st.Entities {
				if e.Kind == game.KindMob && e.TypeID == 2008 && !e.Dead && !excluded[id] {
					target = e
					return true
				}
			}
			return false
		})
		send(proto.TargetMobPacket(target.ID))
		poll(func() bool { return target.HealthKnown })
		if target.Dead || target.HP <= 0 {
			excluded[target.ID] = true
			target = nil
		}
	}
	send(proto.MovePacket(0, 0, target.X+1, target.Y, target.Z))
	send(proto.MobAttackPacket(target.ID))
	poll(func() bool { return st.Self.Dead || st.Quests[1].Counters[0] > 0 })
	send(proto.StopAttackPacket())
	if st.Self.Dead {
		t.Fatal("disposable character died")
	}
	if st.Quests[1].Counters[0] != 1 {
		t.Fatalf("unexpected quest progress %+v", st.Quests[1])
	}
	if !target.Dead {
		t.Fatal("counter advanced before mob death")
	}
	found := false
	for _, line := range st.Chat {
		if line.Text == "First Mission in Teos: objective 1 = 1" {
			found = true
		}
	}
	if !found {
		t.Fatal("confirmed quest progress did not reach presentation")
	}
	if err := w.Shutdown(10 * time.Second); err != nil {
		t.Fatal(err)
	}
	st2, _, _, wait2 := testworld.Enter(t)
	wait2(func() bool { return st2.Quests[1].Counters[0] == 1 })
}

// Quest 1253 auto-completes on mob 228. N2 supplies one remaining kill on a
// disposable character positioned near this original map-2 spawn.
func TestAutomaticQuestSuccessAgainstRapture(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_QUEST_AUTO") != "1" {
		t.Skip("requires disposable automatic-quest fixture")
	}
	st, w, send, wait := testworld.Enter(t)
	arc, err := archive.Open(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	a := inputApp()
	a.state, a.world, a.arc = st, w, arc
	wait(func() bool { q, ok := st.Quests[1253]; return ok && q.Counters[0] == 14 })
	var target *game.Entity
	wait(func() bool {
		for _, e := range st.Entities {
			if e.Kind == game.KindMob && e.TypeID == 228 && !e.Dead {
				target = e
				return true
			}
		}
		return false
	})
	send(proto.MovePacket(0, 0, target.X+1, target.Y, target.Z))
	send(proto.MobAttackPacket(target.ID))
	deadline := time.Now().Add(35 * time.Second)
	for st.LastQuestEnd.ID != 1253 {
		a.pumpWorld()
		if a.state == nil || st.Self.Dead || time.Now().After(deadline) {
			t.Fatal("automatic quest combat did not complete")
		}
		time.Sleep(10 * time.Millisecond)
	}
	send(proto.StopAttackPacket())
	result := st.LastQuestEnd
	if result.Success != 1 || !a.questDataLoaded {
		t.Fatal("automatic success did not load quest data", result)
	}
	want := a.questResultText(a.questDefs[1252], &result)
	found := false
	for _, line := range st.Chat {
		found = found || line.Text == want
	}
	if !found {
		t.Fatal("confirmed automatic reward was not displayed", want)
	}
	if _, active := st.Quests[1253]; active || st.FinishedQuests[1253] != 1 {
		t.Fatal("automatic completion state was not retained")
	}
}

func TestQuestItemHandInAgainstRapture(t *testing.T) {
	st, w, _, wait := testworld.Enter(t)
	arc, err := archive.Open(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	a := inputApp()
	a.state = st
	a.world = w
	a.arc = arc
	a.npcDefs = nil
	a.npcQuestUI, _ = ui.NewNPCQuest(inputTextures{})
	a.wv.pos[0], a.wv.pos[1], a.wv.pos[2] = st.Self.X, st.Self.Y, st.Self.Z
	var npc uint32
	wait(func() bool {
		for id, e := range st.Entities {
			if e.Kind == game.KindNpc && e.NpcType == 1 && e.TypeID == 6 {
				npc = id
				return true
			}
		}
		return false
	})
	wait(func() bool { _, ok := st.Quests[9]; return ok })
	a.worldControls(&platform.Input{Chars: []rune{'n'}}, 1024, 768)
	if a.shop.npcID != npc {
		t.Fatal("quest merchant not opened")
	}
	sx, sy := a.serviceQuestOrigin(1024, 768, st.Entities[npc])
	click := func(x, y int) {
		a.worldControls(&platform.Input{MouseX: x, MouseY: y, Pressed: true, MouseDown: true}, 1024, 768)
		a.worldControls(&platform.Input{MouseX: x, MouseY: y, Clicked: true}, 1024, 768)
	}
	click(sx+5, sy+5)
	if !a.shop.questMode {
		t.Fatal("merchant quest dialogue not opened")
	}
	if got := a.questObjective(a.questDefs[8], st.Quests[9]); got != "Durable Leather: 5 / 5 (in bags)" {
		t.Fatalf("wrong item objective: %s", got)
	}
	entries := a.npcQuestEntries(st.Entities[npc])
	index := -1
	for i, e := range entries {
		if e.id == 9 && e.ending {
			index = i
			break
		}
	}
	if index < 0 {
		t.Fatal("quest hand-in not offered")
	}
	a.npcQuest.offset = min(index, max(0, len(entries)-ui.NPCQuestRows))
	x, y := ui.NPCQuestOrigin(1024, 768)
	click(x+30, y+146+(index-a.npcQuest.offset)*26)
	click(x+60, y+470)
	poll := func(cond func() bool) {
		t.Helper()
		deadline := time.Now().Add(8 * time.Second)
		for !cond() {
			a.pumpWorld()
			if a.state == nil {
				t.Fatal("disconnected")
			}
			if time.Now().After(deadline) {
				t.Fatal("item quest response timed out")
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
	poll(func() bool { return st.QuestChoosing && st.QuestChoice == 9 })
	click(x+30, y+315)
	poll(func() bool { _, ok := st.Quests[9]; return !ok })
	if st.LastQuestEnd.ID != 9 || st.LastQuestEnd.Success != 1 {
		t.Fatalf("hand-in failed: %+v", st.LastQuestEnd)
	}
	for key, item := range st.Inventory {
		if key[0] >= 1 && key[0] <= 5 && item.Type == 27 && item.TypeID == 2 {
			t.Fatal("quest items were not consumed")
		}
	}
}
