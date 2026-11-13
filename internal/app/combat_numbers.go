package app

import (
	"math"
	"strconv"

	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

// Phoenix combat_numbers.gd: original count atlas, world-sized billboards,
// one-second rise/fade and a bounded queue. Positions detach at confirmation.
type combatNumber struct {
	Point                     render.V3
	Started                   float64
	Amount                    uint32
	Critical, Missed, Healing bool
}

func (a *App) showCombatNumber(target uint32, amount uint16, critical, missed, healing bool) {
	if a.wv == nil || amount == 0 && !missed {
		return
	}
	point, ok := a.effectPosition(target)
	if !ok {
		return
	}
	point[1] += 1.2
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
	a.combatNumbers = append(a.combatNumbers, combatNumber{point, a.clock, uint32(amount), critical && !missed, missed, healing})
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
	right := render.V3{view[0], view[4], view[8]}
	up := render.V3{view[1], view[5], view[9]}
	vp := view.Mul(projection)
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
	if len(a.combatNumbers) == 0 || a.scene3d == nil {
		return
	}
	atlas, err := a.Interface("count.tga")
	if err != nil || atlas == nil || atlas.W != 512 || atlas.H != 32 {
		return
	}
	view, projection := a.scene3d.View, a.scene3d.Proj
	right := render.V3{view[0], view[4], view[8]}
	for _, entry := range a.combatNumbers {
		age := max(0, float32(a.clock-entry.Started))
		point := entry.Point.Add(render.V3{0, age * .3, 0})
		tint := [4]float32{248.0 / 255, 247.0 / 255, 245.0 / 255, 1 - age}
		if entry.Healing {
			tint = [4]float32{46.0 / 255, 204.0 / 255, 113.0 / 255, 1 - age}
		}
		width := float32(.2)
		if entry.Critical {
			width *= 1.5
		}
		glyph := func(p render.V3, dw, dh float32, sx, sw int, color [4]float32) {
			x, y, rw, rh, ok := combatGlyphRect(p, dw, dh, view, projection, w, h)
			if !ok {
				return
			}
			a.sprites.Draw(atlas, x, y, rw, rh, float32(sx)/512, 0, float32(sx+sw)/512, 1, color)
		}
		if entry.Missed {
			glyph(point, width*4, width*1.1875, 416, 96, [4]float32{1, 1, 1, 1 - age})
			continue
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
}
