package ui

import (
	"fmt"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

// EP5 social sheets share the guild window origin and native 337x512 frame.
type SocialPanel struct {
	SortColumn                                   int
	Descending                                   bool
	sortHeld                                     int
	frames                                       [3]*render.Texture
	headers                                      [3][5]*render.Texture
	add, whisper, remove, block, unblock, search Button
	invite                                       Button
	close                                        Button
	scroll                                       panelScroll
	Name                                         TextField
	Focus                                        bool
	EditingMemo                                  uint32
	held                                         int
}

func NewSocialPanel(t Textures) (*SocialPanel, error) {
	ld := &loader{t: t}
	p := &SocialPanel{SortColumn: 1, sortHeld: -1, held: -1, Name: TextField{Max: 20}}
	for i, n := range []string{"friend.png", "friend_cut.png", "party_find.png"} {
		p.frames[i] = ld.get(n)
	}
	for page, names := range [][]string{{"guild_sort_no.png", "guild_sort_name.png", "guild_sort_job.png", "friend_sort_memo.png", "guild_sort_on.png"}, {"guild_sort_no.png", "guild_sort_name.png", "", "friend_sort_memo.png", ""}, {"guild_sort_no.png", "guild_sort_name.png", "guild_sort_job.png", "party_find_sort_level.png", ""}} {
		for i, n := range names {
			if n != "" {
				p.headers[page][i] = ld.get(n)
			}
		}
	}
	makeButton := func(file string, x, y, hw, hh, row int) Button {
		tex := ld.get(file)
		b := Button{Tex: tex, OffX: x, OffY: y, DrawW: 128, DrawH: 32, HitW: hw, HitH: hh}
		width := 128
		height := 128
		if tex != nil {
			height = tex.H
			width = tex.W
		}
		for state, line := range []int{0, 32, 64, 64, 96} {
			b.UV[state] = UV{0, float32(row+line) / float32(height), 128.0 / float32(width), float32(row+line+32) / float32(height)}
		}
		return b
	}
	p.add = makeButton("friend_find_button.png", 237, 46, 74, 20, 0)
	p.whisper = makeButton("friend_button.png", 48, 466, 83, 27, 0)
	p.remove = makeButton("friend_button.png", 234, 466, 70, 27, 0)
	for i := range p.remove.UV {
		p.remove.UV[i].U0 += .5
		p.remove.UV[i].U1 += .5
	}
	p.block = makeButton("friend_cut_button.png", 237, 46, 74, 20, 0)
	p.unblock = makeButton("friend_cut_button2.png", 48, 466, 66, 27, 0)
	p.search = makeButton("party_find_button1.png", 184, 46, 122, 22, 0)
	p.invite = makeButton("party_find_button4.png", 145, 466, 66, 27, 0)
	p.close = nativeClose(ld.get("talk1_close_button.png"), 307, 8)
	p.scroll = newPanelScroll(ld.get("scrollbar.png"), 303, 99, 356)
	return p, ld.err
}
func (p *SocialPanel) buttons(page int) []*Button {
	if page == 1 {
		return []*Button{&p.add, &p.whisper, &p.remove}
	}
	if page == 2 {
		return []*Button{&p.block, &p.unblock}
	}
	return []*Button{&p.search, &p.whisper, &p.invite}
}

// action: 0 add/register, 1 whisper/unblock, 2 delete, 3 edit memo.
func (p *SocialPanel) Update(in *platform.Input, x, y, page, count, offset int, selected bool) (row, scroll, action int, closed bool) {
	if in.Pressed {
		p.sortHeld = -1
	}
	header := -1
	for i, left := range []int{36, 62, 184, 247, 276} {
		width := []int{25, 121, 61, 27, 37}[i]
		if p.headers[page-1][i] != nil && in.MouseX >= x+left && in.MouseX < x+left+width && in.MouseY >= y+70 && in.MouseY < y+99 {
			header = i
		}
	}
	if in.Pressed {
		p.sortHeld = header
	}
	if in.Clicked && header >= 0 && header == p.sortHeld {
		p.Descending = p.SortColumn == header && !p.Descending
		p.SortColumn = header
		p.sortHeld = -1
	}
	row, action = -1, -1
	scroll = offset
	p.close.OriginX, p.close.OriginY = x, y
	closed = p.close.Update(in) || in.HasKey(platform.KeyEscape)
	if in.Pressed {
		p.Focus = page != 3 && in.MouseX >= x+60 && in.MouseX < x+230 && in.MouseY >= y+43 && in.MouseY < y+69
	}
	if p.Focus {
		p.Name.Edit(in)
		if in.HasKey(platform.KeyEnter) {
			action = 0
			if p.EditingMemo != 0 {
				action = 3
			}
		}
	}
	for i, b := range p.buttons(page) {
		b.OriginX, b.OriginY = x, y
		b.Disabled = i > 0 && !selected
		if b.Update(in) {
			action = i
		}
	}
	if in.Pressed {
		p.held = -1
		if in.MouseX >= x+36 && in.MouseX < x+303 && in.MouseY >= y+99 && in.MouseY < y+453 {
			p.held = (in.MouseY - y - 99) / 23
		}
	}
	if in.Clicked && p.held >= 0 {
		hit := (in.MouseY - y - 99) / 23
		if in.MouseX >= x+36 && in.MouseX < x+303 && hit == p.held && offset+hit < count {
			row = offset + hit
		}
		p.held = -1
	}
	scroll = p.scroll.Update(in, x, y, count, 15, offset)
	if in.Wheel != 0 && in.MouseX >= x+36 && in.MouseX < x+316 && in.MouseY >= y+99 && in.MouseY < y+455 {
		scroll = max(0, min(max(0, count-15), scroll-int(in.Wheel)))
	}
	return
}
func (p *SocialPanel) Draw(s *render.Sprites, f *Font, blank *render.Texture, x, y, page int, rows []proto.SocialContact, search []proto.PartySearchEntry, offset int, selected uint32, classes [6]string) {
	tex := p.frames[page-1]
	s.Draw(tex, float32(x), float32(y), 337, 512, 0, 0, 337.0/512, 1, white)
	for i, t := range p.headers[page-1] {
		if t != nil {
			s.Draw(t, float32(x+[]int{36, 62, 184, 247, 276}[i]), float32(y+70), float32(t.W), 32, 0, 0, 1, 32.0/float32(t.H), white)
		}
	}
	if page != 3 {
		shown := p.Name.Text
		if p.Focus {
			shown += "|"
		}
		f.Draw(s, shown, x+69, y+50, white, 160)
	}
	count := len(rows)
	if page == 3 {
		count = len(search)
	}
	for i := 0; i < 15 && offset+i < count; i++ {
		cy := y + 104 + i*23
		tint := [4]float32{.55, .55, .55, 1}
		if page == 3 {
			r := search[offset+i]
			f.Draw(s, fmt.Sprint(offset+i+1), x+40, cy, white, 24)
			f.Draw(s, fmt.Sprint(r.Level), x+250, cy, white, 50)
			f.Draw(s, r.Name, x+67, cy, white, 114)
			if r.Class < 6 {
				f.Draw(s, classes[r.Class], x+189, cy, white, 110)
			}
			continue
		}
		r := rows[offset+i]
		if r.ID == selected {
			tint = white
		}
		f.Draw(s, fmt.Sprint(offset+i+1), x+40, cy, tint, 24)
		f.Draw(s, r.Name, x+67, cy, tint, 114)
		if page == 1 && r.Class < 6 {
			f.Draw(s, classes[r.Class], x+189, cy, tint, 56)
		}
		f.Draw(s, r.Memo, x+250, cy, tint, 25)
		if page == 1 {
			state := "Off"
			color := [4]float32{1, 0, 0, 1}
			if r.Online {
				state = "On"
				color = [4]float32{1, 1, 0, 1}
			}
			f.Draw(s, state, x+284, cy, color, 28)
		}
	}
	for _, b := range p.buttons(page) {
		b.OriginX, b.OriginY = x, y
		b.Draw(s)
	}
	p.close.OriginX, p.close.OriginY = x, y
	p.close.Draw(s)
	p.scroll.Draw(s, x, y, count, 15, offset)
}
