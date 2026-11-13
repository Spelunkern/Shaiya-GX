package app

import (
	"log"

	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
)

type waterPresentation struct {
	swimming, touching, floating bool
	lastMark                     render.V3
	effects                      []*scene.Effect
	failed                       bool
}

// Phoenix player_controller.gd leaves stationary contact marks every .56 units
// and one ring on entering flotation. Use the extracted Cwater.EFT emitters,
// colours, waterRing/smoke textures and horizontal facing without synthetic FX.
func (a *App) updateWaterPresentation(dt float32, moving, swimming bool) {
	v := a.wv
	p := &v.waterPresentation
	p.swimming = swimming
	kept := p.effects[:0]
	for _, e := range p.effects {
		e.Update(dt)
		if !e.Done() {
			kept = append(kept, e)
		}
	}
	clear(p.effects[len(kept):])
	p.effects = kept
	touching := v.world.WaterContactAt(v.pos) && !a.state.Self.Dead && !a.state.Self.Mounted && !v.jumping
	floating := touching && swimming && !moving
	delta := v.pos.Sub(p.lastMark)
	delta[1] = 0
	if touching && (!p.touching || floating && !p.floating || moving && delta.Len() >= .56) {
		p.lastMark = v.pos
		if len(p.effects) < 8 && !p.failed {
			e, err := scene.LoadEffect(a.assets, "Effect/Cwater.EFT", 0)
			if err != nil {
				log.Printf("water contact: %v", err)
				p.failed = true
			} else {
				m := render.Identity()
				m[12], m[13], m[14] = v.pos[0], 0, v.pos[2]
				e.SetTransform(m)
				e.Update(1.0 / 60)
				p.effects = append(p.effects, e)
			}
		}
	}
	p.touching, p.floating = touching, floating
}
