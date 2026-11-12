package ui

import "math"

// WindowPosition retains the normalized window center, following Phoenix
// midnight_drag.gd. Native frame dimensions and texture coordinates stay fixed.
type WindowPosition struct {
	CenterX, CenterY float64
	Moved            bool
	Vertical         bool `json:",omitempty"`
}

func (p WindowPosition) Valid() bool {
	return !math.IsNaN(p.CenterX) && !math.IsNaN(p.CenterY) && !math.IsInf(p.CenterX, 0) && !math.IsInf(p.CenterY, 0) && p.CenterX >= 0 && p.CenterX <= 1 && p.CenterY >= 0 && p.CenterY <= 1
}
func (p WindowPosition) Origin(w, h, width, height, x, y int) (int, int) {
	if p.Moved && p.Valid() {
		x = int(math.Round(p.CenterX*float64(w) - float64(width)/2))
		y = int(math.Round(p.CenterY*float64(h) - float64(height)/2))
	}
	return max(0, min(x, w-width)), max(0, min(y, h-height))
}
func (p *WindowPosition) Move(x, y, w, h, width, height int) {
	if w <= 0 || h <= 0 {
		return
	}
	x, y = max(0, min(x, w-width)), max(0, min(y, h-height))
	p.CenterX = min(1, (float64(x)+float64(width)/2)/float64(w))
	p.CenterY = min(1, (float64(y)+float64(height)/2)/float64(h))
	p.Moved = true
}
