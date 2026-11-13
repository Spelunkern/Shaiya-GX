package ui

import (
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

// The 0x52c430 scrollbar: 13x19 arrow hit areas and the original 13x8 thumb.
type panelScroll struct {
	texture      *render.Texture
	x, y, height int
	arrows       [2]Button
	dragging     bool
	grab         int
}

func newPanelScroll(texture *render.Texture, x, y, height int) panelScroll {
	p := panelScroll{texture: texture, x: x, y: y, height: height}
	for i := range p.arrows {
		p.arrows[i] = Button{Tex: texture, OffX: x, OffY: y + i*(height-19), DrawW: 16, DrawH: 32, HitW: 13, HitH: 19}
		for state, col := range []int{0, 1, 1, 1, 2} {
			p.arrows[i].UV[state] = UV{float32(col) / 8, float32(i) / 2, float32(col+1) / 8, float32(i+1) / 2}
		}
	}
	return p
}
func (p *panelScroll) Update(in *platform.Input, x, y, count, visible, offset int) int {
	last := max(0, count-visible)
	offset = max(0, min(last, offset))
	for i := range p.arrows {
		b := &p.arrows[i]
		b.OriginX, b.OriginY = x, y
		b.Disabled = last == 0
		if b.Update(in) {
			offset = max(0, min(last, offset+2*i-1))
		}
	}
	track := max(1, p.height-46)
	thumb := y + p.y + 19
	if last > 0 {
		thumb += offset * track / last
	}
	if in.Pressed && last > 0 && in.MouseX >= x+p.x && in.MouseX < x+p.x+13 && in.MouseY >= y+p.y+19 && in.MouseY < y+p.y+p.height-19 {
		p.dragging = true
		p.grab = 4
		if in.MouseY >= thumb && in.MouseY < thumb+8 {
			p.grab = in.MouseY - thumb
		}
	}
	if p.dragging && (in.MouseDown || in.Pressed) {
		offset = max(0, min(last, (in.MouseY-y-p.y-19-p.grab)*last/track))
	}
	if !in.MouseDown && !in.Pressed {
		p.dragging = false
	}
	return offset
}
func (p *panelScroll) Draw(s *render.Sprites, x, y, count, visible, offset int) {
	for i := range p.arrows {
		b := &p.arrows[i]
		b.OriginX, b.OriginY = x, y
		b.Disabled = count <= visible
		b.Draw(s)
	}
	if p.texture == nil || count <= visible {
		return
	}
	thumb := p.y + 19 + max(0, min(count-visible, offset))*max(1, p.height-46)/(count-visible)
	s.Draw(p.texture, float32(x+p.x), float32(y+thumb), 13, 8, .5, 0, .5+13.0/128, 8.0/64, white)
}
