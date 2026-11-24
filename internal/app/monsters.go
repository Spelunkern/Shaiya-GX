package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/movespeed"
	"log"
	"math"

	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
)

type mobActor struct {
	ground       mobGroundCache
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
	deathAt      float64
	dead         bool
}

func (a *App) prepareMonsters() {
	effectBudget := 2
	v := a.wv
	a.updatePlayerAnimationRates()
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

		if e.Dead {
			if !cached.dead {
				cached.freezeCorpse(v.world, e, a.clock)
			}
			if err := cached.actor.SetAction(5); err != nil {
				a.assets.Missing[err.Error()] = true
			}
			cached.actor.CorpseFade = cached.updateCorpse(true, a.clock)
			a.setMobAnimationRate(e, cached.actor)
			cached.actor.Update(float32(a.frameDT))
			cached.actor.PrepareEffects(float32(a.frameDT), a.clientSettings().SkillEffects, false, &effectBudget)
			continue
		}
		if cached.dead {
			cached.position = remotePosition{}
			cached.ground = mobGroundCache{}
		}
		newMove := cached.position.initialized && cached.position.moveSerial != e.MoveSerial
		pos := cached.position.update(e, a.clock)
		if newMove && cached.position.duration > 0 && !(e.MobHeightKnown && e.MobMoveDurationMS > 0) {
			def := cached.actor.Definition
			step, ms := def.NormalStep, def.NormalTimeMS
			if e.Motion != 0 {
				step, ms = def.ChaseStep, def.ChaseTimeMS
			}
			tier := uint8(2)
			if e.MobSpeedKnown {
				tier = e.MobMoveSpeed
			}
			// 0x4312a0: constant speed from the monster's step and time, so a
			// longer hop takes proportionally longer.
			if v := movespeed.MobUnitsPerSecond(int(step), ms, tier); v > 0 {
				delta := cached.position.to.Sub(cached.position.from)
				cached.position.duration = min(10, max(.05, math.Hypot(float64(delta[0]), float64(delta[2]))/v))
			}
		}
		pos = cached.ground.resolve(v.world, pos, e.MobHeightKnown, e.PositionResetSerial)
		fwd := pos.Sub(cached.pos)
		if cached.actor.Attacking() {
			if target, ok := a.effectPosition(cached.attackTarget); ok {
				fwd = target.Sub(pos)
			}
		}
		fwd[1] = 0
		if fwd.Len() > 0.001 {
			cached.actor.SetTransform(render.Basis(pos, fwd.Scale(-1), render.V3{0, 1, 0}))
		} else {
			cached.actor.World[12], cached.actor.World[13], cached.actor.World[14] = pos[0], pos[1], pos[2]
			if cached.actor.World[15] == 0 {
				cached.actor.SetTransform(render.Basis(pos, render.V3{0, 0, -1}, render.V3{0, 1, 0}))
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
		if err := cached.actor.SetAction(action); err != nil {
			a.assets.Missing[err.Error()] = true
		}
		cached.actor.CorpseFade = cached.updateCorpse(false, a.clock)
		a.setMobAnimationRate(e, cached.actor)
		cached.actor.Update(float32(a.frameDT))
		cached.actor.PrepareEffects(float32(a.frameDT), a.clientSettings().SkillEffects, !e.Dead, &effectBudget)
	}
}

// updatePlayerAnimationRates runs with the monsters: remote characters take
// their rates from the speed indices the server broadcasts (0x051C).
func (a *App) updatePlayerAnimationRates() {
	for id, r := range a.wv.actors {
		e := a.state.Entities[id]
		if e == nil || e.Speed == nil || r.character == nil || e.Shape == nil {
			continue
		}
		base, have := a.weaponAttackTime(e.Shape.Equipment[5][0], e.Shape.Equipment[5][1])
		r.character.SetSpeedTiers(e.Speed.Move, e.Speed.Attack, base, have)
	}
}

func (a *App) drawMonsters() {
	v := a.wv
	frustum := render.ViewFrustum(a.scene3d.View, a.scene3d.VisibilityProjection())
	v.visibleMobActors = v.visibleMobActors[:0]
	v.crowdCharacters = v.crowdCharacters[:0]
	for id, e := range a.state.Entities {
		if e.Kind != game.KindMob {
			continue
		}
		if cached := v.mobs[id]; cached != nil && cached.actor != nil {
			if cached.actor.CorpseFade < 1 && cached.actor.Visible(frustum, a.scene3d.ClipPlane) {
				v.visibleMobActors = append(v.visibleMobActors, cached.actor)
				v.crowdCharacters = append(v.crowdCharacters, cached.actor.Character)
			}
		}
	}
	if a.scene3d.ClipPlane == ([4]float32{}) {
		v.anyMobVisible = len(v.visibleMobActors) > 0
	}
	v.mobBatch.DrawMin(a.skinner, a.scene3d, v.crowdCharacters, 4)
	if a.clientSettings().SkillEffects {
		for _, actor := range v.visibleMobActors {
			actor.DrawEffects(a.scene3d)
		}
	}
	clear(v.visibleMobActors)
	v.visibleMobActors = v.visibleMobActors[:0]
	clear(v.crowdCharacters)
	v.crowdCharacters = v.crowdCharacters[:0]
}

// Phoenix's deliberate alternative to native corpse sinking: hold the death
// pose for 3.5 seconds, then discard increasing pixel coverage over 1.5s.
func corpseFade(age float64) float32 { return float32(min(1, max(0, (age-3.5)/1.5))) }

func (m *mobActor) updateCorpse(dead bool, now float64) float32 {
	if dead && !m.dead {
		m.deathAt = now
	}
	m.dead = dead
	if !dead {
		return 0
	}
	return corpseFade(now - m.deathAt)
}

// Freeze the last rendered location and orientation at death. Pending movement
// and later network/ground corrections must not drag a corpse toward a target.
func (m *mobActor) freezeCorpse(w *scene.World, e *game.Entity, now float64) {
	if !m.position.initialized {
		m.pos = m.ground.resolve(w, render.V3{e.X, e.Y, e.Z}, e.MobHeightKnown, e.PositionResetSerial)
		if m.actor != nil {
			m.actor.SetTransform(render.Basis(m.pos, render.V3{0, 0, -1}, render.V3{0, 1, 0}))
		}
	}
	m.position = remotePosition{initialized: true, pos: m.pos, from: m.pos, to: m.pos, moveSerial: e.MoveSerial, resetSerial: e.PositionResetSerial, lastArrival: now}
	m.attackTarget = 0
}
