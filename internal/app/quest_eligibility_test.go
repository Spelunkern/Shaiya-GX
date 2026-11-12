package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"testing"
)

func TestQuestStartVisibilityMatchesServerConditions(t *testing.T) {
	a := inputApp()
	a.state.SelfID = 7
	a.state.Slots[0] = proto.CharacterSlot{ID: 7, Level: 10, Race: 0, Class: 0, Mode: 1}
	a.state.FinishedQuests = map[uint16]uint8{}
	e := &game.Entity{Kind: game.KindNpc, NpcType: 8, TypeID: 83}
	q := sdata.Quest{MinLevel: 3, MaxLevel: 20, Faction: 2, Mode: 1, Jobs: [6]uint8{1, 0, 0, 0, 0, 0}, StartType: 1, StartNPCType: 8, StartNPCID: 83}
	if !a.questStartVisible(q, e) {
		t.Fatal("eligible quest hidden")
	}
	for _, tc := range []struct {
		name   string
		mutate func(*sdata.Quest)
	}{
		{"level", func(q *sdata.Quest) { q.MinLevel = 11 }}, {"class", func(q *sdata.Quest) { q.Jobs[0] = 0 }},
		{"race", func(q *sdata.Quest) { q.Faction = 5 }}, {"mode", func(q *sdata.Quest) { q.Mode = 2 }},
		{"npc", func(q *sdata.Quest) { q.StartNPCID = 84 }}, {"item quest", func(q *sdata.Quest) { q.StartType = 3 }},
		{"prerequisite", func(q *sdata.Quest) { q.PrevQuest = 5 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := q
			tc.mutate(&r)
			if a.questStartVisible(r, e) {
				t.Fatal("ineligible quest visible")
			}
		})
	}
	q.PrevQuest = 5
	a.state.FinishedQuests[5] = 0
	if a.questStartVisible(q, e) {
		t.Fatal("failed prerequisite accepted")
	}
	a.state.FinishedQuests[5] = 1
	if !a.questStartVisible(q, e) {
		t.Fatal("successful prerequisite rejected")
	}
	q.StartType = 2
	q.StartItemType = 25
	q.StartItemID = 3
	a.state.InventoryReady = true
	if a.questStartVisible(q, e) {
		t.Fatal("missing start item accepted")
	}
	a.state.Inventory[[2]uint8{1, 0}] = proto.InventoryItem{Type: 25, TypeID: 3, Count: 1}
	if !a.questStartVisible(q, e) {
		t.Fatal("start item rejected")
	}
	a.state.InventoryReady = false
	delete(a.state.Inventory, [2]uint8{1, 0})
	if !a.questStartVisible(q, e) {
		t.Fatal("unknown inventory incorrectly rejected")
	}
}
