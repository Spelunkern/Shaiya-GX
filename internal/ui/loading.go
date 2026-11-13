package ui

import (
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"math"
)

// LoadingScreen follows Phoenix loading_screen_hud.gd: aspect-fit art, a
// 552x156 frame, clipped 430x18 fill and a 40x53 highlight, 24px from bottom.
type LoadingScreen struct{ art, frame, fill, light *render.Texture }

func NewLoadingScreen(t Textures) *LoadingScreen {
	l := &LoadingScreen{}
	l.art, _ = t.Interface("random_loading01.png")
	l.frame, _ = t.Interface("loadbar_bg.png")
	l.fill, _ = t.Interface("loadbar.png")
	l.light, _ = t.Interface("loadbar_light.png")
	return l
}
func (l *LoadingScreen) Draw(s *render.Sprites, f *Font, solid *render.Texture, w, h int, progress float32, status string) {
	white := [4]float32{1, 1, 1, 1}
	s.Draw(solid, 0, 0, float32(w), float32(h), 0, 0, 1, 1, [4]float32{.02, .02, .025, 1})
	if l.art != nil {
		scale := min(float32(w)/float32(l.art.W), float32(h)/float32(l.art.H))
		aw, ah := float32(l.art.W)*scale, float32(l.art.H)*scale
		s.Draw(l.art, (float32(w)-aw)/2, (float32(h)-ah)/2, aw, ah, 0, 0, 1, 1, white)
	}
	scale := max(.1, min(float32(1), min(float32(w-32)/552, float32(h-32)/180)))
	x, y := (float32(w)-552*scale)/2, float32(h)-180*scale
	draw := func(t *render.Texture, px, py, sw, sh, visible float32) {
		if t != nil && visible > 0 {
			s.Draw(t, x+px*scale, y+py*scale, sw*scale*visible, sh*scale, 0, 0, sw*visible/float32(t.W), sh/float32(t.H), white)
		}
	}
	draw(l.frame, 0, 0, 552, 156, 1)
	p := max(float32(0), min(float32(1), progress))
	if math.IsNaN(float64(p)) {
		p = 0
	}
	if l.fill != nil {
		draw(l.fill, 58, 31, 430, 18, p)
	} else {
		s.Draw(solid, x+58*scale, y+31*scale, 430*scale, 18*scale, 0, 0, 1, 1, [4]float32{.1, .1, .1, 1})
		s.Draw(solid, x+58*scale, y+31*scale, 430*scale*p, 18*scale, 0, 0, 1, 1, [4]float32{.8, .1, .05, 1})
	}
	if p > 0 {
		draw(l.light, 58+430*p-20, 31+9-26.5, 40, 53, 1)
	}
	f.DrawOutlined(s, status, int(x+276*scale)-f.Width(status)/2, int(y+68*scale), [4]float32{.98, .93, .8, 1})
}
