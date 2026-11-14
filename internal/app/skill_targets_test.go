package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"testing"
)

func TestTargetedSkillsRejectMissingStaleDeadAndNPC(t *testing.T) {
	a := inputApp()
	a.state.SelfID = 1
	k := proto.LearnedSkill{ID: 30, Level: 1, Number: 7}
	a.skillDefs = map[sdata.SkillKey]sdata.Skill{{ID: 30, Level: 1}: {Target: 1, Effect: 1}}
	for _, tc := range []struct {
		name   string
		target uint32
		entity *game.Entity
	}{
		{"none", 0, nil}, {"stale", 99, nil}, {"npc", 99, &game.Entity{Kind: game.KindNpc}},
		{"dead", 99, &game.Entity{Kind: game.KindMob, Dead: true}},
		{"zero health", 99, &game.Entity{Kind: game.KindMob, HealthKnown: true}},
		{"self", 1, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a.state.Target = tc.target
			delete(a.state.Entities, 99)
			if tc.entity != nil {
				a.state.Entities[99] = tc.entity
			}
			if p := a.learnedSkillPacket(k); p != nil {
				t.Fatalf("invalid target produced request %x", p)
			}
		})
	}
	a.state.Target = 99
	a.state.Entities[99] = &game.Entity{Kind: game.KindMob}
	if a.learnedSkillPacket(k) == nil {
		t.Fatal("living mob rejected")
	}
	for _, target := range []uint8{2, 4, 5, 6} {
		a.skillDefs[sdata.SkillKey{ID: 30, Level: 1}] = sdata.Skill{Target: target, Effect: 2}
		a.state.Target = 0
		if a.learnedSkillPacket(k) == nil {
			t.Fatalf("caster/area spell %d rejected", target)
		}
	}
}

func TestNPCSkillNamespaceNeverFallsBackToPlayerIDs(t *testing.T) {
	a := inputApp()
	key := sdata.SkillKey{ID: 31, Level: 1}
	a.skillDefs = map[sdata.SkillKey]sdata.Skill{key: {Name: "Player", Icon: 1001, VisualCode: 12}}
	a.npcSkillDefs = map[sdata.SkillKey]sdata.Skill{key: {Name: "Monster", Icon: 2017, VisualCode: 84}}
	for _, tc := range []struct {
		level        uint8
		name         string
		icon, visual uint16
	}{{1, "Player", 1001, 12}, {100, "Monster", 2017, 84}} {
		d, ok := a.skillDefinition(31, tc.level)
		if !ok || d.Name != tc.name || d.Icon != tc.icon || d.VisualCode != tc.visual {
			t.Fatalf("wrong namespace: %+v", d)
		}
	}
	delete(a.npcSkillDefs, key)
	if _, ok := a.skillDefinition(31, 100); ok {
		t.Fatal("NPC ID incorrectly resolved to player definition")
	}
}

func TestSkillFallbackSelfAndNearestLivingEnemy(t *testing.T) {
	a := inputApp()
	a.state.SelfID = 42
	k := proto.LearnedSkill{ID: 30, Level: 1, Number: 7}
	key := sdata.SkillKey{ID: 30, Level: 1}
	a.skillDefs = map[sdata.SkillKey]sdata.Skill{key: {Target: 1, Effect: 4}}
	if p := a.learnedSkillPacket(k); len(p) != 7 || proto.Op(p) != proto.SkillUseNoTarget || proto.NewReader(p).U8() != 7 {
		t.Fatalf("self buff request %x", p)
	}
	if got := a.resolveSkillTarget(a.skillDefs[key]); got != 42 {
		t.Fatal("friendly spell did not default to self", got)
	}
	a.skillDefs[key] = sdata.Skill{Target: 1, Effect: 1}
	a.wv.pos[0], a.wv.pos[2] = 10, 10
	a.state.Entities = map[uint32]*game.Entity{
		1: {ID: 1, Kind: game.KindNpc, X: 10, Z: 10}, 2: {ID: 2, Kind: game.KindMob, Dead: true, X: 10, Z: 10},
		3: {ID: 3, Kind: game.KindUser, X: 10, Z: 10}, 4: {ID: 4, Kind: game.KindMob, X: 20, Z: 10},
		5: {ID: 5, Kind: game.KindMob, X: 12, Z: 10}, 6: {ID: 6, Kind: game.KindMob, X: 12, Z: 10},
	}
	p := a.learnedSkillPacket(k)
	if len(p) != 7 || a.state.Target != 5 {
		t.Fatalf("nearest enemy %x target %d", p, a.state.Target)
	}
	a.state.Target = 4
	if got := a.resolveSkillTarget(a.skillDefs[key]); got != 4 {
		t.Fatal("explicit target replaced", got)
	}
	a.state.Target = 0
	if got := a.resolveSkillTarget(sdata.Skill{Target: 6, Effect: 1}); got != 0 || a.state.Target != 0 {
		t.Fatal("caster area auto-selected a victim")
	}
}
