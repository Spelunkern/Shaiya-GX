package ui

import (
	"github.com/Spelunkern/shaiya-go-client/internal/platform"
	"github.com/Spelunkern/shaiya-go-client/internal/render"
)

var white = [4]float32{1, 1, 1, 1}

// Button states, in the order of the UV arrays of the widget object
// (0x527000 layout: u0 @+0x48, u1 @+0x5c, v0 @+0x70, v1 @+0x84, 5 each).
const (
	StateNormal = iota
	StateHover
	StatePressed
	StateChecked
	StateDisabled
)

// UV is one state's texture rect.
type UV struct{ U0, V0, U1, V1 float32 }

// FourRowUV is the usual 4-row button sheet passed by the constructors
// (normal, hover, pressed, pressed again, disabled).
var FourRowUV = [5]UV{{0, 0, 1, .25}, {0, .25, 1, .5}, {0, .5, 1, .75}, {0, .5, 1, .75}, {0, .75, 1, 1}}

// Button is the 0x528830 widget: drawn at origin+offset with size DrawW x
// DrawH, clickable over HitW x HitH.
type Button struct {
	Tex               *render.Texture
	OriginX, OriginY  int
	OffX, OffY        int
	DrawW, DrawH      int
	HitW, HitH        int
	UV                [5]UV
	Disabled, Checked bool
	Toggle            bool // checkbox: a click flips Checked
	state             int
	held              bool
}

func (b *Button) X() int { return b.OriginX + b.OffX }
func (b *Button) Y() int { return b.OriginY + b.OffY }

func (b *Button) hit(in *platform.Input) bool {
	x, y := b.X(), b.Y()
	return in.MouseX >= x && in.MouseX < x+b.HitW && in.MouseY >= y && in.MouseY < y+b.HitH
}

// Update returns true when the button was clicked this frame.
func (b *Button) Update(in *platform.Input) bool {
	if b.Disabled {
		b.state = StateDisabled
		return false
	}
	over := b.hit(in)
	if in.Pressed && over {
		b.held = true
	}
	clicked := false
	if in.Clicked {
		clicked = b.held && over
		b.held = false
	}
	switch {
	case b.held && over:
		b.state = StatePressed
	case over:
		b.state = StateHover
	default:
		b.state = StateNormal
	}
	if clicked && b.Toggle {
		b.Checked = !b.Checked
	}
	return clicked
}

func (b *Button) Draw(s *render.Sprites) {
	if b.Tex == nil {
		return
	}
	st := b.state
	if b.Toggle && b.Checked && st != StateDisabled {
		st = StateChecked
	}
	uv := b.UV[st]
	s.Draw(b.Tex, float32(b.X()), float32(b.Y()), float32(b.DrawW), float32(b.DrawH), uv.U0, uv.V0, uv.U1, uv.V1, white)
}

// TextField is a single-line Latin-1 input.
type TextField struct {
	Text   string
	Max    int
	Masked bool
}

// Edit applies typed characters and backspace.
func (t *TextField) Edit(in *platform.Input) {
	rs := []rune(t.Text)
	for _, r := range in.Chars {
		if r < 32 || r > 0xFF || len(rs) >= t.Max {
			continue
		}
		rs = append(rs, r)
	}
	if in.HasKey(platform.KeyBackspace) && len(rs) > 0 {
		rs = rs[:len(rs)-1]
	}
	t.Text = string(rs)
}

// Shown is the text as drawn (asterisks when masked).
func (t *TextField) Shown() string {
	if !t.Masked {
		return t.Text
	}
	b := make([]byte, len([]rune(t.Text)))
	for i := range b {
		b[i] = '*'
	}
	return string(b)
}
