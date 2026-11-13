package ui

import (
	"math"

	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

// PreviewNavigation shares the native six-button control between selection
// and creation (0x46ce30/0x468980; actions in 0x46a890/0x465020).
type PreviewNavigation struct {
	bg           *render.Texture
	buttons      [6]Button
	Angle        float64
	turn         float64
	keyboardTurn float64
	Face         bool
}

func newPreviewNavigation(ld *loader) PreviewNavigation {
	n := PreviewNavigation{bg: ld.get("character_navi_bg.png")}
	tex := ld.get("character_navi.png")
	positions := [6][2]int{{8, 38}, {99, 38}, {41, 39}, {69, 39}, {53, 7}, {53, 76}}
	for i, p := range positions {
		b := &n.buttons[i]
		*b = Button{Tex: tex, OffX: p[0], OffY: p[1], DrawW: 32, DrawH: 32, HitW: 23, HitH: 23}
		for state, row := range []int{0, 1, 2, 2, 3} {
			b.UV[state] = UV{float32(i) / 8, float32(row) / 4, float32(i+1) / 8, float32(row+1) / 4}
		}
	}
	return n
}

func (n *PreviewNavigation) Reset() {
	n.Angle, n.turn, n.keyboardTurn, n.Face = 0, 0, 0, false
	for i := range n.buttons {
		n.buttons[i].held = false
	}
}

func (n *PreviewNavigation) Update(in *platform.Input, x, y int, enabled, creation bool) {
	n.keyboardTurn = 0
	for i := range n.buttons {
		b := &n.buttons[i]
		b.OriginX, b.OriginY, b.Disabled = x, y, !enabled
		if !b.Update(in) {
			continue
		}
		switch i {
		case 0:
			n.turn = -1
		case 1:
			n.turn = 1
		case 2:
			n.turn = 0
		case 3:
			n.Angle, n.turn = 0, 0
		case 4, 5:
			n.Face = i == 4
			if creation {
				n.Angle = 0 // 0x465020 resets the creation camera on zoom.
			}
		}
	}
	if enabled {
		if in.Held('<') || in.HasKey(platform.KeyLeft) {
			n.keyboardTurn = -1
		} else if in.Held('>') || in.HasKey(platform.KeyRight) {
			n.keyboardTurn = 1
		}
		if in.HasKey(platform.KeyUp) {
			n.Face = true
		}
		if in.HasKey(platform.KeyDown) {
			n.Face = false
		}
	}
}

// Advance integrates the screen's native rotation rate in radians per second.
func (n *PreviewNavigation) Advance(dt, speed float64) {
	turn := n.turn
	if n.keyboardTurn != 0 {
		turn = n.keyboardTurn
	}
	n.Angle = math.Remainder(n.Angle+turn*speed*max(0, dt), 2*math.Pi)
}

func (n *PreviewNavigation) Draw(s *render.Sprites, x, y int) {
	s.DrawImage(n.bg, float32(x), float32(y))
	for i := range n.buttons {
		b := &n.buttons[i]
		b.OriginX, b.OriginY = x, y
		b.Draw(s)
	}
}
