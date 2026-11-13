package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/effect"
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
)

// Native 0x57bdc0/0x57e320 create a caster/world sequence independently
// of the target's sequence. Sequence zero of IN_* is the release body;
// sequence one is the hand preparation. Passive/refused results create neither.
func skillReleaseLibrary(r proto.SkillResult, d sdata.Skill, weapon uint8) (string, int) {
	if r.Source == 0 || r.Level == 0 || r.Level == 100 || d.Effect == 0 || r.Result > 4 || r.Result == 3 && r.Target == 0 {
		return "", 0
	}
	if (d.Effect == 1 || d.Effect == 3 || d.Effect == 5) && (d.Attack == 1 || d.Attack == 2) {
		// 0x4c9110: bow, javelin and crossbow families use the generic
		// release; melee weapons select their skill-specific activation.
		switch weapon {
		case 11, 13, 14, 59, 62, 63, 64:
			return effect.CastLibrary(r.ID), 0
		}
		return effect.ActivationLibrary(r.ID), int(r.Level) - 1
	}
	return effect.CastLibrary(r.ID), 0
}

type skillRelease struct {
	instance *scene.Effect
	age      float32
}

func (a *App) startSkillReleaseEffect(r proto.SkillResult, d sdata.Skill) {
	file, sequence := skillReleaseLibrary(r, d, a.playerWeapon(r.Source))
	if file == "" || len(a.wv.releaseEffects) >= 32 {
		return
	}
	matrix, ok := a.effectTargetMatrix(r.Source)
	if !ok {
		return
	}
	instance := a.loadSkillEffect(file, sequence)
	if instance == nil {
		return
	}
	// 0x450340 captures the world transform at release. Walking away must
	// not drag the already emitted activation along with the character.
	instance.SetTransform(matrix)
	elapsed := float32(0)
	if actor := a.playerActor(r.Source); actor != nil && actor.ActionActive() {
		elapsed = actor.Time
	}
	advanceSkillEffect(instance, elapsed)
	a.wv.releaseEffects = append(a.wv.releaseEffects, skillRelease{instance, elapsed})
}

func advanceSkillEffect(instance *scene.Effect, elapsed float32) {
	// Chunk only catch-up work; keep the authored sequence offsets intact.
	for left := min(max(elapsed, 0), 2); left > 0; {
		dt := min(left, .25)
		instance.Update(dt)
		left -= dt
	}
}

func (a *App) updateSkillReleaseEffects(dt float32) {
	rows := a.wv.releaseEffects[:0]
	for _, r := range a.wv.releaseEffects {
		r.age += dt
		if r.age >= 5 {
			r.instance.Stop()
		}
		r.instance.Update(dt)
		if !r.instance.Done() {
			rows = append(rows, r)
		}
	}
	clear(a.wv.releaseEffects[len(rows):])
	a.wv.releaseEffects = rows
}
