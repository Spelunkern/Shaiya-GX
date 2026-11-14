package app

import (
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

func TestMapZoomPanAndReleaseDoNotReachWorld(t *testing.T) {
	a := inputApp()
	var err error
	a.minimap, err = ui.NewMinimap(inputTextures{})
	if err != nil {
		t.Fatal(err)
	}
	controls := a.worldControls(&platform.Input{Chars: []rune{'m'}, Down: "W"}, 1024, 768)
	if !a.wv.mapView.full || len(controls.Chars) != 0 || controls.Down != "W" {
		t.Fatal("map hotkey reached world or swallowed held movement")
	}
	x, y, scale := ui.FullMapRect(1024, 768)
	mx, my := x+int(272*scale), y+int(299*scale)
	controls = a.worldControls(&platform.Input{MouseX: mx, MouseY: my, Wheel: 120}, 1024, 768)
	if controls.Wheel != 0 || a.wv.mapView.zoom <= 1 {
		t.Fatal("full map wheel reached camera or failed to zoom")
	}
	a.worldControls(&platform.Input{MouseX: mx, MouseY: my, Pressed: true, MouseDown: true}, 1024, 768)
	a.worldControls(&platform.Input{MouseX: mx + 50, MouseY: my, MouseDown: true}, 1024, 768)
	if a.wv.mapView.u >= .5 {
		t.Fatal("drag did not pan source imagery")
	}
	controls = a.worldControls(&platform.Input{MouseX: 1000, MouseY: 700, Clicked: true}, 1024, 768)
	if controls.Clicked || a.wv.mapView.dragging {
		t.Fatal("map drag released onto a world target")
	}
	controls = a.worldControls(&platform.Input{Keys: []platform.Key{platform.KeyEscape}}, 1024, 768)
	if a.wv.mapView.full || controls.HasKey(platform.KeyEscape) {
		t.Fatal("map Escape reached game logout")
	}
	mx, my = ui.MinimapOrigin(1024)
	controls = a.worldControls(&platform.Input{MouseX: mx + 80, MouseY: my + 80, Wheel: 120}, 1024, 768)
	if controls.Wheel != 0 || a.wv.mapView.diameter >= 900 {
		t.Fatal("small map wheel zoomed camera instead of imagery")
	}
}

func TestQuestMapMarkersFollowProgress(t *testing.T) {
	e := &game.Entity{Kind: game.KindNpc, NpcType: 8, TypeID: 2, X: 20, Z: 30}
	a := &App{state: &game.State{Entities: map[uint32]*game.Entity{1: e}, Quests: map[uint16]proto.ActiveQuest{}, FinishedQuests: map[uint16]uint8{}},
		npcDefs:   map[sdata.NPCKey]sdata.NPC{{Type: 8, ID: 2}: {StartQuests: []uint16{1}, EndQuests: []uint16{1}}},
		questDefs: []sdata.Quest{{StartType: 1, StartNPCType: 8, StartNPCID: 2, EndType: 2, MobCounts: [2]uint8{2, 1}}}}
	marks := a.questMapMarkers(100, nil)
	if len(marks) != 1 || marks[0].Kind != ui.MapQuestStart {
		t.Fatalf("available quest marker: %+v", marks)
	}
	a.state.Quests[1] = proto.ActiveQuest{ID: 1, Counters: [3]uint8{1, 1}}
	if marks = a.questMapMarkers(100, marks[:0]); len(marks) != 0 {
		t.Fatalf("incomplete quest marked as ready: %+v", marks)
	}
	a.state.Quests[1] = proto.ActiveQuest{ID: 1, Counters: [3]uint8{2, 1}}
	if marks = a.questMapMarkers(100, marks[:0]); len(marks) != 1 || marks[0].Kind != ui.MapQuestEnd {
		t.Fatalf("completed quest marker: %+v", marks)
	}
	delete(a.state.Quests, 1)
	a.state.FinishedQuests[1] = 1
	if marks = a.questMapMarkers(100, marks[:0]); len(marks) != 0 {
		t.Fatalf("finished quest still marked: %+v", marks)
	}
}

func TestQuestMapItemReadiness(t *testing.T) {
	q := sdata.Quest{EndType: 1, FarmItems: [3]sdata.QuestItem{{Type: 10, ID: 2, Count: 3}}}
	s := &game.State{Inventory: map[[2]uint8]proto.InventoryItem{{1, 0}: {Type: 10, TypeID: 2, Count: 2}, {1, 1}: {Type: 10, TypeID: 2, Count: 1}}}
	if questMapReady(q, proto.ActiveQuest{}, s) {
		t.Fatal("unknown inventory treated as ready")
	}
	s.InventoryReady = true
	if !questMapReady(q, proto.ActiveQuest{}, s) {
		t.Fatal("items across stacks not counted")
	}
	delete(s.Inventory, [2]uint8{1, 1})
	if questMapReady(q, proto.ActiveQuest{}, s) {
		t.Fatal("missing items treated as ready")
	}
}

func TestQuestMapRespectsKnownMinimumTime(t *testing.T) {
	e := &game.Entity{Kind: game.KindNpc, NpcType: 8, TypeID: 2, X: 20, Z: 30}
	a := inputApp()
	a.npcDefs = map[sdata.NPCKey]sdata.NPC{{Type: 8, ID: 2}: {EndQuests: []uint16{1}}}
	a.questDefs = []sdata.Quest{{EndType: 2, MinMinutes: 2, MobCounts: [2]uint8{1, 0}}}
	a.state.Quests = map[uint16]proto.ActiveQuest{1: {ID: 1, Counters: [3]uint8{1}}}
	a.confirmedQuest(proto.QuestStartPacket(1, 1))
	a.clock += 600 // A delayed frame must not age a newly received packet.
	if _, ok := a.npcQuestMapKind(e); ok {
		t.Fatal("quest marked ready before its minimum time")
	}
	a.questStartedAt[1] = time.Now().Add(-119 * time.Second)
	if _, ok := a.npcQuestMapKind(e); ok {
		t.Fatal("quest marked ready before two minutes elapsed")
	}
	a.questStartedAt[1] = time.Now().Add(-120 * time.Second)
	if kind, ok := a.npcQuestMapKind(e); !ok || kind != ui.MapQuestEnd {
		t.Fatal("quest not marked ready after minimum time", kind, ok)
	}
	// A reconnect snapshot has no start timestamp for untimed quests. Preserve
	// the existing hint and leave the authoritative decision to the server.
	a.questStartedAt = nil
	if _, ok := a.npcQuestMapKind(e); !ok {
		t.Fatal("unknown start time suppressed the quest marker")
	}
}

func TestPartyMapMarkersUseOnlyCurrentVisibleAllies(t *testing.T) {
	s := &game.State{SelfID: 1, Party: game.PartyState{Members: []proto.PartyMember{{ID: 1}, {ID: 2}, {ID: 3}, {ID: 4}}}, Entities: map[uint32]*game.Entity{
		2: {ID: 2, Kind: game.KindUser, X: 25, Z: 75},
		4: {ID: 4, Kind: game.KindMob, X: 50, Z: 50},
	}}
	marks := partyMapMarkers(s, 100, nil)
	if len(marks) != 1 || marks[0] != (ui.MapMarker{U: .25, V: .25}) {
		t.Fatalf("unexpected party markers: %+v", marks)
	}
	s.Entities[2].X = 75
	if marks = partyMapMarkers(s, 100, marks[:0]); len(marks) != 1 || marks[0].U != .75 {
		t.Fatalf("marker did not follow movement: %+v", marks)
	}
	delete(s.Entities, 2)
	if marks = partyMapMarkers(s, 100, marks[:0]); len(marks) != 0 {
		t.Fatalf("retained stale marker after leaving visibility: %+v", marks)
	}
}

func TestMinimapObjectivesHideCompletedAndDeadMobs(t *testing.T) {
	a := inputApp()
	a.questDefs = []sdata.Quest{{EndType: 2, MobIDs: [2]uint16{100, 200}, MobCounts: [2]uint8{3, 2}}}
	a.state.Quests = map[uint16]proto.ActiveQuest{1: {ID: 1, Counters: [3]uint8{3, 1}}}
	a.state.Entities = map[uint32]*game.Entity{
		1: {Kind: game.KindMob, TypeID: 100, X: 30, Z: 40},
		2: {Kind: game.KindMob, TypeID: 200, X: 50, Z: 60},
		3: {Kind: game.KindMob, TypeID: 200, X: 70, Z: 80, Dead: true},
	}
	marks := a.questMapMarkers(100, nil)
	if len(marks) != 1 || marks[0].Kind != ui.MapQuestObjective || marks[0].U != .5 {
		t.Fatal("objective markers", marks)
	}
}
