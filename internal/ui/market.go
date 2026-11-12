package ui

import (
	"fmt"

	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

const MarketWidth, MarketHeight, MarketPageSize = 256, 344, 30

// Default anchor before the player moves the frame.
func MarketOrigin(w, h int) (int, int) {
	return min(32, max(0, w-MarketWidth)), min(82, max(0, h-MarketHeight))
}

// Game.exe 0x508a20: six columns, five rows, (15,112), pitch 39.
func MarketCell(slot int) (x, y int, ok bool) {
	if slot < 0 || slot >= MarketPageSize {
		return
	}
	return 15 + slot%6*39, 112 + slot/6*39, true
}
func MarketHit(x, y int) (int, bool) {
	for i := 0; i < MarketPageSize; i++ {
		cx, cy, _ := MarketCell(i)
		if x >= cx && x < cx+32 && y >= cy && y < cy+32 {
			return i, true
		}
	}
	return 0, false
}

// Market owns the original art and controls (0x5080a0), not transactions.
type Market struct {
	Position WindowPosition
	frame    *render.Texture
	arrows   [2]Button
	close    Button
}

func NewMarket(t Textures) (*Market, error) {
	ld := &loader{t: t}
	m := &Market{frame: ld.get("market.tga")}
	tex := ld.get("market_button.tga")
	for i := range m.arrows {
		b := &m.arrows[i]
		*b = Button{Tex: tex, OffX: 95 + i*50, OffY: 308, DrawW: 32, DrawH: 32, HitW: 22, HitH: 22}
		for state, row := range []int{0, 1, 2, 2, 3} {
			b.UV[state] = UV{float32(i) * .5, float32(row) * .25, float32(i+1) * .5, float32(row+1) * .25}
		}
	}
	// Common native close art; placement follows the auxiliary Phoenix reference.
	m.close = nativeClose(ld.get("talk1_close_button.tga"), 228, 5)
	return m, ld.err
}
func (m *Market) controls(w, h, page, count int, busy bool) {
	x, y := m.Origin(w, h)
	m.close.OriginX, m.close.OriginY = x, y
	for i := range m.arrows {
		b := &m.arrows[i]
		b.OriginX, b.OriginY = x, y
		b.Disabled = busy || (i == 0 && page == 0) || (i == 1 && (page+1)*MarketPageSize >= count)
	}
}
func (m *Market) Update(in *platform.Input, w, h, page, count int, busy bool) (int, bool) {
	page = max(0, min(page, max(0, (count-1)/MarketPageSize)))
	m.controls(w, h, page, count, busy)
	if m.close.Update(in) {
		return page, true
	}
	for i := range m.arrows {
		if m.arrows[i].Update(in) {
			page += i*2 - 1
		}
	}
	return page, false
}
func (m *Market) Draw(s *render.Sprites, f *Font, w, h, page, count int, busy bool, name string, portrait *render.Texture) {
	m.controls(w, h, page, count, busy)
	x, y := m.Origin(w, h)
	s.Draw(m.frame, float32(x), float32(y), 256, 344, 0, 0, 1, 344.0/512, white)
	if portrait != nil {
		s.Draw(portrait, float32(x+7), float32(y+7), 64, 64, 0, 0, 1, 1, white)
	}
	f.Draw(s, name, x+77, y+22, white, 145)
	for i := range m.arrows {
		m.arrows[i].Draw(s)
	}
	m.close.Draw(s)
	text := fmt.Sprintf("%d/%d", page+1, max(1, (count+MarketPageSize-1)/MarketPageSize))
	f.Draw(s, text, x+133-f.Width(text)/2, y+313, white, 34)
}

// Native sale drop area, including gaps between product cells (0x5082d0).
func MarketDropHit(x, y int) bool { return x >= 15 && x <= 245 && y >= 112 && y <= 305 }

func (m *Market) Origin(w, h int) (int, int) {
	x, y := MarketOrigin(w, h)
	if m == nil {
		return x, y
	}
	return m.Position.Origin(w, h, MarketWidth, MarketHeight, x, y)
}
