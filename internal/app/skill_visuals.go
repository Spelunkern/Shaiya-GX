package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/effect"
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
)

// Recipes follow 0x577390 and the target attachments in 0x57bdc0/0x57e320.
// Melee impacts follow the first ALT contact. Area announcements create one
// centre effect even if the server reports no individual recipients.
func skillEffectLibraries(r proto.SkillResult, d sdata.Skill) (flight, impact string) {
	if r.Level == 0 || r.Level == 100 || d.Effect == 0 || r.Target == 0 && d.Target != 6 {
		return
	}
	if r.Result == 4 && (d.Target == 6 || d.Target == 7) {
		return "", effect.ImpactLibrary(r.ID)
	}
	if d.Attack == 3 && (d.Effect == 1 || d.Effect == 3 || d.Effect == 5) {
		if r.Result == 0 || r.Result == 1 || r.Result == 3 {
			flight = effect.ProjectileLibrary(r.ID)
			if flight != "" {
				return flight, effect.ImpactLibrary(r.ID)
			}
		}
	}
	if r.Result <= 1 && d.Target >= 1 && d.Target <= 7 {
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
	a.startSkillReleaseEffect(r, d)
	flight, impact := skillEffectLibraries(r, d)
	if r.Target == 0 && d.Target == 6 {
		r.Target = r.Source
	}
	if flight != "" {
		a.startSkillProjectile(r, flight, impact)
	} else if impact != "" {
		elapsed := float32(0)
		if actor := a.playerActor(r.Source); actor != nil && actor.ActionActive() {
			elapsed = actor.Time
		}
		if d.Attack == 1 && (d.Effect == 1 || d.Effect == 3 || d.Effect == 5) {
			actor := a.playerActor(r.Source)
			slot := scene.SkillAction(d.Animation, d.Attack, d.Effect, a.playerWeapon(r.Source), a.attackVariant)
			if actor != nil && actor.PlayingAction(slot) && actor.ContactTime() > 0 {
				if actor.Time >= actor.ContactTime() {
					a.startSkillImpactAt(r.Target, impact, int(r.Level)-1, actor.Time-actor.ContactTime())
					return
				}
				if len(a.wv.impactCues) < 32 {
					a.wv.impactCues = append(a.wv.impactCues, skillImpactCue{actionClock{actor, actor.ActionID(), actor.ContactTime()}, r.Source, r.Target, impact, int(r.Level) - 1})
				}
				return
			}
		}
		a.startSkillImpactAt(r.Target, impact, int(r.Level)-1, elapsed)
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
	if file, sequence := skillSecondaryLibrary(r, d); file != "" {
		a.startSkillImpact(r.Target, file, sequence)
	}
}

// 0x579390/0x5783d0 select the recipient effect independently of the
// central announcement. A DA_area_* filename does not suppress that hit.
func skillSecondaryLibrary(r proto.SkillResult, d sdata.Skill) (string, int) {
	if r.Level == 0 || r.Level == 100 || r.Target == 0 || r.Result > 1 || d.Effect == 0 {
		return "", 0
	}
	if d.Effect == 2 || d.Effect == 3 || d.Effect == 4 {
		return effect.ImpactLibrary(r.ID), int(r.Level) - 1
	}
	// Native 0x577b40 maps AttribType to the five Adamage sequences.
	sequence := 4
	switch d.Attribute {
	case 1:
		sequence = 2
	case 2:
		sequence = 0
	case 3:
		sequence = 3
	case 4:
		sequence = 1
	}
	return "Effect/DA_target_Adamage.EFT", sequence
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
			a.startSkillImpactAt(cue.target, cue.file, cue.sequence, max(0, cue.actor.Time-cue.time))
		} else {
			kept = append(kept, cue)
		}
	}
	clear(a.wv.impactCues[len(kept):])
	a.wv.impactCues = kept
}

func (a *App) startSkillImpact(target uint32, file string, sequence int) {
	a.startSkillImpactAt(target, file, sequence, 0)
}

func (a *App) startSkillImpactAt(target uint32, file string, sequence int, elapsed float32) {
	if len(a.wv.impacts) >= 32 {
		return
	}
	pos, ok := a.effectPosition(target)
	if !ok {
		return
	}
	a.playSound(effect.ImpactSound(file), pos[0], pos[2])
	if x := a.loadSkillEffect(file, sequence); x != nil {
		// Position before catching up; otherwise births use the world origin.
		if m, ok := a.effectTargetMatrix(target); ok {
			x.SetTransform(m)
		}
		elapsed = min(max(0, elapsed), 2)
		advanceSkillEffect(x, elapsed)
		a.wv.impacts = append(a.wv.impacts, skillImpact{target: target, instance: x, age: elapsed})
	}
}
