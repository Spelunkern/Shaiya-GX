package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
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
	a.wv.approachSkill = nil
	a.wv.approachNPC = false
	a.wv.approachTarget, a.wv.approachLoot = target, false
	return a.updateActionApproach(&platform.Input{})
}

func (a *App) approachLoot(id uint32) bool {
	a.wv.autoRun = false
	a.wv.approachSkill = nil
	a.wv.approachNPC = false
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
	if v.approachNPC {
		e := a.state.Entities[v.approachTarget]
		if e == nil || e.Kind != game.KindNpc || e.NpcType != 2 {
			v.approachTarget, v.travel = 0, false
			return true
		}
		goal = render.V3{e.X, e.Y, e.Z}
		reach = 7
	} else if v.approachLoot {
		item, ok := a.state.GroundItems[v.approachTarget]
		if !ok {
			v.approachTarget, v.travel = 0, false
			return true
		}
		goal = render.V3{item.X, item.Y, item.Z}
	} else {
		e := a.state.Entities[v.approachTarget]
		if e == nil || e.Dead && (len(v.approachSkill) == 0 || e.Kind == game.KindMob) || a.state.Target != v.approachTarget {
			v.approachTarget, v.travel = 0, false
			return true
		}
		goal = render.V3{e.X, e.Y, e.Z}
		reach = a.attackReach()
		if len(v.approachSkill) > 0 {
			current, ok := a.state.Skills[v.approachLearned.Number]
			if !ok || current.ID != v.approachLearned.ID || current.Level != v.approachLearned.Level || a.skills.cooldowns[current.ID] > a.clock {
				v.approachTarget, v.travel = 0, false
				return true
			}
			reach = max(2, float32(a.skillDefs[sdata.SkillKey{ID: current.ID, Level: current.Level}].AttackRange))
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
	if v.approachNPC {
		v.approachNPC = false
		a.openNPC(id)
		return a.state != nil
	}
	if loot {
		if err := a.world.Send(proto.PickupPacket(id)); err != nil {
			a.backToLogin("Unable to request item pickup.")
			return false
		}
		return true
	}
	a.state.Target = id
	if len(v.approachSkill) > 0 {
		packet := v.approachSkill
		v.approachSkill = nil
		if !a.faceRequestedTarget(id) {
			return false
		}
		if err := a.world.Send(packet); err != nil {
			a.backToLogin("Unable to cast after approaching target.")
			return false
		}
		a.requestedSkillPresentation(packet)
		return true
	}
	return a.attackTarget()
}

func (a *App) attackReach() float32 {
	reach := float32(2)
	weapon := a.state.Inventory[[2]uint8{0, 5}]
	if def := a.itemsItem(weapon.Type, weapon.TypeID); def != nil {
		reach = max(reach, float32(def.Range))
	}
	return reach
}

func (a *App) approachSkillPacket(packet []byte) bool {
	if a.wv == nil || len(packet) != 7 || proto.Op(packet) != proto.SkillUse && proto.Op(packet) != proto.SkillUseNoTarget {
		return false
	}
	r := proto.NewReader(packet)
	number, target := r.U8(), r.U32()
	// A new command replaces any queued approach, including when the new
	// spell is already in range. Otherwise the old spell could fire later.
	a.wv.approachTarget, a.wv.approachSkill = 0, nil
	a.wv.approachNPC = false
	a.wv.travel, a.wv.autoRun = false, false
	if !a.faceRequestedTarget(target) {
		return true
	}
	if target == 0 || target == a.state.SelfID {
		return false
	}
	e := a.state.Entities[target]
	if e == nil {
		return false
	}
	learned, ok := a.state.Skills[number]
	if !ok {
		return false
	}
	def := a.skillDefs[sdata.SkillKey{ID: learned.ID, Level: learned.Level}]
	delta := render.V3{e.X - a.wv.pos[0], 0, e.Z - a.wv.pos[2]}
	if delta.Len() <= max(2, float32(def.AttackRange)) {
		return false
	}
	a.wv.autoRun = false
	a.wv.approachTarget, a.wv.approachLoot = target, false
	a.wv.approachSkill, a.wv.approachLearned = packet, learned
	a.updateActionApproach(&platform.Input{})
	return true
}
