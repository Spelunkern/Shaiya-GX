package app

import (
	"fmt"
	"log"
	"math"

	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

type npcActor struct {
	questEffect  *scene.Effect
	questKind    ui.MapMarkerKind
	questRefresh float64
	key          sdata.NPCKey
	actor        *scene.CreatureActor
	entity       *game.Entity
	position     remotePosition
	facing       render.V3
}

func (a *App) npcPortrait(def sdata.NPC) *render.Texture {
	portrait := a.assets.Texture(fmt.Sprintf("interface/NpcFace/%d.png", def.Model))
	if portrait == nil {
		portrait = a.assets.Texture("interface/NpcFace/55.png")
	}
	return portrait
}

func (a *App) loadNPCs() {
	if a.npcDefs != nil {
		return
	}
	a.npcDefs = map[sdata.NPCKey]sdata.NPC{}
	b, err := a.arc.ReadFile("Npc/NpcQuest.SData")
	if err == nil {
		var defs map[sdata.NPCKey]sdata.NPC
		defs, err = sdata.ParseNPCs(b)
		if err == nil {
			a.npcDefs = defs
		}
	}
	if err != nil {
		log.Printf("NpcQuest.SData: %v", err)
	}
}

func (a *App) prepareNPCs() {
	a.loadNPCs()
	if !a.questDataLoaded && a.arc != nil {
		_ = a.loadQuests()
	}
	questBudget := 2
	v := a.wv
	if v.npcs == nil {
		v.npcs = map[uint32]*npcActor{}
	}
	for id := range v.npcs {
		if e := a.state.Entities[id]; e == nil || e.Kind != game.KindNpc {
			delete(v.npcs, id)
		}
	}
	questMarkers := 0
	for _, n := range v.npcs {
		if n.questEffect != nil {
			questMarkers++
		}
	}
	for id, e := range a.state.Entities {
		if e.Kind != game.KindNpc {
			continue
		}
		key := sdata.NPCKey{Type: e.NpcType, ID: e.TypeID}
		cached := v.npcs[id]
		if cached == nil || cached.key != key || cached.entity != e {
			angle := float64(e.Dir) / 65536 * 2 * math.Pi
			cached = &npcActor{key: key, entity: e, facing: render.V3{float32(math.Sin(angle)), 0, float32(math.Cos(angle))}}
			if def, ok := a.npcDefs[key]; ok {
				var err error
				cached.actor, err = a.assets.BuildNPC(def)
				if err != nil {
					log.Printf("NPC %d (%d/%d): %v", id, key.Type, key.ID, err)
				}
			}
			v.npcs[id] = cached
		}
		if cached.actor == nil {
			continue
		}
		before := cached.position.pos
		initialized := cached.position.initialized
		pos := cached.position.updateNPC(e, a.clock, a.npcDefs[key])
		patrolling := initialized && cached.position.duration > 0 && a.clock < cached.position.start+cached.position.duration
		if (patrolling || e.Summoned) && v.world.HasTerrain() {
			pos[1] = v.world.Height(pos[0], pos[2])
			cached.position.pos = pos
		}
		delta := pos.Sub(before)
		delta[1] = 0
		if initialized && delta.Len() > .001 {
			cached.facing = delta.Norm()
		}
		cached.actor.World = render.Basis(pos, cached.facing.Scale(-1), render.V3{0, 1, 0})
		action := 6
		if initialized && delta.Len() > .001 {
			action = 0
		}
		if err := cached.actor.SetAction(action); err != nil {
			a.assets.Missing[err.Error()] = true
		}
		cached.actor.Update(float32(a.frameDT))
		hadQuestEffect := cached.questEffect != nil
		if hadQuestEffect || questMarkers < 16 {
			a.prepareNPCQuestEffect(cached, e, pos, &questBudget)
			if !hadQuestEffect && cached.questEffect != nil {
				questMarkers++
			} else if hadQuestEffect && cached.questEffect == nil {
				questMarkers--
			}
		}
	}
}

func (a *App) drawNPCs() {
	frustum := render.ViewFrustum(a.scene3d.View, a.scene3d.VisibilityProjection())
	for id, e := range a.state.Entities {
		if e.Kind != game.KindNpc {
			continue
		}
		if cached := a.wv.npcs[id]; cached != nil && cached.actor != nil {
			if cached.actor.Visible(frustum, a.scene3d.ClipPlane) {
				cached.actor.Draw(a.skinner, a.scene3d)
				if cached.questEffect != nil && !a.scene3d.Shadow {
					cached.questEffect.Draw(a.scene3d)
				}
			}
		}
	}
}

func (a *App) prepareNPCQuestEffect(n *npcActor, e *game.Entity, pos render.V3, budget *int) {
	if !a.clientSettings().SkillEffects || pos.Sub(a.wv.pos).Len() > 45 {
		n.questEffect = nil
		return
	}
	if a.clock >= n.questRefresh || n.questRefresh == 0 {
		n.questRefresh = a.clock + .25
		kind, ok := a.npcQuestMapKind(e)
		if !ok {
			n.questEffect = nil
			return
		}
		if n.questEffect != nil && n.questKind != kind {
			n.questEffect = nil
		}
		n.questKind = kind
	}
	if n.questEffect == nil || n.questEffect.Done() {
		// A missing/changed mark is checked again before any emitter is created.
		kind, ok := a.npcQuestMapKind(e)
		if !ok || *budget <= 0 {
			return
		}
		n.questKind = kind
		sequence := 0
		if kind == ui.MapQuestEnd {
			sequence = 1
		}
		if slot, ok := a.selfSlot(); ok && slot.Race >= 2 {
			sequence += 2
		}
		*budget--
		n.questEffect = a.loadSkillEffect("Effect/Quest.EFT", sequence)
		if n.questEffect == nil {
			return
		}
	}
	anchor := pos.Add(render.V3{0, 2.5, 0})
	if crown, ok := n.actor.NameAnchor(); ok {
		anchor = crown.Add(render.V3{0, .6, 0})
	}
	matrix := render.Identity()
	matrix[12], matrix[13], matrix[14] = anchor[0], anchor[1], anchor[2]
	n.questEffect.SetTransform(matrix)
	n.questEffect.Update(float32(a.frameDT))
}
