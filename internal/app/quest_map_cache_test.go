package app

import (
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

func questCatalogApp() *App {
	s := game.New()
	s.MapID = 1
	s.SelfID = 1
	s.MapQuestNPCsReady = true
	s.InventoryReady = true
	s.MapQuestNPCs = []proto.QuestNPC{{Type: 8, TypeID: 2, X: 900, Z: 850}}
	s.Quests = map[uint16]proto.ActiveQuest{}
	s.FinishedQuests = map[uint16]uint8{}
	s.Slots[0] = proto.CharacterSlot{ID: 1, Level: 10, Class: 0, Race: 0}
	return &App{state: s, npcDefs: map[sdata.NPCKey]sdata.NPC{{Type: 8, ID: 2}: {StartQuests: []uint16{1}, EndQuests: []uint16{1}}}, questDefs: []sdata.Quest{{StartType: 1, StartNPCType: 8, StartNPCID: 2, EndNPCType: 8, EndNPCID: 2, EndType: 2, MinLevel: 1, MaxLevel: 60, Jobs: [6]uint8{1}, MobIDs: [2]uint16{99}, MobCounts: [2]uint8{2}}}}
}

func TestQuestMapEntireMapWithoutVisibleNPCAndCacheHits(t *testing.T) {
	a := questCatalogApp()
	marks := a.questMapMarkers(1000, nil)
	if len(marks) != 1 || marks[0].Kind != ui.MapQuestStart || marks[0].U != .9 || len(a.state.Entities) != 0 {
		t.Fatal("out-of-range mission missing", marks)
	}
	buffer := make([]ui.MapMarker, 0, 8)
	for i := 0; i < 1000; i++ {
		a.state.Self.X = float32(i)
		a.questMapMarkers(1000, buffer[:0])
	}
	if a.questMapCache.builds != 1 {
		t.Fatal("walking/drawing rebuilt quest eligibility", a.questMapCache.builds)
	}
	a.state.Inventory[[2]uint8{1, 0}] = proto.InventoryItem{Type: 1, TypeID: 1, Count: 1}
	a.state.QuestInventoryRevision++
	a.questMapMarkers(1000, buffer[:0])
	if a.questMapCache.builds != 1 {
		t.Fatal("irrelevant loot rebuilt quest markers")
	}
	a.state.Quests[1] = proto.ActiveQuest{ID: 1, Counters: [3]uint8{1}}
	a.state.QuestMapRevision++
	if got := a.questMapMarkers(1000, buffer[:0]); len(got) != 0 {
		t.Fatal("started incomplete quest still offered", got)
	}
	a.state.Quests[1] = proto.ActiveQuest{ID: 1, Counters: [3]uint8{2}}
	a.state.QuestMapRevision++
	if got := a.questMapMarkers(1000, buffer[:0]); len(got) != 1 || got[0].Kind != ui.MapQuestEnd {
		t.Fatal("completed far mission missing", got)
	}
	if a.questMapCache.builds != 3 {
		t.Fatal("progress did not invalidate once")
	}
}

func TestQuestMapInventoryDependenciesAndLevel(t *testing.T) {
	a := questCatalogApp()
	a.questDefs[0].StartType = 2
	a.questDefs[0].StartItemType = 25
	a.questDefs[0].StartItemID = 4
	if got := a.questMapMarkers(1000, nil); len(got) != 0 {
		t.Fatal("quest offered without start item")
	}
	a.state.Inventory[[2]uint8{1, 0}] = proto.InventoryItem{Type: 1, TypeID: 1, Count: 1}
	a.state.QuestInventoryRevision++
	a.questMapMarkers(1000, nil)
	if a.questMapCache.builds != 1 {
		t.Fatal("unrelated inventory invalidated cache")
	}
	a.state.Inventory[[2]uint8{1, 1}] = proto.InventoryItem{Type: 25, TypeID: 4, Count: 2}
	a.state.QuestInventoryRevision++
	if got := a.questMapMarkers(1000, nil); len(got) != 1 {
		t.Fatal("new start item did not unlock mission")
	}
	a.state.Inventory[[2]uint8{2, 2}] = a.state.Inventory[[2]uint8{1, 1}]
	delete(a.state.Inventory, [2]uint8{1, 1})
	a.state.QuestInventoryRevision++
	a.questMapMarkers(1000, nil)
	if a.questMapCache.builds != 2 {
		t.Fatal("moving unchanged quest-item totals rebuilt cache")
	}
	delete(a.state.Inventory, [2]uint8{2, 2})
	a.state.QuestInventoryRevision++
	if got := a.questMapMarkers(1000, nil); len(got) != 0 {
		t.Fatal("consumed start item left stale marker")
	}
	a.questDefs[0].StartType = 1
	a.questDefs[0].MinLevel = 20
	a.state.QuestMapRevision++
	if got := a.questMapMarkers(1000, nil); len(got) != 0 {
		t.Fatal("quest ignores minimum level")
	}
	slot := a.state.Slots[0]
	slot.Level = 20
	a.state.Slots[0] = slot
	a.state.QuestMapRevision++
	if got := a.questMapMarkers(1000, nil); len(got) != 1 {
		t.Fatal("level-up did not unlock map mission")
	}
}

func TestQuestMapMinimumTimeRefreshesExactlyOnce(t *testing.T) {
	a := questCatalogApp()
	now := time.Now()
	a.questDefs[0].MinMinutes = 2
	a.state.Quests[1] = proto.ActiveQuest{ID: 1, Counters: [3]uint8{2}}
	a.questStartedAt = map[uint16]time.Time{1: now}
	c := a.cachedQuestMarkers(1000, now)
	if len(c.markers) != 0 || !c.nextRefresh.Equal(now.Add(2*time.Minute)) {
		t.Fatal("missing precise timer invalidation")
	}
	a.cachedQuestMarkers(1000, now.Add(time.Minute))
	if c.builds != 1 {
		t.Fatal("timer polled by rebuilding")
	}
	a.cachedQuestMarkers(1000, now.Add(2*time.Minute))
	if c.builds != 2 || len(c.markers) != 1 || c.markers[0].Kind != ui.MapQuestEnd || !c.nextRefresh.IsZero() {
		t.Fatal("deadline did not update once")
	}
	a.cachedQuestMarkers(1000, now.Add(3*time.Minute))
	if c.builds != 2 {
		t.Fatal("expired deadline repeatedly rebuilt")
	}
}

func TestQuestMapCacheSteadyFramesAllocateNothing(t *testing.T) {
	a := questCatalogApp()
	buffer := make([]ui.MapMarker, 0, 8)
	a.questMapMarkers(1000, buffer[:0])
	if allocations := testing.AllocsPerRun(1000, func() { a.questMapMarkers(1000, buffer[:0]) }); allocations != 0 {
		t.Fatalf("steady map markers allocate %.1f times/frame", allocations)
	}
}

func BenchmarkQuestMapCached(b *testing.B) {
	a := questCatalogApp()
	buffer := make([]ui.MapMarker, 0, 8)
	a.questMapMarkers(1000, buffer[:0])
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a.questMapMarkers(1000, buffer[:0])
	}
}
