package app

import (
	"log"

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
	delay      float32
	instance   *scene.Effect
	arrived    bool
}
type skillImpact struct {
	target   uint32
	age      float32
	instance *scene.Effect
}

// Native magic mode 2 (0x57e320 -> 0x450650 -> 0x40e760): start at
// caster logical Y+0.7 (feet+1.6), wait 12 keys at default 30/s, then
// speed=initial distance*5. Time already spent in release is not waited twice.
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
	elapsed := float32(0)
	if actor := a.playerActor(r.Source); actor != nil && actor.ActionActive() {
		elapsed = actor.Time
	}
	p := skillProjectile{target: r.Target, sequence: int(r.Level) - 1, impact: impact, pos: source.Add(render.V3{0, 1.6, 0}), instance: x, delay: .4, age: elapsed}
	target, _ := a.effectPosition(r.Target)
	target[1] += .9
	flightElapsed := max(0, elapsed-p.delay)
	if flightElapsed > 0 {
		distance := target.Sub(p.pos).Len()
		dir := p.move(target, flightElapsed)
		if p.arrived {
			// At distance*5 the authored flight takes 0.2s. A late result
			// must not replay the launch and the whole flight after release.
			travel := float32(0)
			if p.speed > 0 {
				travel = distance / p.speed
			}
			a.startSkillImpactAt(r.Target, impact, p.sequence, max(0, flightElapsed-travel))
			return
		}
		p.instance.SetTransform(render.Basis(p.pos, dir, render.V3{0, 1, 0}))
		p.instance.Update(flightElapsed)
	}
	a.wv.projectiles = append(a.wv.projectiles, p)
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
		// Native 0x417ae0 subtracts .9 from the logical origin. Character.World
		// is already at the feet here; subtracting it again buries ground FX.
		return c.World, true
	}
	if c := a.wv.mobs[id]; c != nil && c.actor != nil {
		return c.actor.World, true
	}
	if p, ok := a.effectPosition(id); ok {
		return render.Basis(p, render.V3{0, 0, 1}, render.V3{0, 1, 0}), true
	}
	return render.M4{}, false
}
func (a *App) updateResolvedEffects() {
	v := a.wv
	initialImpacts := len(v.impacts)
	a.updateSkillImpactCues()
	dt := float32(min(max(a.frameDT, 0), .25))
	a.updateSkillReleaseEffects(dt)
	kept := v.projectiles[:0]
	for _, p := range v.projectiles {
		p.age += dt
		target, ok := a.effectPosition(p.target)
		if !ok || p.age > 10 {
			p.arrived = true
			p.instance.Stop()
		}
		if !p.arrived && p.age >= p.delay {
			target[1] += .9 // both users and mobs are stored at their feet
			dir := p.move(target, dt)
			if p.arrived {
				p.instance.Stop()
				a.startSkillImpact(p.target, p.impact, p.sequence)
			}
			p.instance.SetTransform(render.Basis(p.pos, dir, render.V3{0, 1, 0}))
		}
		if p.age < p.delay && !p.arrived {
			kept = append(kept, p)
			continue
		}
		p.instance.Update(min(dt, max(0, p.age-p.delay)))
		if p.instance.Done() && p.arrived {
			continue
		}
		kept = append(kept, p)
	}
	clear(v.projectiles[len(kept):])
	v.projectiles = kept
	alive := v.impacts[:0]
	for i, p := range v.impacts {
		advance := dt
		if i >= initialImpacts {
			advance = 0
		}
		p.age += advance
		if m, ok := a.effectTargetMatrix(p.target); ok {
			p.instance.SetTransform(m)
		} else {
			p.instance.Stop()
		}
		if p.age >= 5 {
			p.instance.Stop()
		} // Bound lingering authored loops in this first delivery.
		p.instance.Update(advance)
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
	for _, r := range a.wv.releaseEffects {
		r.instance.Draw(a.scene3d)
	}
	for _, p := range a.wv.projectiles {
		if p.age >= p.delay || p.arrived {
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
	step := p.speed * min(dt, max(0, p.age-p.delay))
	if distance <= step+.005 {
		p.pos = target
		p.arrived = true
	} else {
		p.pos = p.pos.Add(delta.Norm().Scale(step))
	}
	return delta.Norm()
}
