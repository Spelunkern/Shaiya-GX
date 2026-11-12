package app

import (
	"fmt"
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
)

func (a *App) playerActor(id uint32) *scene.Character {
	if a.wv == nil {
		return nil
	}
	if id == a.state.SelfID {
		return a.wv.self
	}
	if actor := a.wv.actors[id]; actor != nil {
		return actor.character
	}
	return nil
}

func (a *App) playerWeapon(id uint32) uint8 {
	if id == a.state.SelfID {
		return a.state.Inventory[[2]uint8{0, 5}].Type
	}
	if e := a.state.Entities[id]; e != nil && e.Shape != nil {
		return e.Shape.Equipment[5][0]
	}
	return 0
}

// confirmedAnimation presents network outcomes; it never changes health.
func (a *App) confirmedAnimation(p []byte) {
	op := proto.Op(p)
	v := a.wv
	if v == nil {
		return
	}
	switch op {
	case proto.CharacterMotion:
		m := a.state.LastMotion
		if m.ID == a.state.SelfID && (m.Motion == proto.MotionSitDown || m.Motion == proto.MotionStandUp) {
			v.sitPending = false
		}
		if m.Motion == proto.MotionSitDown || m.Motion == proto.MotionStandUp {
			if actor := a.playerActor(m.ID); actor != nil {
				action := 10
				if m.Motion == proto.MotionStandUp {
					action = 11
				}
				actor.PlayAction(action, false)
			}
		}
	case proto.SkillCastStarted:
		r := a.state.LastCast
		if actor := a.playerActor(r.Source); actor != nil {
			a.loadSkills()
			d := a.skillDefs[sdata.SkillKey{ID: r.ID, Level: r.Level}]
			a.faceSkillTarget(r.Source, r.Target)
			actor.PlayCast(scene.SkillCastAction(d.Animation, d.Effect), float32(d.ReadyTime)*.25)
		}
	case proto.SkillUse, proto.SkillUseNoTarget, proto.SkillUserDrain, proto.SkillMobDrain:
		r := a.state.LastSkill
		if actor := a.playerActor(r.Source); actor != nil {
			actor.CancelCast()
		}
		if r.Result > 4 || r.Result == 3 && r.Target == 0 {
			return
		}
		if actor := a.playerActor(r.Source); actor != nil {
			a.loadSkills()
			d, ok := a.skillDefs[sdata.SkillKey{ID: r.ID, Level: r.Level}]
			if !ok {
				return
			}
			weapon := a.playerWeapon(r.Source)
			a.faceSkillTarget(r.Source, r.Target)
			a.attackVariant++
			actor.PlayAction(scene.SkillAction(d.Animation, d.Attack, d.Effect, weapon, a.attackVariant), false)
		}
	case proto.CharacterMove:
		if actor := a.playerActor(proto.NewReader(p).U32()); actor != nil {
			actor.CancelCast()
			if move, err := proto.DecodeUserMove(p); err == nil && move.Motion&0x3f != 0 {
				cancelGesture(actor)
			}
		}
	case proto.CharacterMobAutoAttack, proto.CharacterUserAutoAttack, proto.MobAttackCharacter:
		result := a.state.LastAttack
		if actor := a.playerActor(result.Source); actor != nil {
			typ := a.playerWeapon(result.Source)
			if result.Source == a.state.SelfID {
				if target := a.state.Entities[result.Target]; target != nil {
					dir := (render.V3{target.X, 0, target.Z}).Sub(render.V3{v.pos[0], 0, v.pos[2]}).Norm()
					if dir.Len() > 0 {
						v.faceCombatTarget(dir)
					}
				}
			}
			a.attackVariant++
			if actor.PlayAction(scene.AttackAction(typ, a.attackVariant), false) {
				a.queueAttackSounds(result, typ, actor)
			}
		}
		if mob := v.mobs[result.Source]; mob != nil && mob.actor != nil {
			mob.actor.PlayOnce(2)
		}
		if result.HP > 0 {
			if mob := v.mobs[result.Target]; mob != nil && mob.actor != nil {
				mob.actor.PlayOnce(7)
			}
		}
		a.combatNotice = fmt.Sprintf("Damage: %d", result.HP)
		if result.Result == 3 {
			a.combatNotice = "Miss"
		}
		a.combatNoticeUntil = a.clock + 2
	case proto.CharacterDeath:
		id := proto.NewReader(p).U32()
		if c := a.playerActor(id); c != nil {
			c.PlayAction(9, true)
		}
	case proto.CharacterResurrect:
		id := proto.NewReader(p).U32()
		if c := a.playerActor(id); c != nil {
			c.ReleaseAction()
		}
	}
}

func (a *App) faceSkillTarget(source, target uint32) {
	c := a.playerActor(source)
	if c == nil || source == target {
		return
	}
	e := a.state.Entities[target]
	if e == nil {
		return
	}
	pos := render.V3{c.World[12], c.World[13], c.World[14]}
	dir := (render.V3{e.X, pos[1], e.Z}).Sub(pos)
	if dir.Len() < .001 {
		return
	}
	if source == a.state.SelfID {
		a.wv.faceCombatTarget(dir.Norm())
	} else {
		a.wv.actors[source].facing = dir.Norm()
	}
}
