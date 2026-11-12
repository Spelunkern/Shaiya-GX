package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

func (a *App) autoAttackNearest() bool {
	target := a.state.Target
	e := a.state.Entities[target]
	if e == nil || e.Dead || e.ID == a.state.SelfID || e.Kind != game.KindMob && e.Kind != game.KindUser {
		target = 0
		best := float32(50 * 50)
		for id, mob := range a.state.Entities {
			if mob.Kind != game.KindMob || mob.Dead || mob.HealthKnown && mob.HP == 0 {
				continue
			}
			dx, dz := mob.X-a.wv.pos[0], mob.Z-a.wv.pos[2]
			if d := dx*dx + dz*dz; d < best || d == best && (target == 0 || id < target) {
				target, best = id, d
			}
		}
	}
	if target == 0 {
		return true
	}
	if target != a.state.Target && a.state.Entities[target].Kind == game.KindMob {
		if err := a.world.Send(proto.TargetMobPacket(target)); err != nil {
			a.backToLogin("Unable to select target.")
			return false
		}
	}
	a.state.Target = target
	a.wv.autoRun = false
	a.wv.approachTarget, a.wv.approachLoot = target, false
	return a.updateActionApproach(&platform.Input{})
}

func (a *App) approachLoot(id uint32) bool {
	a.wv.autoRun = false
	a.wv.approachTarget, a.wv.approachLoot = id, true
	return a.updateActionApproach(&platform.Input{})
}

// Approach is visual prediction only. The request is sent once in reach and
// server packets remain responsible for attacks, pickup and item removal.
func (a *App) updateActionApproach(in *platform.Input) bool {
	v := a.wv
	if v.approachTarget == 0 || in.SuppressAutoRun {
		return true
	}
	if in.Clicked {
		v.approachTarget = 0
		return true
	}
	if a.state.Self.Dead || in.Held('W') || in.Held('S') || in.Held('A') || in.Held('D') || in.Held('Q') || in.Held('E') || in.Space || in.HasKey(platform.KeyNumLock) || hasChar(in, 'x') {
		v.approachTarget, v.travel = 0, false
		return true
	}
	var goal render.V3
	reach := float32(1.5)
	if v.approachLoot {
		item, ok := a.state.GroundItems[v.approachTarget]
		if !ok {
			v.approachTarget, v.travel = 0, false
			return true
		}
		goal = render.V3{item.X, item.Y, item.Z}
	} else {
		e := a.state.Entities[v.approachTarget]
		if e == nil || e.Dead {
			v.approachTarget, v.travel = 0, false
			return true
		}
		goal = render.V3{e.X, e.Y, e.Z}
		reach = 2
		weapon := a.state.Inventory[[2]uint8{0, 5}]
		if def := a.itemsItem(weapon.Type, weapon.TypeID); def != nil {
			reach = max(reach, float32(def.Range))
		}
	}
	delta := goal.Sub(v.pos)
	delta[1] = 0
	if delta.Len() > reach {
		v.travel = true
		v.destination = goal.Sub(delta.Norm().Scale(reach * .9))
		return true
	}
	id, loot := v.approachTarget, v.approachLoot
	v.approachTarget, v.travel = 0, false
	if loot {
		if err := a.world.Send(proto.PickupPacket(id)); err != nil {
			a.backToLogin("Unable to request item pickup.")
			return false
		}
		return true
	}
	a.state.Target = id
	return a.attackTarget()
}
