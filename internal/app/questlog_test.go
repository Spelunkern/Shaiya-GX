package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
	"testing"
	"time"
)

func TestQuestJournalSelectionAndCancel(t *testing.T) {
	a := inputApp()
	a.questUI, _ = ui.NewQuestLog(inputTextures{})
	a.questConfirm, _ = ui.NewMessage(inputTextures{})
	a.questDataLoaded = true
	a.questDefs = []sdata.Quest{{Name: "First"}, {Name: "Second"}}
	a.state.Quests = map[uint16]proto.ActiveQuest{1: {ID: 1}, 2: {ID: 2}}
	a.worldControls(&platform.Input{Chars: []rune{'u'}}, 1024, 768)
	if !a.questLog.open {
		t.Fatal("journal did not open")
	}
	x, y := a.questUI.Origin(1024, 768)
	a.worldControls(&platform.Input{MouseX: x + 30, MouseY: y + 90, Pressed: true, MouseDown: true}, 1024, 768)
	a.worldControls(&platform.Input{MouseX: x + 30, MouseY: y + 90, Clicked: true}, 1024, 768)
	if a.questLog.selected != 2 {
		t.Fatalf("row selection=%d", a.questLog.selected)
	}
	a.worldControls(&platform.Input{MouseX: x + 420, MouseY: y + 460, Pressed: true, MouseDown: true}, 1024, 768)
	a.worldControls(&platform.Input{MouseX: x + 420, MouseY: y + 460, Clicked: true}, 1024, 768)
	if !a.questConfirm.Visible {
		t.Fatal("abandon lacked confirmation")
	}
	a.updateQuestLog(&platform.Input{Keys: []platform.Key{platform.KeyEscape}}, 1024, 768)
	if a.questConfirm.Visible || len(a.state.Quests) != 2 {
		t.Fatal("cancel changed quests")
	}
}

func TestQuestTimeHintUsesBoundedServerMinutes(t *testing.T) {
	a := inputApp()
	q := proto.ActiveQuest{ID: 12, MinutesLeft: 3}
	def := sdata.Quest{MaxMinutes: 10}
	if got := a.questTimeHint(def, q); got != "" {
		t.Fatalf("missing timestamp should not invent time: %q", got)
	}
	a.questSnapshotAt = map[uint16]time.Time{12: time.Now()}
	if got := a.questTimeHint(def, q); got != "Time remaining: under 4 min" {
		t.Fatal(got)
	}
	a.questSnapshotAt[12] = time.Now().Add(-4 * time.Minute)
	if got := a.questTimeHint(def, q); got != "Time limit reached; awaiting server." {
		t.Fatal(got)
	}
	a.questStartedAt = map[uint16]time.Time{12: time.Now().Add(-time.Minute)}
	if got := a.questTimeHint(def, q); got != "Time remaining: under 9 min" {
		t.Fatal(got)
	}
	def.MaxMinutes = 0
	if got := a.questTimeHint(def, q); got != "" {
		t.Fatalf("untimed quest received countdown %q", got)
	}
}

func TestQuestItemObjectiveUsesCurrentBagContents(t *testing.T) {
	a := inputApp()
	q := sdata.Quest{EndType: 1, FarmItems: [3]sdata.QuestItem{{Type: 27, ID: 2, Count: 5}}}
	active := proto.ActiveQuest{Counters: [3]uint8{9}}
	if got := a.questObjective(q, active); got != "Item 27/2: 9 / 5 (collected)" {
		t.Fatal(got)
	}
	a.state.InventoryReady = true
	a.state.Inventory[[2]uint8{0, 1}] = proto.InventoryItem{Type: 27, TypeID: 2, Count: 10}
	a.state.Inventory[[2]uint8{1, 1}] = proto.InventoryItem{Type: 27, TypeID: 2, Count: 2}
	a.state.Inventory[[2]uint8{2, 1}] = proto.InventoryItem{Type: 27, TypeID: 2, Count: 1}
	if got := a.questObjective(q, active); got != "Item 27/2: 3 / 5 (in bags)" {
		t.Fatal(got)
	}
	delete(a.state.Inventory, [2]uint8{1, 1})
	if got := a.questObjective(q, active); got != "Item 27/2: 1 / 5 (in bags)" {
		t.Fatal(got)
	}
}

func TestQuestObjectiveNamesHandInNPC(t *testing.T) {
	a := inputApp()
	a.npcDefs = map[sdata.NPCKey]sdata.NPC{{Type: 8, ID: 83}: {Name: "Captain"}}
	q := sdata.Quest{EndType: 0, EndNPCType: 8, EndNPCID: 83}
	if got := a.questObjective(q, proto.ActiveQuest{}); got != "Speak with Captain" {
		t.Fatal(got)
	}
	q.EndType, q.MobIDs, q.MobCounts = 2, [2]uint16{7}, [2]uint8{3}
	if got := a.questObjective(q, proto.ActiveQuest{Counters: [3]uint8{2}}); got != "Creature 7: 2 / 3\nReturn to Captain" {
		t.Fatal(got)
	}
	delete(a.npcDefs, sdata.NPCKey{Type: 8, ID: 83})
	q.EndType = 0
	if got := a.questObjective(q, proto.ActiveQuest{}); got != "Speak with NPC 8/83" {
		t.Fatal(got)
	}
}

func TestQuestTextPreservesParagraphsAndDropsFormattingTags(t *testing.T) {
	if got := questText(" First {c1}quest{/c}.\r\n\r\nSecond line.\r\n\r\n\r\n"); got != "First quest.\n\nSecond line." {
		t.Fatal(got)
	}
}

func TestQuestRewardLabelIncludesItemAndServerRewardValues(t *testing.T) {
	a := inputApp()
	a.items = map[sdata.ItemKey]*sdata.Item{{Type: 1, TypeID: 4}: {Name: "Sword"}}
	q := sdata.Quest{Mode: 1, Results: [3]sdata.QuestResult{{Experience: 50, Money: 330, ItemType: 1, ItemID: 4}}}
	if got := a.questRewardLabel(q, 0); got != "1: Sword\n100 XP, 330 gold" {
		t.Fatal(got)
	}
	q.Mode, q.Results[0].ItemID = 3, 99
	if got := a.questRewardLabel(q, 0); got != "1: Item 1/99\n50 XP, 330 gold" {
		t.Fatal(got)
	}
	q.Results[0].ItemType = 0
	if got := a.questRewardLabel(q, 0); got != "1: 50 XP, 330 gold" {
		t.Fatal(got)
	}
}

func TestQuestResultUsesConfirmedVariantAndAmounts(t *testing.T) {
	a := inputApp()
	a.items = map[sdata.ItemKey]*sdata.Item{{Type: 1, TypeID: 4}: {Name: "Sword"}}
	q := sdata.Quest{Summary: "Summary", Completion: [3]string{"First", "Second", "Third"}}
	r := &proto.QuestEndResult{Variant: 1, Experience: 200, Money: 990, Item: [4]uint8{1, 2, 1, 4}}
	if got := a.questResultText(q, r); got != "Second\n\nReceived: 200 XP, 990 gold\nSword" {
		t.Fatal(got)
	}
	r.Variant = 99
	if got := a.questResultText(q, r); got != "Summary\n\nReceived: 200 XP, 990 gold\nSword" {
		t.Fatal(got)
	}
}
