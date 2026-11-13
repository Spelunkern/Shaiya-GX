package ui

import (
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"strconv"
)

type ExchangeMoney struct {
	frame     *render.Texture
	ok, close Button
	X, Y      int
	text      string
	replace   bool
}

func NewExchangeMoney(t Textures) (*ExchangeMoney, error) {
	ld := &loader{t: t}
	m := &ExchangeMoney{frame: ld.get("change_money.png")}
	m.ok = Button{Tex: ld.get("change_submit_button.png"), OffX: 125, OffY: 65, DrawW: 128, DrawH: 32, HitW: 65, HitH: 26, UV: [5]UV{{0, 0, .25, 1}, {.25, 0, .5, 1}, {.5, 0, .75, 1}, {.5, 0, .75, 1}, {.75, 0, 1, 1}}}
	m.close = nativeClose(ld.get("talk1_close_button.png"), 174, 4)
	return m, ld.err
}
func (m *ExchangeMoney) Open(w, h int, current uint32) {
	m.X = max(0, (w-200)/2)
	m.Y = max(0, (h-104)/2)
	m.text = strconv.FormatUint(uint64(current), 10)
	m.replace = true
}

// done=false while editing; cancel is distinct from accepting zero.
func (m *ExchangeMoney) Update(in *platform.Input) (value uint64, done, cancel bool) {
	m.ok.OriginX, m.ok.OriginY = m.X, m.Y
	m.close.OriginX, m.close.OriginY = m.X, m.Y
	if in.HasKey(platform.KeyEscape) || m.close.Update(in) {
		return 0, true, true
	}
	if in.HasKey(platform.KeyBackspace) {
		if m.replace {
			m.text = ""
		} else if len(m.text) > 0 {
			m.text = m.text[:len(m.text)-1]
		}
		m.replace = false
	}
	if in.HasKey(platform.KeyDelete) {
		m.text = ""
		m.replace = false
	}
	for _, r := range in.Chars {
		if r < '0' || r > '9' {
			continue
		}
		if m.replace {
			m.text = ""
			m.replace = false
		}
		if len(m.text) < 10 {
			m.text += string(r)
		}
	}
	if in.HasKey(platform.KeyEnter) || m.ok.Update(in) {
		if m.text != "" {
			value, _ = strconv.ParseUint(m.text, 10, 64)
		}
		return value, true, false
	}
	return 0, false, false
}
func (m *ExchangeMoney) Draw(s *render.Sprites, f *Font) {
	s.Draw(m.frame, float32(m.X), float32(m.Y), 200, 104, 0, 0, 200.0/256, 104.0/128, white)
	f.Draw(s, m.text+"_", m.X+28, m.Y+36, white, 102)
	m.ok.OriginX, m.ok.OriginY = m.X, m.Y
	m.close.OriginX, m.close.OriginY = m.X, m.Y
	m.ok.Draw(s)
	m.close.Draw(s)
}
