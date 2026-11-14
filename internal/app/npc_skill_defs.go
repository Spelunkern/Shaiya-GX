package app

import (
	"log"

	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
)

func (a *App) loadNPCSkills() {
	if a.npcSkillDefs != nil {
		return
	}
	a.npcSkillDefs = map[sdata.SkillKey]sdata.Skill{}
	if a.arc == nil {
		return
	}
	raw, err := a.arc.ReadFile("Monster/NpcSkill.SData")
	if err == nil {
		a.npcSkillDefs, err = sdata.ParseSkills(raw)
	}
	if err != nil {
		log.Printf("NpcSkill.SData: %v", err)
		a.npcSkillDefs = map[sdata.SkillKey]sdata.Skill{}
	}
}

// Level 100 is the wire namespace marker, not a player skill rank.
func (a *App) skillDefinition(id uint16, level uint8) (sdata.Skill, bool) {
	if level == 100 {
		a.loadNPCSkills()
		d, ok := a.npcSkillDefs[sdata.SkillKey{ID: id, Level: 1}]
		return d, ok
	}
	if a.arc != nil {
		a.loadSkills()
	}
	d, ok := a.skillDefs[sdata.SkillKey{ID: id, Level: level}]
	return d, ok
}
