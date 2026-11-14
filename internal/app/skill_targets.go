package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"math"
)

// Validate before sending or predicting anything; server still owns hostility,
// costs, range and outcomes. Caster/party and caster-area spells need no victim.
func (a *App) validSkillTarget(d sdata.Skill, target uint32) bool {
	if d.Target == 6 {
		return true
	}
	if d.Target == 2 || d.Target == 4 || d.Target == 5 {
		return target == a.state.SelfID && !a.state.Self.Dead
	}
	if target == 0 {
		return false
	}
	if target == a.state.SelfID {
		return d.Effect != 1 && d.Effect != 3 && d.Effect != 5 && !a.state.Self.Dead
	}
	e := a.state.Entities[target]
	if e == nil {
		// Party snapshots can arrive before their in-range model/entity.
		return a.state.Party.Member(target) != nil && d.Effect != 1 && d.Effect != 3 && d.Effect != 5
	}
	if e.Kind != game.KindUser && e.Kind != game.KindMob {
		return false
	}
	corpse := d.Effect == 6 && (d.Detail == 121 || d.Detail == 122)
	if corpse {
		return e.Kind == game.KindUser && e.Dead
	}
	return !e.Dead && !(e.HealthKnown && e.HP == 0)
}

// Resolve only a missing selection. A selected target is never silently replaced.
func (a *App) resolveSkillTarget(d sdata.Skill) uint32 {
	if d.Target == 6 {
		return 0
	}
	target := a.state.Target
	if d.Target == 2 || d.Target == 4 || d.Target == 5 {
		return a.state.SelfID
	}
	if target != 0 {
		return target
	}
	if (d.Effect == 2 || d.Effect == 4) && a.validSkillTarget(d, a.state.SelfID) {
		return a.state.SelfID
	}
	if d.Effect != 1 && d.Effect != 3 && d.Effect != 5 {
		return 0
	}
	if a.wv == nil {
		return 0
	}
	best := float32(math.MaxFloat32)
	for id, e := range a.state.Entities {
		// Mobs are enemies; players require confirmed duel hostility for auto-pick.
		hostile := e.Kind == game.KindMob || e.Kind == game.KindUser && a.state.Duel.Phase == game.DuelFighting && a.state.Duel.Opponent == id
		if !hostile || !a.validSkillTarget(d, id) {
			continue
		}
		dx, dz := e.X-a.wv.pos[0], e.Z-a.wv.pos[2]
		distance := dx*dx + dz*dz
		if distance < best || distance == best && (target == 0 || id < target) {
			best, target = distance, id
		}
	}
	if target != 0 {
		if e := a.state.Entities[target]; e != nil && e.Kind == game.KindMob && a.world != nil {
			if err := a.world.Send(proto.TargetMobPacket(target)); err != nil {
				a.backToLogin("Unable to select target.")
				return 0
			}
		}
		a.state.Target = target
	}
	return target
}
