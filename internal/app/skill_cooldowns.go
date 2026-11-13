package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

// Confirmed reuse timers drive presentation and suppress duplicate requests.
// Server validation remains authoritative.
func (a *App) confirmedSkills(op proto.Opcode) {
	v := &a.skills
	if op == proto.CharacterSkills {
		v.cooldowns = map[uint16]float64{}
		v.assign = nil
		v.editing = false
		for _, k := range a.state.Skills {
			v.cooldowns[k.ID] = a.clock + float64(max(k.Delay, 0))
		}
		return
	}
	if op != proto.SkillUse && op != proto.SkillUseNoTarget && op != proto.SkillUserDrain && op != proto.SkillMobDrain {
		return
	}
	r := a.state.LastSkill
	// 0..3 are hit/critical/miss, 4 announces an area/multi-hit cast.
	// A failed targetless once-skill can also return 3, without starting reuse.
	if r.Source != a.state.SelfID || r.Result > 4 || r.Result == 3 && r.Target == 0 {
		return
	}
	if v.cooldowns[r.ID] > a.clock {
		return
	} // Multiple targets/hits share one reuse period.
	a.loadSkills()
	if v.cooldowns == nil {
		v.cooldowns = map[uint16]float64{}
	}
	v.cooldowns[r.ID] = a.clock + float64(a.skillDefs[sdata.SkillKey{ID: r.ID, Level: r.Level}].ResetTime)
}
