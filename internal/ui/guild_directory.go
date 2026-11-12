package ui

import (
	"fmt"

	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

const GuildDirectoryWidth, GuildDirectoryHeight, GuildDirectoryRows = 586, 489, 19

type GuildDirectory struct {
	Position                            WindowPosition
	frame                               *render.Texture
	close, join, create, leave, disband Button
	held                                int
}

func NewGuildDirectory(t Textures) (*GuildDirectory, error) {
	ld := &loader{t: t}
	p := &GuildDirectory{frame: ld.get("guild_manager_list.tga"), held: -1}
	p.close = nativeClose(ld.get("talk1_close_button.tga"), 557, 7)
	p.join = Button{Tex: ld.get("guild_master_button.tga"), OffX: 128, OffY: 285, DrawW: 128, DrawH: 32, HitW: 90, HitH: 25}
	for state, col := range []int{0, 1, 2, 2, 3} {
		p.join.UV[state] = UV{float32(col) / 4, .25, float32(col+1) / 4, .5}
	}
	p.create = Button{Tex: p.join.Tex, OffX: 20, OffY: 285, DrawW: 128, DrawH: 32, HitW: 90, HitH: 25}
	p.leave = Button{Tex: p.join.Tex, OffX: 20, OffY: 320, DrawW: 128, DrawH: 32, HitW: 90, HitH: 25}
	p.disband = Button{Tex: p.join.Tex, OffX: 128, OffY: 320, DrawW: 128, DrawH: 32, HitW: 90, HitH: 25}
	for state, col := range []int{0, 1, 2, 2, 3} {
		p.create.UV[state] = UV{float32(col) / 4, 0, float32(col+1) / 4, .25}
		p.leave.UV[state] = UV{float32(col) / 4, .5, float32(col+1) / 4, .75}
		p.disband.UV[state] = UV{float32(col) / 4, .75, float32(col+1) / 4, 1}
	}
	return p, ld.err
}

func (p *GuildDirectory) Origin(w, h int) (int, int) {
	return p.Position.Origin(w, h, GuildDirectoryWidth, GuildDirectoryHeight, max(0, (w-GuildDirectoryWidth)/2), max(0, (h-GuildDirectoryHeight)/2))
}

func (p *GuildDirectory) Update(in *platform.Input, w, h, count, offset int, canJoin, canCreate, canLeave, canDisband bool) (row, scroll int, join, create, leave, disband, close bool) {
	x, y := p.Origin(w, h)
	p.close.OriginX, p.close.OriginY = x, y
	p.join.OriginX, p.join.OriginY, p.join.Disabled = x, y, !canJoin
	p.create.OriginX, p.create.OriginY, p.create.Disabled = x, y, !canCreate
	p.leave.OriginX, p.leave.OriginY, p.leave.Disabled = x, y, !canLeave
	p.disband.OriginX, p.disband.OriginY, p.disband.Disabled = x, y, !canDisband
	close, join, create, leave, disband = p.close.Update(in), p.join.Update(in), p.create.Update(in), p.leave.Update(in), p.disband.Update(in)
	row, scroll = -1, max(0, min(offset, max(0, count-GuildDirectoryRows)))
	hit := -1
	if in.MouseX >= x+270 && in.MouseX < x+575 && in.MouseY >= y+91 && in.MouseY < y+91+GuildDirectoryRows*18 {
		if in.Wheel != 0 {
			delta := 1
			if in.Wheel < 0 {
				delta = -1
			}
			scroll = max(0, min(max(0, count-GuildDirectoryRows), scroll-delta))
		}
		hit = (in.MouseY - y - 91) / 18
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

func (p *GuildDirectory) Draw(s *render.Sprites, f *Font, blank *render.Texture, w, h int, entries []proto.GuildInfo, offset int, selected uint32, canJoin, canCreate, canLeave, canDisband bool) {
	x, y := p.Origin(w, h)
	s.Draw(p.frame, float32(x), float32(y), GuildDirectoryWidth, GuildDirectoryHeight, 0, 0, 586.0/1024, 489.0/512, white)
	for i := 0; i < GuildDirectoryRows && offset+i < len(entries); i++ {
		g := entries[offset+i]
		rowY := y + 93 + i*18
		if g.ID == selected {
			s.Draw(blank, float32(x+270), float32(rowY-1), 300, 17, 0, 0, 1, 1, [4]float32{.28, .23, .15, .55})
		}
		f.Draw(s, g.Name, x+275, rowY, white, 125)
		f.Draw(s, g.Master, x+405, rowY, white, 104)
		f.Draw(s, fmt.Sprint(g.Points), x+512, rowY, white, 55)
	}
	for _, g := range entries {
		if g.ID != selected {
			continue
		}
		f.Draw(s, g.Name, x+20, y+92, white, 210)
		f.Draw(s, "Master: "+g.Master, x+20, y+114, white, 210)
		f.Draw(s, fmt.Sprintf("Rank: %d  Points: %d", g.Rank, g.Points), x+20, y+128, white, 210)
		for i, line := range wrapLines(f, g.Remark, 205) {
			if i >= 6 {
				break
			}
			f.Draw(s, line, x+20, y+145+i*14, white, 205)
		}
		break
	}
	p.close.OriginX, p.close.OriginY = x, y
	p.join.OriginX, p.join.OriginY, p.join.Disabled = x, y, !canJoin
	p.create.OriginX, p.create.OriginY, p.create.Disabled = x, y, !canCreate
	p.leave.OriginX, p.leave.OriginY, p.leave.Disabled = x, y, !canLeave
	p.disband.OriginX, p.disband.OriginY, p.disband.Disabled = x, y, !canDisband
	p.close.Draw(s)
	p.join.Draw(s)
	p.create.Draw(s)
	p.leave.Draw(s)
	p.disband.Draw(s)
}
