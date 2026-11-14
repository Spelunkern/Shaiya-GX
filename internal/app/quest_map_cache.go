package app

import (
	"maps"
	"sort"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

type questMarkerCache struct {
	mapID             uint16
	inventoryRevision uint64
	inventoryReady    bool
	inventoryDeps     map[sdata.ItemKey]bool
	inventoryCounts   map[sdata.ItemKey]uint32
	valid             bool
	state             *game.State
	revision          uint64
	mapSize           float32
	markers           []ui.MapMarker
	objectives        map[uint16]bool
	nextRefresh       time.Time
	builds            uint64
}

func sortMapMarkers(markers []ui.MapMarker) {
	if len(markers) < 2 {
		return
	}
	sort.Slice(markers, func(i, j int) bool {
		if markers[i].Kind != markers[j].Kind {
			return markers[i].Kind < markers[j].Kind
		}
		if markers[i].U != markers[j].U {
			return markers[i].U < markers[j].U
		}
		return markers[i].V < markers[j].V
	})
}

func (a *App) cachedQuestMarkers(mapSize float32, now time.Time) *questMarkerCache {
	c := &a.questMapCache

	same := c.valid && c.state == a.state && c.revision == a.state.QuestMapRevision && c.mapID == a.state.MapID && c.mapSize == mapSize && c.inventoryReady == a.state.InventoryReady
	if same && c.inventoryRevision != a.state.QuestInventoryRevision {
		same = maps.Equal(c.inventoryCounts, questInventoryCounts(a.state, c.inventoryDeps))
		if same {
			c.inventoryRevision = a.state.QuestInventoryRevision
		}
	}
	if same && (c.nextRefresh.IsZero() || now.Before(c.nextRefresh)) {
		return c
	}
	c.valid, c.state, c.revision, c.mapID, c.mapSize = true, a.state, a.state.QuestMapRevision, a.state.MapID, mapSize
	c.inventoryRevision, c.inventoryReady = a.state.QuestInventoryRevision, a.state.InventoryReady
	c.inventoryDeps = make(map[sdata.ItemKey]bool)
	c.markers, c.nextRefresh = c.markers[:0], time.Time{}
	c.objectives = make(map[uint16]bool)
	for id, active := range a.state.Quests {
		if id == 0 || int(id) > len(a.questDefs) {
			continue
		}
		q := a.questDefs[id-1]
		if q.EndType == 1 {
			for _, it := range q.FarmItems {
				if it.Type != 0 {
					c.inventoryDeps[sdata.ItemKey{Type: it.Type, TypeID: it.ID}] = true
				}
			}
		}
		if started, known := a.questStartedAt[id]; known && q.MinMinutes > 0 {
			deadline := started.Add(time.Duration(q.MinMinutes) * time.Minute)
			if now.Before(deadline) && (c.nextRefresh.IsZero() || deadline.Before(c.nextRefresh)) {
				c.nextRefresh = deadline
			}
		}
		if q.EndType == 2 {
			for i, mob := range q.MobIDs {
				if mob != 0 && active.Counters[i] < q.MobCounts[i] {
					c.objectives[mob] = true
				}
			}
		}
	}
	for _, n := range a.state.MapQuestNPCs {
		def := a.npcDefs[sdata.NPCKey{Type: n.Type, ID: n.TypeID}]
		for _, id := range def.StartQuests {
			if id > 0 && int(id) <= len(a.questDefs) {
				q := a.questDefs[id-1]
				if q.StartType == 2 && q.StartItemType != 0 {
					c.inventoryDeps[sdata.ItemKey{Type: q.StartItemType, TypeID: q.StartItemID}] = true
				}
			}
		}
		e := game.Entity{Kind: game.KindNpc, NpcType: n.Type, TypeID: n.TypeID, X: n.X, Z: n.Z}
		kind, show := a.npcQuestMapKindAt(&e, now)
		if !show {
			continue
		}
		if u, v, ok := ui.MapPosition(n.X, n.Z, mapSize); ok {
			c.markers = append(c.markers, ui.MapMarker{U: u, V: v, Kind: kind})
		}
	}
	sortMapMarkers(c.markers)
	c.inventoryCounts = questInventoryCounts(a.state, c.inventoryDeps)
	c.builds++
	return c
}

func (a *App) questMapMarkers(mapSize float32, markers []ui.MapMarker) []ui.MapMarker {
	if !a.state.MapQuestNPCsReady {
		// Legacy worlds and initial loading retain the nearby hint until metadata arrives.
		markers = a.visibleQuestMapMarkers(mapSize, markers)
		sortMapMarkers(markers)
		return markers
	}
	c := a.cachedQuestMarkers(mapSize, time.Now())
	markers = append(markers, c.markers...)
	// Creature objectives and party positions remain live overlays. Their movement
	// never rescans NPC eligibility or invalidates the map-wide static markers.
	start := len(markers)
	for _, e := range a.state.Entities {
		if e.Kind != game.KindMob || e.Dead || !c.objectives[e.TypeID] {
			continue
		}
		if u, v, ok := ui.MapPosition(e.X, e.Z, mapSize); ok {
			markers = append(markers, ui.MapMarker{U: u, V: v, Kind: ui.MapQuestObjective})
		}
	}
	sortMapMarkers(markers[start:])
	return markers
}

func questInventoryCounts(s *game.State, deps map[sdata.ItemKey]bool) map[sdata.ItemKey]uint32 {
	if len(deps) == 0 {
		return nil
	}
	counts := make(map[sdata.ItemKey]uint32)
	for pos, it := range s.Inventory {
		key := sdata.ItemKey{Type: it.Type, TypeID: it.TypeID}
		if pos[0] >= 1 && pos[0] <= 5 && deps[key] {
			counts[key] += uint32(it.Count)
		}
	}
	return counts
}
