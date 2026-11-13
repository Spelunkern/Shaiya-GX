package ui

import "github.com/Spelunkern/Shaiya-GX/internal/render"

// 0x47de10: text starts at (10,10), rows advance 15px. 0x47e1e0
// uses a 90px content width. Longer translated labels may widen it.
const PlayerMenuRowHeight = 15
const PlayerMenuTextInset = 10

type PlayerMenu struct{ frame *render.Texture }

func NewPlayerMenu(t Textures) (*PlayerMenu, error) {
	frame, err := t.Interface("TipBox.TGA")
	return &PlayerMenu{frame: frame}, err
}

func PlayerMenuSize(f *Font, labels []string) (int, int) {
	width := 98 // 90px native content plus the 4px borders.
	if f != nil {
		for _, label := range labels {
			width = max(width, f.Width(label)+20)
		}
	}
	return width, len(labels)*PlayerMenuRowHeight + 32
}

func PlayerMenuRowAt(mx, my, x, y, width, count int) int {
	if mx < x+4 || mx >= x+width-4 || my < y+PlayerMenuTextInset || my >= y+PlayerMenuTextInset+count*PlayerMenuRowHeight {
		return -1
	}
	return (my - y - PlayerMenuTextInset) / PlayerMenuRowHeight
}

// Classic branch of 0x50ff10. The optional RightMenu_* skin is absent
// from this archive; the original falls back to the TipBox atlas.
func (p *PlayerMenu) Draw(s *render.Sprites, f *Font, solid *render.Texture, x, y, width int, labels []string, hover int) {
	_, height := PlayerMenuSize(f, labels)
	contentW, contentH := width-8, height-12
	s.Draw(solid, float32(x+4), float32(y+4), float32(contentW), float32(contentH+4), 0, 0, 1, 1, [4]float32{29.0 / 255, 31.0 / 255, 36.0 / 255, 205.0 / 255})
	draw := func(dx, dy, w, h int, u0, v0, u1, v1 float32) {
		s.Draw(p.frame, float32(x+dx), float32(y+dy), float32(w), float32(h), u0, v0, u1, v1, white)
	}
	draw(0, 0, 4, 4, 0, 0, .25, .25)
	draw(4, 0, contentW, 4, .25, 0, .5, .25)
	draw(width-4, 0, 4, 4, .5, 0, .75, .25)
	draw(0, 4, 4, contentH+4, .25, .25, .5, .5)
	draw(width-4, 4, 4, contentH+4, .75, .25, 1, .5)
	draw(0, height-4, 4, 4, 0, .25, .25, .5)
	draw(4, height-4, contentW, 4, .75, 0, 1, .25)
	draw(width-4, height-4, 4, 4, .5, .25, .75, .5)
	for i, label := range labels {
		tint := white
		if i == hover {
			tint = RGBColor(0x0000ff)
		} // native packed 0xff0000ff
		f.Draw(s, label, x+PlayerMenuTextInset, y+PlayerMenuTextInset+i*PlayerMenuRowHeight, tint, 0)
	}
}
