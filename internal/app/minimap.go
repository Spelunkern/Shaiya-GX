package app

import (
	"fmt"
	"log"
	"math"
	"time"

	img "github.com/Spelunkern/Shaiya-GX/internal/data/image"
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

type mapView struct {
	image                  *render.Texture
	loaded, full, dragging bool
	diameter, zoom         float32
	u, v                   float32
	lastX, lastY           int
}

// Quest readiness is only a presentation hint; Rapture validates turn-in.
func questMapReady(q sdata.Quest, active proto.ActiveQuest, s *game.State) bool {
	switch q.EndType {
	case 0:
		return q.EndNPCType != 0
	case 1:
		if !s.InventoryReady {
			return false
		}
		var counts [3]int
		for key, item := range s.Inventory {
			if key[0] < 1 || key[0] > 5 {
				continue
			}
			for i, want := range q.FarmItems {
				if want.Type != 0 && item.Type == want.Type && item.TypeID == want.ID {
					counts[i] += int(item.Count)
					break
				}
			}
		}
		for i, want := range q.FarmItems {
			if counts[i] < int(want.Count) {
				return false
			}
		}
		return true
	case 2:
		return active.Counters[0] >= q.MobCounts[0] && active.Counters[1] >= q.MobCounts[1]
	case 3:
		return active.Counters[0] >= q.PVPKills
	default:
		return false
	}
}

func (a *App) npcQuestMapKind(e *game.Entity) (ui.MapMarkerKind, bool) {
	def := a.npcDefs[sdata.NPCKey{Type: e.NpcType, ID: e.TypeID}]
	for _, id := range def.EndQuests {
		if active, ok := a.state.Quests[id]; ok && id > 0 && int(id) <= len(a.questDefs) {
			q := a.questDefs[id-1]
			started, known := a.questStartedAt[id]
			if (!known || time.Since(started).Seconds() >= float64(q.MinMinutes)*60) && questMapReady(q, active, a.state) {
				return ui.MapQuestEnd, true
			}
		}
	}
	for _, id := range def.StartQuests {
		if _, ok := a.state.Quests[id]; ok {
			continue
		}
		if _, ok := a.state.FinishedQuests[id]; ok {
			continue
		}
		if id > 0 && int(id) <= len(a.questDefs) && a.questStartVisible(a.questDefs[id-1], e) {
			return ui.MapQuestStart, true
		}
	}
	return 0, false
}

func (a *App) questMapMarkers(mapSize float32, markers []ui.MapMarker) []ui.MapMarker {
	for _, e := range a.state.Entities {
		if e.Kind != game.KindNpc {
			continue
		}
		kind, ok := a.npcQuestMapKind(e)
		if !ok {
			continue
		}
		if u, v, ok := ui.MapPosition(e.X, e.Z, mapSize); ok {
			markers = append(markers, ui.MapMarker{U: u, V: v, Kind: kind})
		}
	}
	return markers
}

// Only visible entities provide a current position known to be on this map.
// Party roster coordinates lack a map ID and can become stale out of range.
func partyMapMarkers(s *game.State, mapSize float32, markers []ui.MapMarker) []ui.MapMarker {
	for _, member := range s.Party.Members {
		if member.ID == s.SelfID {
			continue
		}
		if e := s.Entities[member.ID]; e != nil && e.Kind == game.KindUser {
			if u, v, ok := ui.MapPosition(e.X, e.Z, mapSize); ok {
				markers = append(markers, ui.MapMarker{U: u, V: v})
			}
		}
	}
	return markers
}

func (a *App) mapState() *mapView {
	if a.wv == nil {
		return nil
	}
	v := &a.wv.mapView
	if v.diameter == 0 {
		v.diameter, v.zoom, v.u, v.v = 900, 1, .5, .5
	}
	return v
}

func (a *App) loadMapImage(v *mapView) {
	if v.loaded {
		return
	}
	v.loaded = true
	file := fmt.Sprintf("interface/MiniMap/%d.png", a.wv.mapID)
	if _, ok := a.arc.Lookup(file); !ok {
		return // Not every original map includes a minimap image.
	}
	b, err := a.arc.ReadFile(file)
	if err == nil {
		var decoded img.Texture
		decoded, err = img.DecodeResource(file, b)
		if err == nil {
			v.image = render.Upload(decoded)
		}
	}
	if err != nil {
		log.Printf("minimap %d: %v", a.wv.mapID, err)
	}
}

func (a *App) updateMinimap(in *platform.Input, w, h int) bool {
	if a.minimap == nil {
		return false
	}
	v := a.mapState()
	if v == nil {
		return false
	}
	for _, ch := range in.Chars {
		if ch == 'm' || ch == 'M' {
			v.full, v.dragging = !v.full, false
			return true
		}
	}
	if v.full && in.HasKey(platform.KeyEscape) {
		v.full, v.dragging = false, false
		return true
	}
	action := a.minimap.Update(in, w, h, v.full)
	if action.Close {
		v.full, v.dragging = false, false
		return true
	}
	if action.Toggle {
		v.full = !v.full
		return true
	}
	if v.full {
		x, y, scale := a.minimap.FullRect(w, h)
		mx, my, side := x+int(16*scale), y+int(43*scale), int(512*scale)
		over := in.MouseX >= mx && in.MouseX < mx+side && in.MouseY >= my && in.MouseY < my+side
		if in.Pressed && over {
			v.dragging, v.lastX, v.lastY = true, in.MouseX, in.MouseY
		}
		if v.dragging && in.MouseDown {
			v.u -= float32(in.MouseX-v.lastX) / float32(side) / v.zoom
			v.v -= float32(in.MouseY-v.lastY) / float32(side) / v.zoom
			v.lastX, v.lastY = in.MouseX, in.MouseY
		}
		if !in.MouseDown {
			v.dragging = false
		}
		if in.Wheel != 0 && over {
			old := ui.MapWindow(v.u, v.v, 1/v.zoom)
			fx, fy := float32(in.MouseX-mx)/float32(side), float32(in.MouseY-my)/float32(side)
			v.zoom = min(8, max(1, v.zoom*float32(math.Pow(1.2, in.Wheel/120))))
			v.u = old.U0 + fx*(old.U1-old.U0) + (.5-fx)/v.zoom
			v.v = old.V0 + fy*(old.V1-old.V0) + (.5-fy)/v.zoom
		}
		view := ui.MapWindow(v.u, v.v, 1/v.zoom)
		v.u, v.v = (view.U0+view.U1)/2, (view.V0+view.V1)/2
		return v.dragging || panelPointer(in, x, y, int(544*scale), int(579*scale))
	}
	x, y := a.minimap.Origin(w, h)
	if action.Zoom != 0 || in.Wheel != 0 && panelPointer(in, x, y, 196, 197) {
		steps := float64(action.Zoom)
		if in.Wheel != 0 {
			steps += in.Wheel / 120
		}
		v.diameter = min(2000, max(100, v.diameter*float32(math.Pow(1.2, -steps))))
	}
	return panelPointer(in, x, y, 196, 197)
}

func (a *App) drawMinimap(w, h int, full bool) {
	if a.minimap == nil || a.wv == nil || a.wv.world == nil || a.wv.world.WLD == nil {
		return
	}
	v := a.mapState()
	if full && !v.full {
		return
	}
	a.loadMapImage(v)
	if !a.questDataLoaded {
		if err := a.loadQuests(); err != nil {
			log.Printf("minimap quest data: %v", err)
		}
	}
	u, z, ok := ui.MapPosition(a.wv.pos[0], a.wv.pos[2], float32(a.wv.world.WLD.MapSize))
	if !ok {
		u, z = -1, -1
	}
	heading := float32(math.Atan2(float64(a.wv.facing[0]), float64(a.wv.facing[2])))
	var storage [32]ui.MapMarker
	markers := partyMapMarkers(a.state, float32(a.wv.world.WLD.MapSize), storage[:0])
	markers = a.questMapMarkers(float32(a.wv.world.WLD.MapSize), markers)
	if full {
		a.minimap.DrawFull(a.sprites, a.font, v.image, a.white, w, h, ui.MapWindow(v.u, v.v, 1/v.zoom), u, z, heading, a.wv.mapID, markers)
		return
	}
	span := float32(1)
	if size := a.wv.world.WLD.MapSize; size > 0 {
		span = v.diameter / float32(size)
	}
	name := []rune(ui.MapName(a.msg, a.wv.mapID))
	suffix := " - " + time.Now().Format("15:04")
	for len(name) > 1 && a.font.Width(string(name)+suffix) > 134 {
		name = name[:len(name)-1]
	}
	a.minimap.Caption = string(name) + suffix
	a.minimap.Draw(a.sprites, a.font, v.image, a.white, w, h, ui.MapWindow(u, z, span), u, z, heading, a.wv.mapID, a.wv.pos[0], a.wv.pos[2], markers)
}
