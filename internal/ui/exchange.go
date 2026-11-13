package ui

import (
	"fmt"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

const ExchangeWidth, ExchangeHeight = 200, 489

type ExchangePanel struct {
	Position       WindowPosition
	frame          *render.Texture
	confirm, close Button
	duel           bool
}

func NewExchangePanel(t Textures) (*ExchangePanel, error) {
	return newExchangePanel(t, "change.png")
}
func NewDuelPanel(t Textures) (*ExchangePanel, error) {
	return newExchangePanel(t, "1on1_battle.png")
}
func newExchangePanel(t Textures, frame string) (*ExchangePanel, error) {
	ld := &loader{t: t}
	p := &ExchangePanel{frame: ld.get(frame), duel: frame == "1on1_battle.png"}
	p.confirm = Button{Tex: ld.get("change_decision_button.png"), OffX: 67, OffY: 210, DrawW: 128, DrawH: 32, HitW: 65, HitH: 26, UV: [5]UV{{0, 0, .25, 1}, {.25, 0, .5, 1}, {.5, 0, .75, 1}, {.5, 0, .75, 1}, {.75, 0, 1, 1}}}
	p.close = nativeClose(ld.get("talk1_close_button.png"), 174, 5)
	return p, ld.err
}
func (p *ExchangePanel) Origin(w, h int) (int, int) {
	return p.Position.Origin(w, h, ExchangeWidth, ExchangeHeight, 32, min(82, max(0, h-ExchangeHeight)))
}
func ExchangeCell(side, index int) (int, int, bool) {
	if side < 0 || side > 1 || index < 0 || index >= 8 {
		return 0, 0, false
	}
	return 22 + (index%4)*40, 91 + (index/4)*40 + side*237, true
}
func ExchangeHit(x, y int) (int, bool) {
	for i := 0; i < 8; i++ {
		cx, cy, _ := ExchangeCell(0, i)
		if x >= cx && x < cx+32 && y >= cy && y < cy+32 {
			return i, true
		}
	}
	return 0, false
}
func (p *ExchangePanel) Update(in *platform.Input, w, h int) (confirm, close bool) {
	x, y := p.Origin(w, h)
	p.confirm.OriginX, p.confirm.OriginY = x, y
	p.close.OriginX, p.close.OriginY = x, y
	return p.confirm.Update(in), p.close.Update(in)
}
func (p *ExchangePanel) Draw(s *render.Sprites, f *Font, w, h int, name string, ownMoney, otherMoney uint32, ready, otherReady bool) {
	x, y := p.Origin(w, h)
	s.Draw(p.frame, float32(x), float32(y), ExchangeWidth, ExchangeHeight, 0, 0, 200.0/256, 489.0/512, white)
	if p.duel {
		f.Draw(s, name, x+45, y+294, white, 120)
	} else {
		f.Draw(s, name, x+20, y+299, white, 160)
	}
	f.Draw(s, fmt.Sprint(ownMoney), x+90, y+179, white, 90)
	f.Draw(s, fmt.Sprint(otherMoney), x+90, y+416, white, 90)
	if ready {
		f.Draw(s, "Ready", x+72, y+242, white, 64)
	}
	if otherReady {
		f.Draw(s, "Ready", x+72, y+448, white, 64)
	}
	p.confirm.OriginX, p.confirm.OriginY = x, y
	p.close.OriginX, p.close.OriginY = x, y
	p.confirm.Draw(s)
	p.close.Draw(s)
}
