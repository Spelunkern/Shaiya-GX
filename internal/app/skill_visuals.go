package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/effect"
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
)

// Recipes follow 0x577390 and the target attachments in 0x57bdc0/0x57e320.
// Melee impacts follow the first ALT contact. Area/weapon-flight dispatch
// and additional contacts remain pending.
func skillEffectLibraries(r proto.SkillResult, d sdata.Skill) (flight, impact string) {
	if r.Level == 0 || r.Level == 100 || r.Target == 0 || d.Effect == 0 {
		return
	}
	if d.Attack == 3 && (d.Effect == 1 || d.Effect == 3 || d.Effect == 5) {
		if r.Result == 0 || r.Result == 1 || r.Result == 3 {
			flight = effect.ProjectileLibrary(r.ID)
			if flight != "" {
				return flight, effect.ImpactLibrary(r.ID)
			}
		}
	}
	if r.Result <= 1 && d.Target >= 1 && d.Target <= 3 {
		impact = effect.ImpactLibrary(r.ID)
	}
	return
}

func (a *App) startSkillResultEffects(r proto.SkillResult) {
	a.loadSkills()
	d, ok := a.skillDefs[sdata.SkillKey{ID: r.ID, Level: r.Level}]
	if !ok {
		return
	}
	flight, impact := skillEffectLibraries(r, d)
	if flight != "" {
		a.startSkillProjectile(r, flight, impact)
	} else if impact != "" {
		if d.Attack == 1 && (d.Effect == 1 || d.Effect == 3 || d.Effect == 5) {
			actor := a.playerActor(r.Source)
			slot := scene.SkillAction(d.Animation, d.Attack, d.Effect, a.playerWeapon(r.Source), a.attackVariant)
			if actor.PlayingAction(slot) && actor.ContactTime() > 0 {
				if len(a.wv.impactCues) < 32 {
					a.wv.impactCues = append(a.wv.impactCues, skillImpactCue{actionClock{actor, actor.ActionID(), actor.ContactTime()}, r.Source, r.Target, impact, int(r.Level) - 1})
				}
				return
			}
		}
		a.startSkillImpact(r.Target, impact, int(r.Level)-1)
	}
}

// Area, party and additional hits have already announced the cast; render an
// impact for each successful target without restarting the caster's animation.
func (a *App) startSkillSecondaryEffect(r proto.SkillResult) {
	if a.wv == nil || r.Level == 0 || r.Level == 100 || r.Target == 0 || r.Result > 1 {
		return
	}
	a.loadSkills()
	d, ok := a.skillDefs[sdata.SkillKey{ID: r.ID, Level: r.Level}]
	if !ok || d.Effect == 0 {
		return
	}
	if file := effect.ImpactLibrary(r.ID); file != "" {
		a.startSkillImpact(r.Target, file, int(r.Level)-1)
	}
}

type skillImpactCue struct {
	actionClock
	source, target uint32
	file           string
	sequence       int
}

func (a *App) updateSkillImpactCues() {
	kept := a.wv.impactCues[:0]
	for _, cue := range a.wv.impactCues {
		due, alive := cue.ready(a.playerActor(cue.source))
		if !alive {
			continue
		}
		if due {
			a.startSkillImpact(cue.target, cue.file, cue.sequence)
		} else {
			kept = append(kept, cue)
		}
	}
	clear(a.wv.impactCues[len(kept):])
	a.wv.impactCues = kept
}

func (a *App) startSkillImpact(target uint32, file string, sequence int) {
	if len(a.wv.impacts) >= 32 {
		return
	}
	pos, ok := a.effectPosition(target)
	if !ok {
		return
	}
	a.playSound(effect.ImpactSound(file), pos[0], pos[2])
	if x := a.loadSkillEffect(file, sequence); x != nil {
		a.wv.impacts = append(a.wv.impacts, skillImpact{target: target, instance: x})
	}
}
