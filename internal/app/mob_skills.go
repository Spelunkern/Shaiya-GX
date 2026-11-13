package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/effect"
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

// Native 0x57add0/0x579b80/0x57a2c0 look up level one and use impact
// sequence two for monster recipients. Wire level 100 is a marker, not rank 100.
func (a *App) startMobSkillEffects(r proto.SkillResult) {
	if a.wv == nil || r.Result > 4 {
		return
	}
	if mob := a.wv.mobs[r.Source]; mob != nil && mob.actor != nil {
		if mob.lastSkill != r.ID || a.clock-mob.skillAt > .15 || !mob.actor.Attacking() {
			mob.attackTarget = r.Target
			mob.actor.PlayOnce(mob.actor.AttackAction(r.AttackSlot))
			mob.lastSkill, mob.skillAt = r.ID, a.clock
		}
	}
	a.loadSkills()
	d, ok := a.skillDefs[sdata.SkillKey{ID: r.ID, Level: 1}]
	if !ok || d.Effect == 0 || r.Result > 1 {
		return
	}
	target := r.Target
	if d.Effect == 2 || d.Effect == 4 {
		target = r.Source
	}
	if file := effect.ImpactLibrary(r.ID); file != "" {
		a.startSkillImpact(target, file, 2)
	}
	if d.Effect != 2 && d.Effect != 3 && d.Effect != 4 {
		visual := r
		visual.Level = 1
		if file, sequence := skillSecondaryLibrary(visual, d); file != "" {
			a.startSkillImpact(target, file, sequence)
		}
	}
}
