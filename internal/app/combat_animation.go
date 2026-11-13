package app

import (
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
		// The server may require preparation even when local SData says
		// instant. Its cast-start supersedes the predicted release and clock.
		if r.Source == a.state.SelfID {
			v.requestedSkill = nil
		}

		if actor := a.playerActor(r.Source); actor != nil {
			a.loadSkills()
			d := a.skillDefs[sdata.SkillKey{ID: r.ID, Level: r.Level}]
			a.faceSkillTarget(r.Source, r.Target)
			actor.PlayCast(scene.SkillCastAction(d.Animation, d.Effect), float32(d.ReadyTime)*.25)
		}
	case proto.SkillUse, proto.SkillUseNoTarget, proto.SkillUserDrain, proto.SkillMobDrain:
		r := a.state.LastSkill
		predicted := a.matchingRequestedSkill(r)
		if r.Source == a.state.SelfID {
			v.requestedSkill = nil
		}
		if predicted {
			if r.Result > 4 || r.Result == 3 && r.Target == 0 {
				a.wv.self.ReleaseAction()
			} else {
				return
			}
		}
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
			mob.attackTarget = result.Target
			mob.actor.PlayOnce(mob.actor.AttackAction(0))
		}
		if result.HP > 0 {
			if mob := v.mobs[result.Target]; mob != nil && mob.actor != nil {
				mob.actor.PlayOnce(7)
			}
		}
		if result.Source == a.state.SelfID || result.Target == a.state.SelfID || result.Target == a.state.Target {
			a.showCombatNumber(result.Target, result.HP, result.Result == 1, result.Result == 3, false)
		}
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
	if a.wv == nil || source == target {
		return
	}
	goal, ok := a.effectPosition(target)
	if !ok {
		return
	}
	if source == a.state.SelfID {
		dir := goal.Sub(a.wv.pos)
		dir[1] = 0
		if dir.Len() > .001 {
			a.wv.faceCombatTarget(dir.Norm())
		}
	} else if actor := a.wv.actors[source]; actor != nil && actor.character != nil {
		pos := a.visualEntityPosition(actor.entity)
		dir := goal.Sub(pos)
		dir[1] = 0
		if dir.Len() > .001 {
			actor.facing = dir.Norm()
		}
	}
}

// Facing is predicted on input, before any attack/cast confirmation. Publish
// the stationary direction before the action so peers see the same heading.
func (a *App) faceRequestedTarget(target uint32) bool {
	if a.wv == nil {
		return true
	}
	previous := a.wv.facing
	a.faceSkillTarget(a.state.SelfID, target)
	if a.world != nil && previous.Sub(a.wv.facing).Len() > .0001 {
		if err := a.world.Send(a.wv.movePacket(0)); err != nil {
			a.backToLogin("Unable to turn toward target.")
			return false
		}
	}
	return true
}
