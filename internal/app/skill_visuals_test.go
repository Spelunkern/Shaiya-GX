package app

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
)

func TestSkillEffectsRouteDirectAndProjectileResults(t *testing.T) {
	for _, tc := range []struct {
		id                     uint16
		attack, effect, target uint8
		result                 uint8
		flight, impact         string
	}{
		{37, 3, 3, 3, 0, "CA_ice.EFT", "DA_target_generic.EFT"},
		{37, 3, 3, 3, 3, "CA_ice.EFT", "DA_target_generic.EFT"},
		{30, 0, 4, 2, 0, "", "DA_target_bash.EFT"},
		{31, 0, 4, 2, 0, "", "DA_target_concentration.EFT"},
		{57, 1, 1, 3, 1, "", "DA_target_generic.EFT"},
		{35, 3, 3, 3, 0, "", "DA_target_root.EFT"},
		{42, 3, 3, 3, 0, "", "DA_target_slumber.EFT"},
		{57, 1, 1, 3, 3, "", ""}, // No direct hit for a miss/rejection.
		{30, 0, 4, 2, 5, "", ""},
		{30, 0, 4, 2, 4, "", ""}, // Area announcements do not duplicate impacts.
		{38, 1, 1, 7, 0, "", ""}, // Area dispatch remains separate.
		{30, 0, 0, 0, 0, "", ""},
		{65535, 1, 1, 3, 0, "", ""},
	} {
		r := proto.SkillResult{ID: tc.id, Level: 1, Target: 42, Result: tc.result}
		d := sdata.Skill{Attack: tc.attack, Effect: tc.effect, Target: tc.target}
		flight, impact := skillEffectLibraries(r, d)
		wantFlight, wantImpact := tc.flight, tc.impact
		if wantFlight != "" {
			wantFlight = "Effect/" + wantFlight
		}
		if wantImpact != "" {
			wantImpact = "Effect/" + wantImpact
		}
		if flight != wantFlight || impact != wantImpact {
			t.Fatalf("skill %d result %d: %q/%q, want %q/%q", tc.id, tc.result, flight, impact, wantFlight, wantImpact)
		}
		for _, invalid := range []proto.SkillResult{{ID: tc.id, Target: 42}, {ID: tc.id, Level: 100, Target: 42}, {ID: tc.id, Level: 1}} {
			if f, i := skillEffectLibraries(invalid, d); f != "" || i != "" {
				t.Fatal("invalid result created a visual recipe", invalid)
			}
		}
	}
}

func TestSkillEffectPlayerAnchorKeepsActorWorldUnchanged(t *testing.T) {
	a := inputApp()
	a.state.SelfID = 1
	a.wv.pos = render.V3{10, 20, 30}
	world := render.Basis(a.wv.pos, render.V3{0, 0, 1}, render.V3{0, 1, 0})
	a.wv.self = &scene.Character{World: world}
	if m, ok := a.effectTargetMatrix(1); !ok || m[13] != world[13]-.9 || a.wv.self.World != world {
		t.Fatal("player effect offset mutated the actor", m, ok)
	}
	a.wv.self = nil
	if m, ok := a.effectTargetMatrix(1); !ok || m[13] != world[13]-.9 {
		t.Fatal("unloaded player did not keep the same effect origin", m, ok)
	}
	a.state.Entities[11] = &game.Entity{ID: 11, Kind: game.KindMob, X: 10, Y: 20, Z: 30}
	if m, ok := a.effectTargetMatrix(11); !ok || m[13] != world[13] {
		t.Fatal("mob effect incorrectly used the player offset", m, ok)
	}
}
