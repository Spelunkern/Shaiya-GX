package ui

import (
	"fmt"

	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

const GuildWidth, GuildHeight, GuildRows = 337, 512, 19

// GuildPanel uses the roster half of the original 1024x512 guild sheet.
type GuildPanel struct {
	Position     WindowPosition
	frame        *render.Texture
	waitFrame    *render.Texture
	headers      [5]*render.Texture
	waitHeaders  [3]*render.Texture
	close        Button
	actions      [3]Button
	waitActions  [2]Button
	toggle, back Button
	notice       Button
	held         int
}

func NewGuildPanel(t Textures) (*GuildPanel, error) {
	ld := &loader{t: t}
	p := &GuildPanel{frame: ld.get("guild_master.tga"), held: -1}
	p.waitFrame = ld.get("guild_wait.tga")
	for i, name := range []string{"guild_sort_lv.tga", "guild_sort_name.tga", "guild_sort_job.tga", "guild_sort_grade.tga", "guild_sort_on.tga"} {
		p.headers[i] = ld.get(name)
	}
	for i, name := range []string{"guild_wait_lv.tga", "guild_wait_name.tga", "guild_wait_job.tga"} {
		p.waitHeaders[i] = ld.get(name)
	}
	p.close = nativeClose(ld.get("talk1_close_button.tga"), 307, 8)
	buttons := ld.get("guild_button1.tga")
	for i := range p.actions {
		b := &p.actions[i]
		*b = Button{Tex: buttons, OffX: 37 + 69*i, OffY: 453, DrawW: 128, DrawH: 32, HitW: 65, HitH: 25}
		for state, col := range []int{0, 1, 2, 2, 3} {
			b.UV[state] = UV{float32(col) / 4, float32(i) / 4, float32(col+1) / 4, float32(i+1) / 4}
		}
	}
	waitButtons := ld.get("guild_wait_button.tga")
	for i := range p.waitActions {
		b := &p.waitActions[i]
		*b = Button{Tex: waitButtons, OffX: 41 + 72*i, OffY: 453, DrawW: 128, DrawH: 32, HitW: 65, HitH: 25}
		for state, col := range []int{0, 1, 2, 2, 3} {
			b.UV[state] = UV{float32(col) / 4, float32(i) / 2, float32(col+1) / 4, float32(i+1) / 2}
		}
	}
	p.toggle = Button{Tex: ld.get("guild_button2.tga"), OffX: 237, OffY: 47, DrawW: 128, DrawH: 32, HitW: 75, HitH: 21, UV: FourRowUV}
	p.back = Button{Tex: ld.get("guild_wait_button2.tga"), OffX: 237, OffY: 47, DrawW: 128, DrawH: 32, HitW: 75, HitH: 21, UV: FourRowUV}
	p.notice = Button{OffX: 220, OffY: 481, DrawW: 90, DrawH: 25, HitW: 90, HitH: 25}
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
	if etinKnown {
		f.Draw(s, fmt.Sprintf("Etin: %d", etin), x+39, y+485, white, 170)
	}
	color := [4]float32{.18, .16, .12, .8}
	if !enabled {
		color = [4]float32{.09, .09, .09, .5}
	}
	s.Draw(blank, float32(x+220), float32(y+481), 90, 25, 0, 0, 1, 1, color)
	f.Draw(s, "Edit notice", x+230, y+486, white, 80)
}

func (p *GuildPanel) Origin(w, h int) (int, int) {
	return p.Position.Origin(w, h, GuildWidth, GuildHeight, 32, max(0, (h-GuildHeight)/2))
}

func (p *GuildPanel) Update(in *platform.Input, w, h, count, offset int, waiting, canToggle bool, allowed [3]bool) (row, scroll, action int, toggle, close bool) {
	x, y := p.Origin(w, h)
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
	if in.MouseX >= x+37 && in.MouseX < x+306 && in.MouseY >= y+99 && in.MouseY < y+99+GuildRows*18 {
		if in.Wheel != 0 {
			delta := 1
			if in.Wheel < 0 {
				delta = -1
			}
			scroll = max(0, min(max(0, count-GuildRows), scroll-delta))
		}
		hit = (in.MouseY - y - 99) / 18
	}
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
	// The left 256px of guild_master is a separate guild-directory panel.
	if waiting {
		s.Draw(p.waitFrame, float32(x), float32(y), 337, 512, 0, 0, 337.0/512, 1, white)
	} else {
		s.Draw(p.frame, float32(x+26), float32(y), 311, 489, 256.0/1024, 0, 586.0/1024, 489.0/512, white)
	}
	if !waiting {
		for i, rect := range [][2]int{{36, 25}, {62, 105}, {168, 32}, {201, 68}, {270, 43}} {
			if p.headers[i] != nil {
				t := p.headers[i]
				s.Draw(t, float32(x+rect[0]), float32(y+70), float32(rect[1]), 29, 0, 0, float32(rect[1])/float32(t.W), 29/float32(t.H), white)
			}
		}
	} else {
		for i, rect := range [][2]int{{69, 32}, {101, 138}, {239, 73}} {
			if t := p.waitHeaders[i]; t != nil {
				s.Draw(t, float32(x+rect[0]), float32(y+70), float32(rect[1]), 29, 0, 0, float32(rect[1])/float32(t.W), 29/float32(t.H), white)
			}
		}
	}
	nameX := 38
	if waiting {
		nameX = 65
	}
	f.Draw(s, name, x+nameX, y+43, white, 160)
	classes := [...]string{"Ftr", "Def", "Rng", "Arc", "Mag", "Pri"}
	for i := 0; i < GuildRows && offset+i < len(members); i++ {
		m := members[offset+i]
		rowY := y + 101 + i*18
		if m.ID == selected {
			s.Draw(blank, float32(x+36), float32(rowY-1), 276, 17, 0, 0, 1, 1, [4]float32{.28, .23, .15, .55})
		}
		color := [4]float32{.85, .81, .72, 1}
		if !m.Online {
			color = [4]float32{.53, .53, .53, 1}
		}
		class := "?"
		if int(m.Class) < len(classes) {
			class = classes[m.Class]
		}
		if waiting {
			f.Draw(s, fmt.Sprint(m.Level), x+72, rowY, white, 27)
			f.Draw(s, m.Name, x+104, rowY, white, 130)
			f.Draw(s, class, x+242, rowY, white, 68)
			continue
		}
		f.Draw(s, fmt.Sprint(m.Level), x+38, rowY, color, 24)
		f.Draw(s, m.Name, x+63, rowY, color, 102)
		f.Draw(s, class, x+169, rowY, color, 31)
		f.Draw(s, fmt.Sprint(m.Rank), x+205, rowY, color, 62)
		status := "Off"
		if m.Online {
			status = "On"
		}
		f.Draw(s, status, x+272, rowY, color, 40)
	}
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
