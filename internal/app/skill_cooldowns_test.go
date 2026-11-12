package app

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

func TestCooldownRestoresAndStartsFromConfirmedUse(t *testing.T) {
	a := &App{clock: 100, state: game.New(), skillDefs: map[sdata.SkillKey]sdata.Skill{{ID: 30, Level: 1}: {ResetTime: 10}}}
	a.state.SelfID = 42
	a.state.Skills[7] = proto.LearnedSkill{ID: 30, Delay: 5}
	a.confirmedSkills(proto.CharacterSkills)
	if a.skills.cooldowns[30] != 105 {
		t.Fatal("remaining server delay not restored")
	}
	a.clock = 106
	a.state.LastSkill = proto.SkillResult{Source: 43, ID: 30, Level: 1}
	a.confirmedSkills(proto.SkillUse)
	if a.skills.cooldowns[30] != 105 {
		t.Fatal("another player started cooldown")
	}
	a.state.LastSkill.Source = 42
	a.state.LastSkill.Result = 5
	a.confirmedSkills(proto.SkillUse)
	if a.skills.cooldowns[30] != 105 {
		t.Fatal("rejected use started cooldown")
	}
	a.state.LastSkill.Result = 0
	a.confirmedSkills(proto.SkillUse)
	if a.skills.cooldowns[30] != 116 {
		t.Fatal("confirmed use did not start cooldown")
	}
	a.clock = 107
	a.confirmedSkills(proto.SkillUse)
	if a.skills.cooldowns[30] != 116 {
		t.Fatal("multi-hit extended cooldown")
	}
}

func TestBarEditingPreservesOtherBarsAndAllowsEmptyBar(t *testing.T) {
	before := []proto.QuickSlot{{Bar: 0, Slot: 2, Bag: 100, Number: 1}, {Bar: 1, Slot: 2, Bag: 100, Number: 2}}
	item := proto.QuickSlot{Bag: 1, Number: 3}
	after := replaceAction(before, 2, &item)
	if len(after) != 2 || after[0] != before[1] || after[1].Bag != 1 || before[0].Number != 1 {
		t.Fatal("replacement mutated unrelated slot")
	}
	after = replaceAction(after, 2, nil)
	if len(after) != 1 || after[0] != before[1] {
		t.Fatal("clearing affected another bar")
	}
	a := &App{state: game.New(), skillDefs: map[sdata.SkillKey]sdata.Skill{}}
	a.state.Skills[1] = proto.LearnedSkill{ID: 30, Number: 1}
	a.state.SkillBarReady = true
	if a.skillBar()[0] != nil {
		t.Fatal("server empty bar replaced by fallback")
	}
}
