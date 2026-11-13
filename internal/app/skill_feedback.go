package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

// Skill damage is presentation only. HP and resources follow server snapshots.
func (a *App) confirmedSkillFeedback(op proto.Opcode) {
	if a.state == nil {
		return
	}
	var source, target uint32
	var value uint16
	critical := false
	label := "Damage"
	switch op {
	case proto.SkillUse, proto.SkillUseNoTarget, proto.SkillUserDrain, proto.SkillMobDrain, proto.SkillPartyMember, proto.SkillMobOther, proto.MobSkillUse, proto.MobSkillDrain, proto.MobSkillOther:
		r := a.state.LastSkill
		if r.Result > 3 || r.Target == 0 {
			return
		}
		source, target, value = r.Source, r.Target, r.HP
		critical = r.Result == 1
		if source != a.state.SelfID && target != a.state.SelfID && target != a.state.Target {
			return
		}
		if r.HealedHP > 0 {
			a.showCombatNumber(r.Source, r.HealedHP, r.Result == 1, false, true)
		}
		if r.Result == 3 {
			label = "Miss"
		} else {
			a.loadSkills()
			if d, ok := a.skillDefs[sdata.SkillKey{ID: r.ID, Level: r.Level}]; ok && d.Effect == 2 {
				label = "Healing"
			}
		}
	case proto.SkillUserExtraDamage, proto.SkillExtraDamage:
		r := a.state.LastSkillExtra
		source, target, value = r.Source, r.Target, r.HP
	case proto.SkillUserReflectedDamage, proto.SkillMobReflectedDamage:
		r := a.state.LastReflectedDamage
		source, target, value = r.Source, r.Target, r.HP
		label = "Reflected"
	case proto.MobPeriodicDamage, proto.SkillKeep:
		r := a.state.LastPeriodicEffect
		target, value = r.Target, r.HP
		if target != a.state.SelfID && target != a.state.Target {
			return
		}
		if op == proto.SkillKeep {
			a.loadSkills()
			d, ok := a.skillDefs[sdata.SkillKey{ID: r.Skill, Level: r.Level}]
			if !ok {
				return
			}
			switch d.Effect {
			case 2, 4:
				label = "Healing"
			case 1, 3:
				label = "Damage"
			default:
				return
			}
		}
	default:
		return
	}
	if source != a.state.SelfID && target != a.state.SelfID && target != a.state.Target {
		return
	}
	if label != "Miss" && value == 0 {
		return
	}
	a.showCombatNumber(target, value, critical, label == "Miss", label == "Healing")
}
