package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
)

// Only the actor animation is predicted. Results, impacts and resources still
// require server packets. An acknowledgment must not restart this same action.
type requestedSkillPresentation struct {
	key     sdata.SkillKey
	target  uint32
	actor   *scene.Character
	action  uint64
	started float64
}

func (a *App) requestedSkillPresentation(packet []byte) {
	if a.wv == nil || len(packet) != 7 || proto.Op(packet) != proto.SkillUse && proto.Op(packet) != proto.SkillUseNoTarget {
		return
	}
	r := proto.NewReader(packet)
	number, target := r.U8(), r.U32()
	k, ok := a.state.Skills[number]
	if !ok {
		return
	}
	key := sdata.SkillKey{ID: k.ID, Level: k.Level}
	d, ok := a.skillDefs[key]
	if !ok || !a.validSkillTarget(d, target) {
		return
	}
	actor := a.wv.self
	if actor == nil {
		return
	}
	// Timed casts start on the server's cast-start packet. Instant skills
	// can begin their release animation as soon as the request is sent.
	if d.ReadyTime > 0 {
		return
	}
	p := &requestedSkillPresentation{key: key, target: target, actor: actor, started: a.clock}
	a.attackVariant++
	played := actor.PlayAction(scene.SkillAction(d.Animation, d.Attack, d.Effect, a.playerWeapon(a.state.SelfID), a.attackVariant), false)
	if played {
		p.action = actor.ActionID()
		a.wv.requestedSkill = p
	}
}

func (a *App) updateRequestedSkillPresentation() {
	p := a.wv.requestedSkill
	if p == nil {
		return
	}
	if p.actor != a.wv.self || p.actor.ActionID() != p.action || a.clock-p.started > 15 {
		a.wv.requestedSkill = nil
		return
	}
}

func (a *App) matchingRequestedSkill(r proto.SkillResult) bool {
	p := a.wv.requestedSkill
	return p != nil && r.Source == a.state.SelfID && p.key == (sdata.SkillKey{ID: r.ID, Level: r.Level}) && (p.target == r.Target || r.Result > 4) && p.actor == a.wv.self && p.actor.ActionID() == p.action && a.clock-p.started < 15
}
