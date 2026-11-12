package app

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

func TestSkillFeedbackUsesOnlyRelevantConfirmedHits(t *testing.T) {
	a := inputApp()
	a.state.SelfID = 1
	a.skillDefs = map[sdata.SkillKey]sdata.Skill{{ID: 4, Level: 1}: {Effect: 2}}
	a.state.LastSkill = proto.SkillResult{Source: 1, Target: 2, ID: 4, Level: 1, HP: 34}
	a.confirmedSkillFeedback(proto.SkillPartyMember)
	if a.combatNotice != "Healing: 34" {
		t.Fatal(a.combatNotice)
	}
	a.state.LastSkill = proto.SkillResult{Source: 4, Target: 5, ID: 4, Level: 1, HP: 34}
	a.combatNotice = ""
	a.confirmedSkillFeedback(proto.SkillMobOther)
	if a.combatNotice != "" {
		t.Fatal("unrelated combat overwrote feedback")
	}
	a.state.LastReflectedDamage = proto.ReflectedDamage{Source: 2, Target: 1, HP: 9}
	a.confirmedSkillFeedback(proto.SkillUserReflectedDamage)
	if a.combatNotice != "Reflected: 9" {
		t.Fatal(a.combatNotice)
	}
}

func TestPeriodicSkillFeedbackUsesRelevantTargetsAndSkillType(t *testing.T) {
	a := inputApp()
	a.state.SelfID, a.state.Target = 1, 2
	a.skillDefs = map[sdata.SkillKey]sdata.Skill{
		{ID: 4, Level: 1}: {Effect: 4},
		{ID: 5, Level: 1}: {Effect: 3},
	}
	a.state.LastPeriodicEffect = proto.PeriodicEffect{Target: 1, Skill: 4, Level: 1, HP: 30}
	a.confirmedSkillFeedback(proto.SkillKeep)
	if a.combatNotice != "Healing: 30" {
		t.Fatal(a.combatNotice)
	}
	a.state.LastPeriodicEffect.Skill = 5
	a.confirmedSkillFeedback(proto.SkillKeep)
	if a.combatNotice != "Damage: 30" {
		t.Fatal(a.combatNotice)
	}
	a.state.LastPeriodicEffect.Target = 2
	a.confirmedSkillFeedback(proto.MobPeriodicDamage)
	if a.combatNotice != "Damage: 30" {
		t.Fatal(a.combatNotice)
	}
	a.combatNotice = ""
	a.state.LastPeriodicEffect.Target = 3
	a.confirmedSkillFeedback(proto.MobPeriodicDamage)
	if a.combatNotice != "" {
		t.Fatal("unrelated pulse shown")
	}
	a.state.LastPeriodicEffect.Target, a.state.LastPeriodicEffect.Skill = 1, 999
	a.confirmedSkillFeedback(proto.SkillKeep)
	if a.combatNotice != "" {
		t.Fatal("unknown skill type guessed")
	}
}
