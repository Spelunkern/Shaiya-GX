package ui

import (
	"fmt"

	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

const GuildWidth, GuildHeight, GuildRows = 337, 512, 15

// GuildPanel follows 0x4f2960/0x4ef9d0, using the original single roster sheet.
type GuildPanel struct {
	Page, navHeld           int
	Position                WindowPosition
	frame                   *render.Texture
	waitFrame               *render.Texture
	headers                 [5]*render.Texture
	waitHeaders             [3]*render.Texture
	close                   Button
	actions                 [3]Button
	waitActions             [2]Button
	toggle, back            Button
	notice                  Button
	held                    int
	scroll                  panelScroll
	headerButtons           [5]Button
	SortColumn              int
	Descending, SortChanged bool
	ClassNames              [6]string
	nav                     *render.Texture
	hoverRow                int
}

func NewGuildPanel(t Textures) (*GuildPanel, error) {
	ld := &loader{t: t}
	p := &GuildPanel{frame: ld.get("guild.tga"), held: -1, navHeld: -1, hoverRow: -1, SortColumn: 3}
	p.waitFrame = ld.get("guild_wait.tga")
	p.nav = ld.get("friend_left_nav.tga")
	p.scroll = newPanelScroll(ld.get("scrollbar.tga"), 303, 99, 356)
	for i, name := range []string{"guild_sort_lv.tga", "guild_sort_name.tga", "guild_sort_job.tga", "guild_sort_grade.tga", "guild_sort_on.tga"} {
		p.headers[i] = ld.get(name)
		left := [5]int{36, 62, 184, 247, 276}[i]
		width := [5]int{25, 121, 61, 27, 37}[i]
		p.headerButtons[i] = Button{Tex: p.headers[i], OffX: left, OffY: 70, DrawW: []int{32, 128, 64, 32, 64}[i], DrawH: 32, HitW: width, HitH: 29, UV: FourRowUV}
	}
	for i, name := range []string{"guild_wait_lv.tga", "guild_wait_name.tga", "guild_wait_job.tga"} {
		p.waitHeaders[i] = ld.get(name)
	}
	p.close = nativeClose(ld.get("talk1_close_button.tga"), 307, 8)
	buttons := ld.get("guild_button1.tga")
	for i := range p.actions {
		b := &p.actions[i]
		*b = Button{Tex: buttons, OffX: 37 + 69*i, OffY: 466, DrawW: 128, DrawH: 32, HitW: 65, HitH: 25}
		for state, col := range []int{0, 1, 2, 2, 3} {
			b.UV[state] = UV{float32(col) / 4, float32(i) / 4, float32(col+1) / 4, float32(i+1) / 4}
		}
	}
	waitButtons := ld.get("guild_wait_button.tga")
	for i := range p.waitActions {
		b := &p.waitActions[i]
		*b = Button{Tex: waitButtons, OffX: 41 + 72*i, OffY: 466, DrawW: 128, DrawH: 32, HitW: 65, HitH: 25}
		for state, col := range []int{0, 1, 2, 2, 3} {
			b.UV[state] = UV{float32(col) / 4, float32(i) / 2, float32(col+1) / 4, float32(i+1) / 2}
		}
	}
	p.toggle = Button{Tex: ld.get("guild_button2.tga"), OffX: 237, OffY: 47, DrawW: 128, DrawH: 32, HitW: 75, HitH: 21, UV: FourRowUV}
	p.back = Button{Tex: ld.get("guild_wait_button2.tga"), OffX: 237, OffY: 47, DrawW: 128, DrawH: 32, HitW: 75, HitH: 21, UV: FourRowUV}
	p.notice = Button{Tex: buttons, OffX: 249, OffY: 466, DrawW: 128, DrawH: 32, HitW: 65, HitH: 25}
	for state, col := range []int{0, 1, 2, 2, 3} {
		p.notice.UV[state] = UV{float32(col) / 4, .75, float32(col+1) / 4, 1}
	}
	return p, ld.err
}

func (p *GuildPanel) UpdateNotice(in *platform.Input, w, h int, enabled bool) bool {
	x, y := p.Origin(w, h)
	p.notice.OriginX, p.notice.OriginY, p.notice.Disabled = x, y, !enabled
	return p.notice.Update(in)
}

func (p *GuildPanel) DrawNotice(s *render.Sprites, f *Font, blank *render.Texture, w, h int, enabled bool, etin uint32, etinKnown bool) {
	x, y := p.Origin(w, h)
	p.notice.OriginX, p.notice.OriginY, p.notice.Disabled = x, y, !enabled
	p.notice.Draw(s)
}

func (p *GuildPanel) Origin(w, h int) (int, int) {
	return p.Position.Origin(w, h, GuildWidth, GuildHeight, 32, max(0, (h-GuildHeight)/2))
}

func (p *GuildPanel) Update(in *platform.Input, w, h, count, offset int, waiting, canToggle bool, allowed [3]bool) (row, scroll, action int, toggle, close bool) {
	x, y := p.Origin(w, h)
	p.SortChanged = false
	if !waiting {
		for i := range p.headerButtons {
			b := &p.headerButtons[i]
			b.OriginX, b.OriginY = x, y
			if b.Update(in) {
				p.Descending = p.SortColumn == i && !p.Descending
				p.SortColumn = i
				p.SortChanged = true
			}
		}
	}
	p.close.OriginX, p.close.OriginY = x, y
	close = p.close.Update(in)
	row, scroll, action = -1, max(0, min(offset, max(0, count-GuildRows))), -1
	buttonSet := p.actions[:]
	toggleButton := &p.toggle
	if waiting {
		buttonSet, toggleButton = p.waitActions[:], &p.back
	}
	toggleButton.OriginX, toggleButton.OriginY, toggleButton.Disabled = x, y, !canToggle
	toggle = toggleButton.Update(in)
	for i := range buttonSet {
		b := &buttonSet[i]
		b.OriginX, b.OriginY, b.Disabled = x, y, !allowed[i]
		if b.Update(in) {
			action = i
		}
	}
	hit := -1
	if in.MouseX >= x+37 && in.MouseX < x+303 && in.MouseY >= y+112 && in.MouseY < y+112+GuildRows*23 {
		if in.Wheel != 0 {
			delta := 1
			if in.Wheel < 0 {
				delta = -1
			}
			scroll = max(0, min(max(0, count-GuildRows), scroll-delta))
		}
		hit = (in.MouseY - y - 112) / 23
	}
	scroll = p.scroll.Update(in, x, y, count, GuildRows, scroll)
	p.hoverRow = hit
	if in.Pressed {
		p.held = hit
	}
	if in.Clicked {
		if hit >= 0 && hit == p.held && scroll+hit < count {
			row = scroll + hit
		}
		p.held = -1
	}
	return
}

func (p *GuildPanel) Draw(s *render.Sprites, f *Font, blank *render.Texture, w, h int, name string, members []proto.GuildMember, offset int, selected uint32, waiting, canToggle bool, allowed [3]bool) {
	x, y := p.Origin(w, h)
	frame := p.frame
	if waiting {
		frame = p.waitFrame
	}
	s.Draw(frame, float32(x), float32(y), 337, 512, 0, 0, 337.0/512, 1, white)
	p.DrawNavigation(s, x, y)

	if waiting {
		for i, rect := range [][2]int{{69, 33}, {101, 138}, {239, 73}} {
			t := p.waitHeaders[i]
			if t != nil {
				s.Draw(t, float32(x+rect[0]), float32(y+70), float32([]int{64, 256, 128}[i]), 32, 0, 0, 1, .25, white)
			}
		}
	} else {
		for i := range p.headerButtons {
			b := &p.headerButtons[i]
			b.OriginX, b.OriginY = x, y
			b.Draw(s)
		}
	}
	f.Draw(s, name, x+69, y+52, [4]float32{.6, .827, .106, 1}, 160)
	classes := [6]string{"Fighter", "Defender", "Ranger", "Archer", "Mage", "Priest"}
	for i, name := range p.ClassNames {
		if name != "" {
			classes[i] = name
		}
	}
	for i := 0; i < GuildRows && offset+i < len(members); i++ {
		m := members[offset+i]
		top := y + 112 + i*23
		tint := [4]float32{130.0 / 255, 128.0 / 255, 120.0 / 255, 1}
		if i == p.hoverRow {
			tint = [4]float32{.667, .667, .667, 1}
		}
		if m.ID == selected {
			tint = white
		}
		class := "?"
		if int(m.Class) < len(classes) {
			class = classes[m.Class]
		}
		if waiting {
			f.Draw(s, fmt.Sprint(offset+i+1), x+42, top, tint, 25)
			f.Draw(s, fmt.Sprint(m.Level), x+79, top, tint, 28)
			f.Draw(s, m.Name, x+112, top, tint, 125)
			f.Draw(s, class, x+247, top, tint, 65)
			continue
		}
		f.Draw(s, fmt.Sprint(m.Level), x+40, top, tint, 24)
		f.Draw(s, m.Name, x+67, top, tint, 114)
		f.Draw(s, class, x+189, top, tint, 56)
		f.Draw(s, fmt.Sprint(m.Rank), x+259, top, tint, 16)
		status, color := "Off", [4]float32{1, 0, 0, 1}
		if m.Online {
			status, color = "On", [4]float32{1, 1, 0, 1}
		}
		f.Draw(s, status, x+284, top, color, 28)
	}
	p.scroll.Draw(s, x, y, len(members), GuildRows, offset)
	p.close.OriginX, p.close.OriginY = x, y
	p.close.Draw(s)
	buttonSet := p.actions[:]
	toggleButton := &p.toggle
	if waiting {
		buttonSet, toggleButton = p.waitActions[:], &p.back
	}
	toggleButton.OriginX, toggleButton.OriginY, toggleButton.Disabled = x, y, !canToggle
	toggleButton.Draw(s)
	for i := range buttonSet {
		b := &buttonSet[i]
		b.OriginX, b.OriginY, b.Disabled = x, y, !allowed[i]
		b.Draw(s)
	}
}

func (p *GuildPanel) UpdateNavigation(in *platform.Input, x, y int) bool {
	hit := -1
	for i, top := range []int{61, 124, 186, 248} {
		if in.MouseX >= x && in.MouseX < x+30 && in.MouseY >= y+top && in.MouseY < y+top+55 {
			hit = i
		}
	}
	if in.Pressed {
		p.navHeld = hit
	}
	if in.Clicked {
		held := p.navHeld
		p.navHeld = -1
		if hit >= 0 && held == hit {
			p.Page = hit
			return true
		}
	}
	return false
}
func (p *GuildPanel) DrawNavigation(s *render.Sprites, x, y int) {
	if p.nav == nil {
		return
	}
	for i, top := range []int{61, 124, 186, 248} {
		left, v := x, float32(0)
		if i == p.Page {
			left += 2
			v = .5
		}
		s.Draw(p.nav, float32(left), float32(y+top), 32, 128, float32(i)/4, v, float32(i+1)/4, v+.25, white)
	}
}
