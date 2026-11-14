package ui

import "github.com/Spelunkern/Shaiya-GX/internal/render"

var statusLabelTint = [4]float32{.933, .894, .808, 1}

// Shared by the portrait's name and the minimap clock: native warm text with
// the same one-pixel black shadow. Layout and fitting remain with each widget.
func drawStatusText(s *render.Sprites, f *Font, text string, x, y, scale float32, maxW int) {
	f.DrawScaled(s, text, x+1, y+1, scale, [4]float32{0, 0, 0, 1}, maxW)
	f.DrawScaled(s, text, x, y, scale, statusLabelTint, maxW)
}
