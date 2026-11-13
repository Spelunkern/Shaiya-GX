package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"testing"
)

func TestSkillFeedbackUsesOnlyRelevantConfirmedHits(t *testing.T) {
	a := inputApp()
	a.state.SelfID = 1
	a.state.Entities[2] = &game.Entity{ID: 2, Kind: game.KindUser, X: 4}
	a.skillDefs = map[sdata.SkillKey]sdata.Skill{{ID: 4, Level: 1}: {Effect: 2}}
	a.state.LastSkill = proto.SkillResult{Source: 1, Target: 2, ID: 4, Level: 1, HP: 34}
	a.confirmedSkillFeedback(proto.SkillPartyMember)
	if len(a.combatNumbers) != 1 || !a.combatNumbers[0].Healing || a.combatNumbers[0].Amount != 34 || a.combatNumbers[0].Point[0] != 4 {
		t.Fatal(a.combatNumbers)
	}
	a.state.LastSkill = proto.SkillResult{Source: 4, Target: 5, ID: 4, Level: 1, HP: 34}
	a.confirmedSkillFeedback(proto.SkillMobOther)
	if len(a.combatNumbers) != 1 {
		t.Fatal("unrelated combat added numbers")
	}
	a.state.LastReflectedDamage = proto.ReflectedDamage{Source: 2, Target: 1, HP: 9}
	a.confirmedSkillFeedback(proto.SkillUserReflectedDamage)
	if len(a.combatNumbers) != 2 || a.combatNumbers[1].Healing || a.combatNumbers[1].Amount != 9 {
		t.Fatal(a.combatNumbers)
	}
}

func TestPeriodicSkillFeedbackUsesRelevantTargetsAndSkillType(t *testing.T) {
	a := inputApp()
	a.state.SelfID, a.state.Target = 1, 2
	a.state.Entities[2] = &game.Entity{ID: 2, Kind: game.KindMob}
	a.skillDefs = map[sdata.SkillKey]sdata.Skill{{ID: 4, Level: 1}: {Effect: 4}, {ID: 5, Level: 1}: {Effect: 3}}
	a.state.LastPeriodicEffect = proto.PeriodicEffect{Target: 1, Skill: 4, Level: 1, HP: 30}
	a.confirmedSkillFeedback(proto.SkillKeep)
	if len(a.combatNumbers) != 1 || !a.combatNumbers[0].Healing {
		t.Fatal(a.combatNumbers)
	}
	a.state.LastPeriodicEffect.Skill = 5
	a.confirmedSkillFeedback(proto.SkillKeep)
	if len(a.combatNumbers) != 2 || a.combatNumbers[1].Healing {
		t.Fatal(a.combatNumbers)
	}
	a.state.LastPeriodicEffect.Target = 2
	a.confirmedSkillFeedback(proto.MobPeriodicDamage)
	if len(a.combatNumbers) != 3 || a.combatNumbers[2].Healing {
		t.Fatal(a.combatNumbers)
	}
	a.state.LastPeriodicEffect.Target = 3
	a.confirmedSkillFeedback(proto.MobPeriodicDamage)
	a.state.LastPeriodicEffect.Target, a.state.LastPeriodicEffect.Skill = 1, 999
	a.confirmedSkillFeedback(proto.SkillKeep)
	if len(a.combatNumbers) != 3 {
		t.Fatal("unrelated or unknown pulse guessed", a.combatNumbers)
	}
}
