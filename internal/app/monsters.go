package app

import (
	"log"
	"math"

	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
)

type mobActor struct {
	typeID       uint16
	entity       *game.Entity
	actor        *scene.CreatureActor
	pos          render.V3
	position     remotePosition
	attackTarget uint32
	lastSkill    uint16
	skillAt      float64
	soundSkill   uint16
	soundAt      float64
}

func (a *App) prepareMonsters() {
	effectBudget := 2
	v := a.wv
	if v.mobs == nil {
		v.mobs = map[uint32]*mobActor{}
	}
	for id := range v.mobs {
		if e := a.state.Entities[id]; e == nil || e.Kind != game.KindMob {
			delete(v.mobs, id)
		}
	}
	for id, e := range a.state.Entities {
		if e.Kind != game.KindMob {
			continue
		}
		cached := v.mobs[id]
		if cached == nil || cached.typeID != e.TypeID || cached.entity != e {
			actor, err := a.assets.BuildMonster(e.TypeID)
			if err != nil {
				log.Printf("mob %d type %d: %v", id, e.TypeID, err)
			}
			cached = &mobActor{typeID: e.TypeID, entity: e, actor: actor, pos: render.V3{e.X, e.Y, e.Z}}
			v.mobs[id] = cached
		}
		if cached.actor == nil {
			continue
		}
		newMove := cached.position.initialized && cached.position.moveSerial != e.MoveSerial
		pos := cached.position.update(e, a.clock)
		if newMove && cached.position.duration > 0 {
			ms := cached.actor.Definition.NormalTimeMS
			if e.Motion != 0 {
				ms = cached.actor.Definition.ChaseTimeMS
			}
			if ms > 0 {
				duration := float64(ms) / 1000
				if e.Motion != 0 && cached.actor.Definition.ChaseStep > 0 {
					delta := cached.position.to.Sub(cached.position.from)
					step := float64(cached.actor.Definition.ChaseStep)
					if e.MobSpeedKnown && e.MobMoveSpeed <= 4 {
						step = max(1, step+float64([]int{-4, -2, 0, 2, 4}[e.MobMoveSpeed]))
					}
					duration *= math.Hypot(float64(delta[0]), float64(delta[2])) / step
				}
				cached.position.duration = min(10, max(.05, duration))
			}
		}
		if v.world.HasTerrain() {
			pos[1] = v.world.Height(pos[0], pos[2])
		}
		fwd := pos.Sub(cached.pos)
		if cached.actor.Attacking() {
			if target, ok := a.effectPosition(cached.attackTarget); ok {
				fwd = target.Sub(pos)
			}
		}
		fwd[1] = 0
		if fwd.Len() > 0.001 {
			cached.actor.World = render.Basis(pos, fwd.Scale(-1), render.V3{0, 1, 0})
		} else {
			cached.actor.World[12], cached.actor.World[13], cached.actor.World[14] = pos[0], pos[1], pos[2]
			if cached.actor.World[15] == 0 {
				cached.actor.World = render.Basis(pos, render.V3{0, 0, -1}, render.V3{0, 1, 0})
			}
		}
		cached.pos = pos
		action := 6
		if cached.position.pos[0] != e.X || cached.position.pos[2] != e.Z {
			action = 0
			if e.Motion != 0 {
				action = 1
			}
		}
		if e.Dead {
			action = 5
		}
		if err := cached.actor.SetAction(action); err != nil {
			a.assets.Missing[err.Error()] = true
		}
		cached.actor.Update(float32(a.frameDT))
		cached.actor.PrepareEffects(a.wv.eye, float32(a.frameDT), a.clientSettings().SkillEffects, !e.Dead, &effectBudget)
	}
}

func (a *App) drawMonsters() {
	frustum := render.ViewFrustum(a.scene3d.View, a.scene3d.VisibilityProjection())
	for id, e := range a.state.Entities {
		if e.Kind != game.KindMob {
			continue
		}
		if cached := a.wv.mobs[id]; cached != nil && cached.actor != nil {
			if cached.actor.Visible(frustum, a.scene3d.ClipPlane) {
				cached.actor.Draw(a.skinner, a.scene3d)
				if a.clientSettings().SkillEffects {
					cached.actor.DrawEffects(a.scene3d)
				}
			}
		}
	}
}
