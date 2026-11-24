package app

import (
	"math"
	"strconv"

	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

type combatNumberKind uint8

const (
	combatDamage combatNumberKind = iota
	combatHealHP
	combatHealSP
	combatHealMP
)

// Phoenix combat_numbers.gd: original count atlas, world-sized billboards,
// one-second rise/fade and a bounded queue. Positions detach at confirmation.
type combatNumber struct {
	Point            render.V3
	Started          float64
	Amount           uint32
	Critical, Missed bool
	Kind             combatNumberKind
}

func (a *App) showCombatNumber(target uint32, amount uint16, critical, missed bool, kind combatNumberKind) {
	a.showCombatNumberAmount(target, uint32(amount), critical, missed, kind)
}
func (a *App) showCombatNumberAmount(target, amount uint32, critical, missed bool, kind combatNumberKind) {
	if a.wv == nil || amount == 0 && !missed {
		return
	}
	point, ok := a.effectPosition(target)
	if !ok {
		return
	}
	point[1] += a.combatNumberHeight(target)
	a.emitCombatNumberAt(point, amount, critical, missed, kind)
}

func (a *App) showDrainRecovery(source uint32, hp, sp, mp uint16, critical bool) {
	if a.wv == nil || hp|sp|mp == 0 {
		return
	}
	point, ok := a.effectPosition(source)
	if !ok {
		return
	}
	point[1] += a.combatNumberHeight(source)
	spacing := float32(.4)
	if critical {
		spacing = .8
	}
	for i, amount := range [...]uint16{hp, sp, mp} {
		if amount == 0 {
			continue
		}
		a.emitCombatNumberAt(point, uint32(amount), critical, false, combatHealHP+combatNumberKind(i))
		point[1] += spacing
	}
}

func (a *App) emitCombatNumberAt(point render.V3, amount uint32, critical, missed bool, kind combatNumberKind) {
	if a.wv == nil || amount == 0 && !missed {
		return
	}
	for _, v := range point {
		if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
			return
		}
	}
	a.expireCombatNumbers()
	if len(a.combatNumbers) >= 128 {
		copy(a.combatNumbers, a.combatNumbers[1:])
		a.combatNumbers = a.combatNumbers[:127]
	}
	a.combatNumbers = append(a.combatNumbers, combatNumber{point, a.clock, amount, critical && !missed, missed, kind})
}
func (a *App) expireCombatNumbers() {
	live := a.combatNumbers[:0]
	for _, entry := range a.combatNumbers {
		if a.clock-entry.Started < 1 {
			live = append(live, entry)
		}
	}
	a.combatNumbers = live
}

func combatGlyphRect(point render.V3, width, height float32, view, projection render.M4, w, h int) (x, y, rw, rh float32, ok bool) {
	return combatGlyphRectVP(point, width, height, view, view.Mul(projection), w, h)
}

func combatGlyphRectVP(point render.V3, width, height float32, view, vp render.M4, w, h int) (x, y, rw, rh float32, ok bool) {
	right := render.V3{view[0], view[4], view[8]}
	up := render.V3{view[1], view[5], view[9]}
	project := func(p render.V3) (float32, float32, bool) {
		cw := p[0]*vp[3] + p[1]*vp[7] + p[2]*vp[11] + vp[15]
		if cw <= 0 {
			return 0, 0, false
		}
		px := p[0]*vp[0] + p[1]*vp[4] + p[2]*vp[8] + vp[12]
		py := p[0]*vp[1] + p[1]*vp[5] + p[2]*vp[9] + vp[13]
		return (px/cw + 1) * float32(w) / 2, (1 - py/cw) * float32(h) / 2, true
	}
	x, y, ok = project(point.Sub(right.Scale(width / 2)).Add(up.Scale(height)))
	if !ok {
		return
	}
	x2, y2, valid := project(point.Add(right.Scale(width / 2)))
	rw, rh = x2-x, y2-y
	ok = valid && rw > 0 && rh > 0 && x < float32(w) && y < float32(h) && x2 > 0 && y2 > 0
	return
}

func (a *App) drawCombatNumbers(w, h int) {
	a.expireCombatNumbers()
	if len(a.combatNumbers) == 0 || a.scene3d == nil || a.clientSettings().HideDamage {
		return // Hide Damage (0x44ae0e): the floating numbers are not drawn
	}
	atlas, err := a.Interface("count.png")
	if err != nil || atlas == nil || atlas.W != 512 || atlas.H != 32 {
		return
	}
	view, projection := a.scene3d.View, a.scene3d.Proj
	vp := view.Mul(projection)
	right := render.V3{view[0], view[4], view[8]}
	draw := func(entry combatNumber, age float32) {
		point := entry.Point.Add(render.V3{0, age * .3, 0})
		tint := entry.Kind.tint(1 - age)
		width := float32(.2)
		if entry.Critical {
			width *= 1.5
		}
		glyph := func(p render.V3, dw, dh float32, sx, sw int, color [4]float32) {
			x, y, rw, rh, ok := combatGlyphRectVP(p, dw, dh, view, vp, w, h)
			if !ok {
				return
			}
			a.sprites.Draw(atlas, x, y, rw, rh, float32(sx)/512, 0, float32(sx+sw)/512, 1, color)
		}
		if entry.Missed {
			glyph(point, width*4, width*1.1875, 416, 96, [4]float32{1, 1, 1, 1 - age})
			return
		}
		digits := strconv.FormatUint(uint64(entry.Amount), 10)
		for index, digit := range digits {
			offset := (float32(index) - float32(len(digits)-1)*.5) * width
			glyph(point.Add(right.Scale(offset)), width, width*1.1875, int(digit-'0')*26, 26, tint)
		}
		if entry.Critical {
			glyph(point.Add(render.V3{0, .4, 0}), width*2.75, width*1.1875, 260, 156, [4]float32{1, 1, 1, 1 - age})
		}
	}
	for _, entry := range a.combatNumbers {
		draw(entry, max(0, float32(a.clock-entry.Started)))
	}
}

func (kind combatNumberKind) tint(alpha float32) [4]float32 {
	switch kind {
	case combatHealHP:
		return [4]float32{0, 1, 0, alpha} // native recovery green (0x4274ff)
	case combatHealSP:
		return [4]float32{249.0 / 255, 189.0 / 255, 15.0 / 255, alpha} // SP gauge, main_stats_bar
	case combatHealMP:
		return [4]float32{0, 175.0 / 255, 229.0 / 255, alpha} // MP gauge, main_stats_bar
	default:
		return [4]float32{248.0 / 255, 247.0 / 255, 245.0 / 255, alpha}
	}
}

// Native users pass a logical origin .9 above feet and the default +1.2.
// Mobs instead pass their MON-authored height; a fixed +1.2 misplaces giants.
func (a *App) combatNumberHeight(target uint32) float32 {
	if target == a.state.SelfID {
		return 2.1
	}
	if m := a.wv.mobs[target]; m != nil && m.actor != nil {
		return m.actor.NumberHeight()
	}
	if e := a.state.Entities[target]; e != nil && e.Kind == game.KindUser {
		return 2.1
	}
	return 1.2
}
