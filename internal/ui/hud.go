package ui

import (
	"fmt"
	"math"

	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

// HUD uses the native status art/layout (0x509f10, 0x5097b0), target
// art/layout (0x50a2e0, 0x50b6a0).
type HUD struct {
	Position, TargetPosition                                         WindowPosition
	status, bars, circle, portraitMask, target, targetBar, elemental *render.Texture
}

type Resource struct{ Current, Max uint32 }

type HUDState struct {
	ViewportW, ViewportH  int
	Name                  string
	Level                 uint16
	HP, MP, SP            Resource
	Exp, PrevExp, NextExp uint32
	BlessCount, BlessMax  uint32
	BlessKnown            bool
	TargetName            string
	TargetHP              Resource
	TargetKnown           bool
	TargetColor           [4]float32
	TargetElement         uint8
	Portrait              *render.Texture
}

func NewHUD(t Textures) (*HUD, error) {
	h := &HUD{}
	for _, asset := range []struct {
		name string
		dst  **render.Texture
	}{
		{"main_stats_bar_bg.png", &h.status}, {"main_stats_bar.png", &h.bars},
		{"main_stats_circle.png", &h.circle}, {"enemy_bar_bg.png", &h.target},
		{"main_stats_circle2.png", &h.portraitMask},
		{"enemy_bar.png", &h.targetBar},
		{"enemy_elemental.png", &h.elemental},
	} {
		tex, err := t.Interface(asset.name)
		if err != nil {
			return nil, err
		}
		*asset.dst = tex
	}
	return h, nil
}

// progress clamps server values and avoids unsigned subtraction underflow.
func progress(value, start, end uint32) float32 {
	if end <= start || value <= start {
		return 0
	}
	if value >= end {
		return 1
	}
	return float32(float64(value-start) / float64(end-start))
}

func (h *HUD) Draw(s *render.Sprites, f *Font, state HUDState) {
	// Phoenix ep4_ui.gd::player: root (8,8), fixed label rectangles.
	x, y := h.Origin(state.ViewportW, state.ViewportH)
	tx, ty := h.TargetOrigin(state.ViewportW, state.ViewportH)
	s.DrawImage(h.status, float32(x), float32(y))
	for i, pool := range []Resource{state.HP, state.MP, state.SP} {
		n := progress(pool.Current, 0, pool.Max)
		s.Draw(h.bars, float32(x+82), float32(y+41+16*i), 114*n, 12, 0, float32(i)*.25, n, float32(i)*.25+12.0/64, white)
	}
	s.DrawImage(h.circle, float32(x), float32(y))
	if state.Portrait != nil {
		s.DrawPortrait(state.Portrait, float32(x+17), float32(y+26), 55, 53)
	}
	tint := [4]float32{.933, .894, .808, 1}
	shadow := [4]float32{0, 0, 0, 1}
	label := func(text string, left, top, width, height int, center bool) {
		if center {
			left += (width - f.Width(text)) / 2
		}
		top += (height - f.Height) / 2
		f.Draw(s, text, left+1, top+1, shadow, width)
		f.Draw(s, text, left, top, tint, width)
	}
	label(state.Name, x+88, y+14, 109, 16, false)
	label(fmt.Sprint(state.Level), x+62, y+15, 22, 20, true)
	for i, pool := range []Resource{state.HP, state.MP, state.SP} {
		label(fmt.Sprintf("%d/%d", pool.Current, pool.Max), x+81, y+36+16*i, 116, 20, true)
	}

	if state.TargetName != "" {
		s.DrawImage(h.target, float32(tx), float32(ty+8))
		tint := state.TargetColor
		if tint[3] == 0 {
			tint = white
		}
		f.Draw(s, state.TargetName, tx+102-f.Width(state.TargetName)/2, ty+26, tint, 170)
		if state.TargetElement >= 1 {
			u := float32((state.TargetElement-1)%4) / 4
			s.Draw(h.elemental, float32(tx), float32(ty+14), 32, 32, u, 0, u+.25, 1, white)
		}
		if state.TargetKnown {
			n := progress(state.TargetHP.Current, 0, state.TargetHP.Max)
			s.Draw(h.targetBar, float32(tx+41), float32(ty+45), 140*n, 16, 0, 0, 140*n/256, 1, white)
			text := fmt.Sprintf("%d/%d", state.TargetHP.Current, state.TargetHP.Max)
			if state.TargetHP.Max == 0 {
				text = fmt.Sprintf("HP %d", state.TargetHP.Current)
			}
			f.Draw(s, text, tx+128-f.Width(text)/2, ty+43, white, 140)
		}
	}
}

func (h *HUD) Origin(w, height int) (int, int) {
	if w <= 0 || height <= 0 {
		return 8, 8
	}
	return h.Position.Origin(w, height, 213, 114, 8, 8)
}
func (h *HUD) TargetOrigin(w, height int) (int, int) {
	if w <= 0 || height <= 0 {
		return 256, 0
	}
	return h.TargetPosition.Origin(w, height, 256, 64, 256, 0)
}

// DrawCast follows Game.exe 0x004a3020: a 5x64 black rectangle and a
// 3x62 blue elapsed-progress column. No texture, spell name or timer text.
func (h *HUD) DrawCast(s *render.Sprites, solid *render.Texture, x, y int, left, total float32) {
	fill, ok := castFillHeight(left, total)
	if !ok {
		return
	}
	s.Draw(solid, float32(x), float32(y), 5, 64, 0, 0, 1, 1, [4]float32{0, 0, 0, 1})
	s.Draw(solid, float32(x+1), float32(y+1), 3, float32(fill), 0, 0, 1, 1, [4]float32{0, 0, 1, 1})
}

func castFillHeight(left, total float32) (int, bool) {
	if left <= 0 || total <= 0 || math.IsNaN(float64(left)) || math.IsNaN(float64(total)) || math.IsInf(float64(left), 0) || math.IsInf(float64(total), 0) {
		return 0, false
	}
	return int(math.RoundToEven((1 - min(float64(left)/float64(total), 1)) * 62)), true
}
