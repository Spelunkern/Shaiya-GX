package world

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
)

func TestQuestNPCCatalogAllMapAndSharedCache(t *testing.T) {
	oldDefs := npcDefs[7]
	npcDefs[7] = []npcDef{{Type: 8, TypeID: 1, StartQuests: []uint16{1}}, {Type: 8, TypeID: 2}, {Type: 8, TypeID: 3, EndQuests: []uint16{1}}}
	t.Cleanup(func() { npcDefs[7] = oldDefs })
	const mapID = 60082
	sh := shard(mapID)
	sh.mu.Lock()
	old := sh.entities
	oldCache := sh.questNPCs
	oldRev := sh.npcRevision.Load()
	sh.entities = MapEntities{}
	sh.questNPCs = questNPCCatalogCache{}
	sh.mu.Unlock()
	t.Cleanup(func() {
		sh.mu.Lock()
		sh.entities = old
		sh.questNPCs = oldCache
		sh.npcRevision.Store(oldRev)
		sh.mu.Unlock()
	})
	AddEntities(mapID, MapEntities{NPCs: []MapNpc{{ID: 1, Type: 8, TypeID: 1, PosX: 20, PosZ: 30, patrol: []NpcPatrolPoint{{X: 900, Z: 800}}}, {ID: 2, Type: 8, TypeID: 2, PosX: 50, PosZ: 50}, {ID: 3, Type: 8, TypeID: 3, PosX: 950, PosZ: 850}}})
	rev, rows := sh.questNPCSnapshot()
	if len(rows) != 2 || rows[0].X != 900 || rows[1].X != 950 {
		t.Fatal("far or patrolling quest NPC absent", rows)
	}
	// Irrelevant NPC additions must not replace the immutable shared catalog.
	AddEntities(mapID, MapEntities{NPCs: []MapNpc{{ID: 4, Type: 8, TypeID: 2}}})
	newRev, again := sh.questNPCSnapshot()
	if newRev == rev || &again[0] != &rows[0] {
		t.Fatal("irrelevant NPC addition rebuilt identical metadata")
	}
	s, packets := gmWireSession(t, db.Character{ID: 991000, MapID: mapID}, false)
	s.player = RegisterPlayer(*s.char, nil, false, 0, s.out, nil)
	t.Cleanup(s.player.Unregister)
	s.syncQuestNPCCatalog(true)
	state := game.New()
	state.MapID = mapID
	if _, err := state.Apply(<-packets); err != nil {
		t.Fatal(err)
	}
	if !state.MapQuestNPCsReady || len(state.MapQuestNPCs) != 2 || len(state.Entities) != 0 {
		t.Fatal("catalog depended on world view")
	}
	s.syncQuestNPCCatalog(false)
	select {
	case p := <-packets:
		t.Fatalf("unchanged catalog resent %x", p)
	default:
	}
}

func TestQuestNPCCatalogChunksFitNativeFrames(t *testing.T) {
	rows := make([]proto.QuestNPC, proto.QuestNPCChunkLimit+1)
	for i := range rows {
		rows[i] = proto.QuestNPC{Type: 8, TypeID: 1, X: float32(i), Z: 1}
	}
	frames := questNPCFrames(1, rows)
	if len(frames) != 2 {
		t.Fatal("catalog not chunked")
	}
	s := game.New()
	s.MapID = 1
	for i, p := range frames {
		if len(p) > 65535 {
			t.Fatal("oversized native frame")
		}
		if _, err := s.Apply(p[2:]); err != nil {
			t.Fatal(err)
		}
		if i == 0 && s.MapQuestNPCsReady {
			t.Fatal("partial chunk became visible")
		}
	}
	if len(s.MapQuestNPCs) != len(rows) {
		t.Fatal("catalog records lost")
	}
}
