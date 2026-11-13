package ui

import (
	"fmt"
	"math"

	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

const MallWidth, MallHeight = 580, 446
const GiftBoxWidth, GiftBoxHeight = 333, 315

type MallProduct struct {
	Name, Code, Description string
	Cost                    uint32
	Icon                    *render.Texture
	UV                      UV
}
type MallIdentity struct{ Name, Server, Class, Mode string }

type MallAction uint8

const (
	MallNone MallAction = iota
	MallClose
	MallBuy
	MallGift
	MallShowGifts
	MallCloseGifts
)

// Mall uses the original 1024x512 and 512x512 EP5 interface sheets.
type Mall struct {
	Position, GiftPosition                       WindowPosition
	frame, giftFrame, blank, scrollbar           *render.Texture
	tabs                                         [4]Button
	buy, gift, giftBox, close, cancel, giftClose Button
	arrows                                       [2]Button
	scrolling                                    bool
	Category, Scroll, Selected, GiftPage         int
	GiftOpen                                     bool
	heldProduct, heldGift, heldTab               int
	selectedGift                                 int
	lastGift                                     int
	lastGiftClick                                float64
}

func NewMall(t Textures, blank *render.Texture) (*Mall, error) {
	ld := &loader{t: t}
	m := &Mall{frame: ld.get("ins_market.png"), giftFrame: ld.get("ins_market_store.png"), scrollbar: ld.get("scrollbar.png"), blank: blank, Selected: -1, heldProduct: -1, heldGift: -1, heldTab: -1, selectedGift: -1, lastGift: -1}
	for i := range m.arrows {
		b := &m.arrows[i]
		*b = Button{Tex: m.scrollbar, OffX: 565, OffY: 164 + i*176, DrawW: 16, DrawH: 32, HitW: 13, HitH: 19}
		for state, col := range []int{0, 1, 1, 1, 2} {
			b.UV[state] = UV{float32(col) / 8, float32(i) / 2, float32(col+1) / 8, float32(i+1) / 2}
		}
	}
	tabs := ld.get("ins_market_menu_button.png")
	for i := range m.tabs {
		b := &m.tabs[i]
		*b = Button{Tex: tabs, OffX: 16 + i*128, OffY: 130, DrawW: 128, DrawH: 32, HitW: 96, HitH: 26, Toggle: true}
		for state, row := range []int{0, 1, 2, 1, 3} {
			b.UV[state] = UV{float32(i) / 4, float32(row) / 4, float32(i+1) / 4, float32(row+1) / 4}
		}
	}
	tex := ld.get("ins_market_button.png")
	for i, b := range []*Button{&m.buy, &m.gift} {
		*b = Button{Tex: tex, OffX: 287 + i*101, OffY: 401, DrawW: 128, DrawH: 32, HitW: 86, HitH: 26}
		for state, row := range []int{0, 1, 2, 2, 3} {
			b.UV[state] = UV{float32(i) / 2, float32(row) / 8, float32(i+1) / 2, float32(row+1) / 8}
		}
	}
	m.giftBox = Button{Tex: ld.get("ins_market_store_button.png"), OffX: 24, OffY: 401, DrawW: 128, DrawH: 32, HitW: 107, HitH: 26, UV: FourRowUV}
	m.cancel = Button{Tex: ld.get("cancel_button.png"), OffX: 489, OffY: 401, DrawW: 128, DrawH: 32, HitW: 65, HitH: 25, UV: FourRowUV}
	closeTex := ld.get("talk1_close_button.png")
	m.close = nativeClose(closeTex, 554, 3)
	m.giftClose = nativeClose(closeTex, 307, 4)
	return m, ld.err
}
func (m *Mall) Origin(w, h int) (int, int) {
	return m.Position.Origin(w, h, MallWidth, MallHeight, (w-MallWidth)/2, (h-MallHeight)/2)
}
func (m *Mall) GiftOrigin(w, h int) (int, int) {
	return m.GiftPosition.Origin(w, h, GiftBoxWidth, GiftBoxHeight, (w-GiftBoxWidth)/2, (h-GiftBoxHeight)/2)
}
func (m *Mall) Reset() {
	m.Category, m.Scroll, m.Selected, m.GiftPage = 0, 0, -1, 0
	m.GiftOpen = false
	m.scrolling = false
	m.heldProduct, m.heldGift, m.heldTab, m.selectedGift, m.lastGift = -1, -1, -1, -1, -1
}

func MallProductRect(index int) (int, int) { return 19 + (index%2)*270, 164 + (index/2)*65 }
func MallGiftCell(slot int) (int, int)     { return 10 + (slot%40)%8*39, 78 + (slot%40)/8*39 }

func (m *Mall) ProductAt(w, h, mouseX, mouseY, count int) int {
	x, y := m.Origin(w, h)
	if mouseX < x+16 || mouseX >= x+556 || mouseY < y+164 || mouseY >= y+359 {
		return -1
	}
	col, row := (mouseX-x-16)/270, (mouseY-y-164)/65
	index := (m.Scroll+row)*2 + col
	if col > 1 || index >= count {
		return -1
	}
	return index
}

func (m *Mall) Update(in *platform.Input, w, h, count int, busy bool) (MallAction, bool) {
	x, y := m.Origin(w, h)
	for i := range m.tabs {
		b := &m.tabs[i]
		b.OriginX, b.OriginY, b.Checked = x, y, i == m.Category
		if b.Update(in) {
			m.Category, m.Scroll, m.Selected = i, 0, -1
		}
	}
	m.close.OriginX, m.close.OriginY = x, y
	m.cancel.OriginX, m.cancel.OriginY = x, y
	if m.close.Update(in) || m.cancel.Update(in) || in.HasKey(platform.KeyEscape) {
		return MallClose, true
	}
	m.giftBox.OriginX, m.giftBox.OriginY = x, y
	if m.giftBox.Update(in) {
		m.GiftOpen = true
		return MallShowGifts, true
	}
	for i, b := range []*Button{&m.buy, &m.gift} {
		b.OriginX, b.OriginY, b.Disabled = x, y, busy || m.Selected < 0 || m.Selected >= count
		if b.Update(in) {
			if i == 0 {
				return MallBuy, true
			}
			return MallGift, true
		}
	}
	insideList := in.MouseX >= x+16 && in.MouseX < x+556 && in.MouseY >= y+164 && in.MouseY < y+359
	maxScroll := max(0, (count+1)/2-3)
	for i := range m.arrows {
		b := &m.arrows[i]
		b.OriginX, b.OriginY, b.Disabled = x, y, maxScroll == 0 || i == 0 && m.Scroll == 0 || i == 1 && m.Scroll >= maxScroll
		if b.Update(in) {
			m.Scroll = max(0, min(maxScroll, m.Scroll+i*2-1))
		}
	}
	if maxScroll > 0 && in.Pressed && in.MouseX >= x+565 && in.MouseX < x+580 && in.MouseY >= y+184 && in.MouseY < y+340 {
		m.scrolling = true
	}
	if m.scrolling {
		if in.MouseDown || in.Clicked {
			m.Scroll = max(0, min(maxScroll, int(math.Round(float64(in.MouseY-y-184)*float64(maxScroll)/148))))
		}
		if in.Clicked || !in.MouseDown {
			m.scrolling = false
		}
	}
	if insideList && in.Wheel != 0 {
		if in.Wheel > 0 {
			m.Scroll--
		} else {
			m.Scroll++
		}
		m.Scroll = max(0, min(m.Scroll, maxScroll))
	}
	hit := -1
	if insideList {
		col := (in.MouseX - x - 16) / 270
		row := (in.MouseY - y - 164) / 65
		hit = (m.Scroll+row)*2 + col
		if col > 1 || hit >= count {
			hit = -1
		}
	}
	if in.Pressed {
		m.heldProduct = hit
	}
	if in.Clicked {
		if hit >= 0 && hit == m.heldProduct {
			m.Selected = hit
		}
		m.heldProduct = -1
	}
	inside := in.MouseX >= x && in.MouseX < x+MallWidth && in.MouseY >= y && in.MouseY < y+MallHeight
	return MallNone, m.scrolling || inside && (in.Pressed || in.Clicked || in.MouseDown || in.RightPressed || in.RightDown || in.Wheel != 0)
}

func (m *Mall) Draw(s *render.Sprites, f *Font, w, h int, products []MallProduct, identity MallIdentity, points uint32, pointsReady, busy bool) {
	x, y := m.Origin(w, h)
	s.Draw(m.frame, float32(x), float32(y), MallWidth, MallHeight, 0, 0, float32(MallWidth)/1024, float32(MallHeight)/512, white)
	f.Draw(s, identity.Name, x+52, y+53, white, 165)
	f.Draw(s, identity.Server, x+52, y+76, white, 165)
	f.Draw(s, identity.Class, x+52, y+99, white, 59)
	f.Draw(s, identity.Mode, x+158, y+99, white, 60)
	pointText := "..."
	if pointsReady {
		pointText = fmt.Sprint(points)
	}
	f.Draw(s, pointText, x+454, y+53, white, 100)
	for i := range m.tabs {
		b := &m.tabs[i]
		b.OriginX, b.OriginY, b.Checked = x, y, i == m.Category
		b.Draw(s)
	}
	for i := 0; i < 6; i++ {
		index := m.Scroll*2 + i
		if index >= len(products) {
			break
		}
		p := products[index]
		dx, dy := MallProductRect(i)
		if index == m.Selected && m.blank != nil {
			s.Draw(m.blank, float32(x+dx), float32(y+dy), 258, 60, 0, 0, 1, 1, [4]float32{.65, .53, .3, .28})
		}
		if p.Icon != nil {
			s.Draw(p.Icon, float32(x+dx+4), float32(y+dy+6), 32, 32, p.UV.U0, p.UV.V0, p.UV.U1, p.UV.V1, white)
		}
		f.Draw(s, p.Name, x+dx+43, y+dy+5, white, 207)
		f.Draw(s, fmt.Sprintf("%d points", p.Cost), x+dx+43, y+dy+24, [4]float32{.95, .8, .45, 1}, 190)
	}
	for _, b := range []*Button{&m.buy, &m.gift} {
		b.OriginX, b.OriginY, b.Disabled = x, y, busy || m.Selected < 0 || m.Selected >= len(products)
		b.Draw(s)
	}
	for _, b := range []*Button{&m.giftBox, &m.cancel, &m.close} {
		b.OriginX, b.OriginY = x, y
		b.Draw(s)
	}
	last := max(0, (len(products)+1)/2-3)
	for i := range m.arrows {
		b := &m.arrows[i]
		b.OriginX, b.OriginY, b.Disabled = x, y, last == 0 || i == 0 && m.Scroll == 0 || i == 1 && m.Scroll >= last
		b.Draw(s)
	}
	thumb := 184
	if last > 0 {
		thumb += m.Scroll * 148 / last
	}
	s.Draw(m.scrollbar, float32(x+565), float32(y+thumb), 13, 8, .5, 0, .5+13.0/128, 8.0/64, white)
}

// Gift box uses six pages of 40 slots; only a double click or right click claims.
func (m *Mall) UpdateGifts(in *platform.Input, w, h int, stored map[uint8]proto.MallStoredItem, now float64, busy bool) (int, MallAction, bool) {
	x, y := m.GiftOrigin(w, h)
	m.giftClose.OriginX, m.giftClose.OriginY = x, y
	if m.giftClose.Update(in) || in.HasKey(platform.KeyEscape) {
		m.GiftOpen = false
		return -1, MallCloseGifts, true
	}
	tab := -1
	if in.MouseX >= x+10 && in.MouseX < x+322 && in.MouseY >= y+54 && in.MouseY < y+77 {
		tab = (in.MouseX - x - 10) / 52
	}
	hit := -1
	for i := 0; i < 40; i++ {
		dx, dy := MallGiftCell(i)
		if in.MouseX >= x+dx && in.MouseX < x+dx+32 && in.MouseY >= y+dy && in.MouseY < y+dy+32 {
			slot := m.GiftPage*40 + i
			if _, ok := stored[uint8(slot)]; ok {
				hit = slot
			}
			break
		}
	}
	if in.Pressed {
		m.heldGift, m.heldTab = hit, tab
	}
	claim := -1
	if in.RightPressed && hit >= 0 && !busy {
		claim = hit
	}
	if in.Clicked {
		if tab >= 0 && m.heldTab == tab {
			m.GiftPage, m.selectedGift = tab, -1
		}
		if hit >= 0 && m.heldGift == hit {
			if hit == m.lastGift && now >= m.lastGiftClick && now-m.lastGiftClick <= .35 && !busy {
				claim = hit
				m.lastGift = -1
			} else {
				m.selectedGift, m.lastGift, m.lastGiftClick = hit, hit, now
			}
		}
		m.heldGift, m.heldTab = -1, -1
	}
	inside := in.MouseX >= x && in.MouseX < x+GiftBoxWidth && in.MouseY >= y && in.MouseY < y+GiftBoxHeight
	return claim, MallNone, inside && (in.Pressed || in.Clicked || in.MouseDown || in.RightPressed || in.RightDown || in.Wheel != 0)
}
func (m *Mall) DrawGifts(s *render.Sprites, f *Font, w, h int, stored map[uint8]proto.MallStoredItem, icon func(proto.MallStoredItem) (*render.Texture, UV, bool)) {
	x, y := m.GiftOrigin(w, h)
	s.Draw(m.giftFrame, float32(x), float32(y), GiftBoxWidth, GiftBoxHeight, 0, 0, float32(GiftBoxWidth)/512, float32(GiftBoxHeight)/512, white)
	if m.blank != nil {
		s.Draw(m.blank, float32(x+10+m.GiftPage*52), float32(y+54), 50, 22, 0, 0, 1, 1, [4]float32{.8, .65, .35, .25})
	}
	for i := 0; i < 40; i++ {
		slot := m.GiftPage*40 + i
		it, ok := stored[uint8(slot)]
		if !ok {
			continue
		}
		dx, dy := MallGiftCell(i)
		if slot == m.selectedGift && m.blank != nil {
			s.Draw(m.blank, float32(x+dx), float32(y+dy), 32, 32, 0, 0, 1, 1, [4]float32{.8, .65, .35, .4})
		}
		if tex, uv, ok := icon(it); ok && tex != nil {
			s.Draw(tex, float32(x+dx), float32(y+dy), 32, 32, uv.U0, uv.V0, uv.U1, uv.V1, white)
		}
		if it.Count > 1 {
			f.Draw(s, fmt.Sprint(it.Count), x+dx+12, y+dy+19, white, 20)
		}
	}
	f.Draw(s, "Right-click to claim", x+14, y+282, [4]float32{.8, .8, .8, 1}, 240)
	m.giftClose.OriginX, m.giftClose.OriginY = x, y
	m.giftClose.Draw(s)
}
