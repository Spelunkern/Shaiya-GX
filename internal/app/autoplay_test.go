package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"testing"
)

func TestAutoplayWaitsForRestoredSkillCooldown(t *testing.T) {
	a := inputApp()
	a.state.Phase = game.PhaseInWorld
	a.state.Target = 99
	a.state.Entities[99] = &game.Entity{ID: 99, HealthKnown: true, HP: 10}
	a.state.Skills[3] = proto.LearnedSkill{ID: 37, Level: 1, Number: 3, Delay: 1}
	a.state.QuickSlots = []proto.QuickSlot{{Slot: 0, Bag: 100, Number: 3}}
	a.skills.cooldowns = map[uint16]float64{37: 3}
	a.clock = 2
	p := &Autoplay{step: 6, CastSlot: 1}
	in := platform.Input{}
	if p.Inject(a, &in, .2) || len(in.Chars) != 0 || p.step != 6 {
		t.Fatal("autoplay cast before restored cooldown expired")
	}
	a.clock = 3
	p.Inject(a, &in, .2)
	if p.step != 7 || len(in.Chars) != 1 || in.Chars[0] != '1' {
		t.Fatal("autoplay did not cast when ready", in.Chars)
	}
}
