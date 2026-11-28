package ui

import (
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

const PersonalMarketWidth, PersonalMarketHeight = 192, 391

const PersonalMarketNone = -1
const (
	PersonalMarketClose = proto.MyShopSlots + iota
	PersonalMarketMessage
	PersonalMarketToggle
	PersonalMarketCancel
	PersonalMarketRemove
)

// Native layout (Game.exe 0x4ebae0): Open Market at (23,353) and the
// Cancel half of stats_on_button at (116,353), both 64x32.
type PersonalMarket struct {
	Position            WindowPosition
	frame, open, cancel *render.Texture
	close               Button
	held                int
}

func NewPersonalMarket(t Textures) (*PersonalMarket, error) {
	ld := &loader{t: t}
	p := &PersonalMarket{held: PersonalMarketNone, close: nativeClose(ld.get("talk1_close_button.png"), 163, 5)}
	if optional, ok := t.(interface{ OptionalInterface(string) *render.Texture }); ok {
		p.frame = optional.OptionalInterface("p_market.png")
		p.open = optional.OptionalInterface("p_market_button.png")
		p.cancel = optional.OptionalInterface("stats_on_button.png")
	}
	return p, ld.err
}

func (p *PersonalMarket) Origin(w, h int) (int, int) {
	return p.Position.Origin(w, h, PersonalMarketWidth, PersonalMarketHeight, min(46, max(0, w-PersonalMarketWidth)), min(86, max(0, h-PersonalMarketHeight)))
}

func PersonalMarketCell(index int) (x, y int) {
	return 21 + index%4*39, 147 + index/4*39
}

func PersonalMarketCellAt(x, y int) int {
	for i := range proto.MyShopSlots {
		cx, cy := PersonalMarketCell(i)
		if x >= cx && x < cx+32 && y >= cy && y < cy+32 {
			return i
		}
	}
	return PersonalMarketNone
}

func (p *PersonalMarket) CancelHeld() {
	p.held = PersonalMarketNone
	p.close.held = false
}

// FreeAt keeps the drag handler and the controls on the same hit areas.
func (p *PersonalMarket) FreeAt(mx, my, w, h int) bool {
	x, y := p.Origin(w, h)
	close := p.close
	close.OriginX, close.OriginY = x, y
	return !close.hit(&platform.Input{MouseX: mx, MouseY: my}) &&
		personalMarketControlAt(mx-x, my-y, false) == PersonalMarketNone
}

func personalMarketControlAt(x, y int, buyer bool) int {
	switch {
	case x >= 16 && x < 174 && y >= 62 && y < 119 && !buyer:
		return PersonalMarketMessage
	case y >= 353 && y < 385 && !buyer && x >= 23 && x < 87:
		return PersonalMarketToggle
	case y >= 353 && y < 385 && x >= 116 && x < 180:
		return PersonalMarketCancel
	default:
		return PersonalMarketCellAt(x, y)
	}
}

// Update returns the control released this frame.
func (p *PersonalMarket) Update(in *platform.Input, w, h int, buyer bool) int {
	x, y := p.Origin(w, h)
	p.close.OriginX, p.close.OriginY = x, y
	if p.close.Update(in) || in.CloseOnEscape() {
		return PersonalMarketClose
	}
	hit := personalMarketControlAt(in.MouseX-x, in.MouseY-y, buyer)
	if in.Pressed {
		p.held = hit
	}
	if in.Clicked {
		held := p.held
		p.held = PersonalMarketNone
		if held == hit {
			return hit
		}
	}
	return PersonalMarketNone
}

// Draw the native frame, buttons and the message. Item icons come from the app.
func (p *PersonalMarket) Draw(s *render.Sprites, f *Font, w, h int, title string, buyer, selling, busy bool) {
	x, y := p.Origin(w, h)
	if p.frame != nil {
		s.Draw(p.frame, float32(x), float32(y), PersonalMarketWidth, PersonalMarketHeight, 0, 0, float32(PersonalMarketWidth)/256, float32(PersonalMarketHeight)/512, white)
	}
	_, lines := wrapMarkupWidth(f, title, white, false, 151)
	for i, line := range lines[:min(len(lines), max(1, 50/f.Height))] {
		f.Draw(s, line, x+20, y+65+i*f.Height, white, 151)
	}
	row := func(held bool) float32 {
		switch {
		case busy:
			return .75
		case held:
			return .5
		}
		return 0
	}
	if p.open != nil && !buyer && !selling {
		r := row(p.held == PersonalMarketToggle)
		s.Draw(p.open, float32(x+23), float32(y+353), 64, 32, 0, r, 1, r+.25, white)
	}
	if p.cancel != nil {
		u := float32(0)
		switch {
		case busy:
			u = .75
		case p.held == PersonalMarketCancel:
			u = .25
		}
		s.Draw(p.cancel, float32(x+116), float32(y+353), 64, 32, u, .5, u+.25, 1, white)
	}
	p.close.OriginX, p.close.OriginY = x, y
	p.close.Draw(s)
}
