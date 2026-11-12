package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/effect"
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
)

type castEffect struct {
	source   uint32
	hand     int
	until    float64
	instance *scene.Effect
}

func (a *App) stopCastEffects(source uint32) {
	if a.wv == nil {
		return
	}
	for _, e := range a.wv.castEffects {
		if e.source == source {
			e.instance.Stop()
		}
	}
}

func (a *App) confirmedEffects(p []byte) {
	v := a.wv
	if v == nil {
		return
	}
	switch proto.Op(p) {
	case proto.CharacterMove, proto.CharacterDeath:
		a.stopCastEffects(proto.NewReader(p).U32())
	case proto.SkillUse, proto.SkillUseNoTarget, proto.SkillUserDrain, proto.SkillMobDrain:
		a.stopCastEffects(a.state.LastSkill.Source)
		a.startSkillResultEffects(a.state.LastSkill)
	case proto.SkillPartyMember, proto.SkillMobOther:
		a.startSkillSecondaryEffect(a.state.LastSkill)
	case proto.SkillCastStarted:
		r := a.state.LastCast
		a.stopCastEffects(r.Source)
		a.loadSkills()
		d := a.skillDefs[sdata.SkillKey{ID: r.ID, Level: r.Level}]
		file := effect.CastLibrary(r.ID)
		for hand := 0; hand < 2; hand++ {
			x := a.loadSkillEffect(file, 1)
			if x == nil {
				return
			}
			if len(v.castEffects) >= 16 {
				v.castEffects = v.castEffects[1:]
			}
			v.castEffects = append(v.castEffects, castEffect{r.Source, hand, a.clock + float64(d.ReadyTime)*.25, x})
		}
	}
}

func (a *App) updateCastEffects() {
	v := a.wv
	alive := v.castEffects[:0]
	for _, e := range v.castEffects {
		if a.clock >= e.until {
			e.instance.Stop()
		}
		actor := a.playerActor(e.source)
		matrix, ok := actor.CastHand(e.hand)
		if ok {
			e.instance.SetTransform(matrix)
		} else {
			e.instance.Stop()
		}
		e.instance.Update(float32(a.frameDT))
		if e.instance.Done() {
			continue
		}
		alive = append(alive, e)
	}
	clear(v.castEffects[len(alive):])
	v.castEffects = alive
}

func (a *App) drawCastEffects() {
	if !a.clientSettings().SkillEffects {
		return
	}
	for _, e := range a.wv.castEffects {
		e.instance.Draw(a.scene3d)
	}
}
