package ui

import (
	"fmt"

	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

// HUD uses the native status art/layout (0x509f10, 0x5097b0), target
// art/layout (0x50a2e0, 0x50b6a0).
type HUD struct {
	status, bars, circle, portraitMask, target, targetBar, elemental *render.Texture
}

type Resource struct{ Current, Max uint32 }

type HUDState struct {
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
	const x, y = 8, 8
	s.DrawImage(h.status, x, y)
	for i, pool := range []Resource{state.HP, state.MP, state.SP} {
		n := progress(pool.Current, 0, pool.Max)
		s.Draw(h.bars, x+82, float32(y+41+16*i), 114*n, 12, 0, float32(i)*.25, n, float32(i)*.25+12.0/64, white)
	}
	s.DrawImage(h.circle, x, y)
	if state.Portrait != nil {
		s.DrawPortrait(state.Portrait, x+17, y+26, 55, 53)
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
		s.DrawImage(h.target, 256, 8)
		tint := state.TargetColor
		if tint[3] == 0 {
			tint = white
		}
		f.Draw(s, state.TargetName, 358-f.Width(state.TargetName)/2, 26, tint, 170)
		if state.TargetElement >= 1 {
			u := float32((state.TargetElement-1)%4) / 4
			s.Draw(h.elemental, 256, 14, 32, 32, u, 0, u+.25, 1, white)
		}
		if state.TargetKnown {
			n := progress(state.TargetHP.Current, 0, state.TargetHP.Max)
			s.Draw(h.targetBar, 297, 45, 140*n, 16, 0, 0, 140*n/256, 1, white)
			text := fmt.Sprintf("%d/%d", state.TargetHP.Current, state.TargetHP.Max)
			if state.TargetHP.Max == 0 {
				text = fmt.Sprintf("HP %d", state.TargetHP.Current)
			}
			f.Draw(s, text, 384-f.Width(text)/2, 43, white, 140)
		}
	}
}
