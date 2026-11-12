package app

import (
	"log"

	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
)

type skillProjectile struct {
	target     uint32
	sequence   int
	impact     string
	pos        render.V3
	age, speed float32
	instance   *scene.Effect
	arrived    bool
}
type skillImpact struct {
	target   uint32
	age      float32
	instance *scene.Effect
}

// Native magic mode 2 (0x57e320 -> 0x450650 -> 0x40e760): start at
// caster Y+0.7, wait 12 keys at default 30/s, then speed=initial distance*5.
// HP and other gameplay outcomes have already been applied from the server.
func (a *App) startSkillProjectile(r proto.SkillResult, file, impact string) {
	if len(a.wv.projectiles) >= 32 {
		return
	}
	source, ok := a.effectPosition(r.Source)
	if !ok {
		return
	}
	if _, ok := a.effectPosition(r.Target); !ok {
		return
	}
	x := a.loadSkillEffect(file, int(r.Level)-1)
	if x == nil {
		return
	}
	a.wv.projectiles = append(a.wv.projectiles, skillProjectile{target: r.Target, sequence: int(r.Level) - 1, impact: impact, pos: source.Add(render.V3{0, .7, 0}), instance: x})
}
func (a *App) loadSkillEffect(file string, sequence int) *scene.Effect {
	if file == "" {
		return nil
	}
	x, err := scene.LoadEffect(a.assets, file, sequence)
	v := a.wv
	if v.effectErrors == nil {
		v.effectErrors = map[string]bool{}
	}
	if !v.effectErrors[file] && (err != nil || x.Skipped > 0) {
		if err != nil {
			log.Printf("skill effect %s: %v", file, err)
		} else {
			log.Printf("skill effect %s: %d unsupported emitters omitted", file, x.Skipped)
		}
		v.effectErrors[file] = true
	}
	return x
}
func (a *App) effectPosition(id uint32) (render.V3, bool) {
	if id == a.state.SelfID {
		return a.wv.pos, true
	}
	if e := a.state.Entities[id]; e != nil {
		return a.visualEntityPosition(e), true
	}
	return render.V3{}, false
}
func (a *App) effectTargetMatrix(id uint32) (render.M4, bool) {
	if c := a.playerActor(id); c != nil {
		// Native 0x417ae0 anchors player effects 0.9 below their world origin.
		m := c.World
		m[13] -= .9
		return m, true
	}
	if c := a.wv.mobs[id]; c != nil && c.actor != nil {
		return c.actor.World, true
	}
	if p, ok := a.effectPosition(id); ok {
		if id == a.state.SelfID || a.state.Entities[id].Kind == game.KindUser {
			p[1] -= .9
		}
		return render.Basis(p, render.V3{0, 0, 1}, render.V3{0, 1, 0}), true
	}
	return render.M4{}, false
}
func (a *App) updateResolvedEffects() {
	v := a.wv
	a.updateSkillImpactCues()
	dt := float32(min(max(a.frameDT, 0), .25))
	kept := v.projectiles[:0]
	for _, p := range v.projectiles {
		p.age += dt
		target, ok := a.effectPosition(p.target)
		if !ok || p.age > 10 {
			p.arrived = true
			p.instance.Stop()
		}
		if !p.arrived && p.age >= .4 {
			if e := a.state.Entities[p.target]; e != nil && e.Kind == game.KindMob {
				target[1] += .9
			}
			dir := p.move(target, dt)
			if p.arrived {
				p.instance.Stop()
				a.startSkillImpact(p.target, p.impact, p.sequence)
			}
			p.instance.SetTransform(render.Basis(p.pos, dir, render.V3{0, 1, 0}))
		}
		if p.age < .4 && !p.arrived {
			kept = append(kept, p)
			continue
		}
		p.instance.Update(min(dt, max(0, p.age-.4)))
		if p.instance.Done() && p.arrived {
			continue
		}
		kept = append(kept, p)
	}
	clear(v.projectiles[len(kept):])
	v.projectiles = kept
	alive := v.impacts[:0]
	for _, p := range v.impacts {
		p.age += dt
		if m, ok := a.effectTargetMatrix(p.target); ok {
			p.instance.SetTransform(m)
		} else {
			p.instance.Stop()
		}
		if p.age >= 5 {
			p.instance.Stop()
		} // Bound lingering authored loops in this first delivery.
		p.instance.Update(dt)
		if p.instance.Done() {
			continue
		}
		alive = append(alive, p)
	}
	clear(v.impacts[len(alive):])
	v.impacts = alive
}

func (a *App) drawResolvedEffects() {
	if !a.clientSettings().SkillEffects {
		return
	}
	for _, p := range a.wv.projectiles {
		if p.age >= .4 || p.arrived {
			p.instance.Draw(a.scene3d)
		}
	}
	for _, p := range a.wv.impacts {
		p.instance.Draw(a.scene3d)
	}
}

func (p *skillProjectile) move(target render.V3, dt float32) render.V3 {
	delta := target.Sub(p.pos)
	distance := delta.Len()
	if p.speed == 0 {
		p.speed = distance * 5
	}
	step := p.speed * min(dt, max(0, p.age-.4))
	if distance <= step+.005 {
		p.pos = target
		p.arrived = true
	} else {
		p.pos = p.pos.Add(delta.Norm().Scale(step))
	}
	return delta.Norm()
}
