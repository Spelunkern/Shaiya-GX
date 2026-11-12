package ui

import (
	"fmt"

	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

const PartyWidth, PartyRowHeight = 165, 52

// PartyPanel: original 0x50a6d0 constructor / 0x50abd0 draw / 0x50a930 input.
// Each 256x64 atlas row advances 52px; hit width is the native 165px.
type PartyPanel struct {
	frame, graph, icons               *render.Texture
	raidFrame, raidRows, raidCheck    *render.Texture
	raidSelector, raidSlider          *render.Texture
	raidClose, raidOK, raidCancel     Button
	RaidPosition                      WindowPosition
	RaidCompactPosition               WindowPosition
	raidHeld                          int
	raidCompactHeld                   uint32
	raidTabHeld                       int
	raidSliderHeld                    bool
	raidDraftDivision                 uint32
	raidDraftAuto                     bool
	raidDraftCompact, raidDraftGroups bool
	raidDraftOpacity                  float32
	raidCompact, raidGroups           bool
	raidOpacity                       float32
	raidGroup                         int
	raidGroupInitialized              bool
	raidDraftReady, raidDraftDirty    bool
	held                              uint32
}

func NewPartyPanel(t Textures) (*PartyPanel, error) {
	ld := &loader{t: t}
	p := &PartyPanel{frame: ld.get("party.tga"), graph: ld.get("party_graph.tga"), icons: ld.get("party_icon.tga"), raidFrame: ld.get("union_party.tga"), raidRows: ld.get("party2.tga"), raidCheck: ld.get("chat_check.tga"), raidSelector: ld.get("union_select.tga"), raidSlider: ld.get("option_graphic_slider.tga"), raidCompact: true, raidGroups: true, raidOpacity: 1}
	p.raidClose = nativeClose(ld.get("talk1_close_button.tga"), 549, 4)
	p.raidOK = Button{Tex: ld.get("ok_button.tga"), OffX: 380, OffY: 402, DrawW: 128, DrawH: 32, HitW: 65, HitH: 25, UV: FourRowUV}
	p.raidCancel = Button{Tex: ld.get("cancel_button.tga"), OffX: 467, OffY: 402, DrawW: 128, DrawH: 32, HitW: 65, HitH: 25, UV: FourRowUV}
	return p, ld.err
}
func PartyOrigin(w, h, count int) (int, int) {
	return max(0, min(100, w-PartyWidth)), max(0, min(100, h-count*PartyRowHeight))
}
func (p *PartyPanel) Update(in *platform.Input, w, h int, members []proto.PartyMember) (uint32, bool) {
	x, y := PartyOrigin(w, h, len(members))
	inside := in.MouseX >= x && in.MouseX < x+PartyWidth && in.MouseY >= y && in.MouseY < y+len(members)*PartyRowHeight
	id := uint32(0)
	if inside {
		id = members[(in.MouseY-y)/PartyRowHeight].ID
	}
	if in.Pressed {
		p.held = id
	}
	selected := uint32(0)
	if in.Clicked {
		if p.held == id {
			selected = id
		}
		p.held = 0
	}
	return selected, inside && (in.Pressed || in.Clicked || in.MouseDown || in.RightPressed || in.RightDown || in.Wheel != 0)
}
func (p *PartyPanel) Draw(s *render.Sprites, f *Font, w, h int, members []proto.PartyMember, leader, target uint32) {
	x, y := PartyOrigin(w, h, len(members))
	for i, m := range members {
		p.drawMember(s, f, x, y+i*PartyRowHeight, m, leader, target)
	}
}

// The native 0x5172b0 window uses five columns of six slots.
const RaidWidth, RaidHeight = 575, 440
const RaidSlotWidth, RaidSlotHeight = 111, 37

func (p *PartyPanel) RaidOrigin(w, h int) (int, int) {
	return p.RaidPosition.Origin(w, h, RaidWidth, RaidHeight, 262+(w-1024)/2, 105)
}

func RaidSlotPosition(w, h int, slot uint16) (int, int) {
	return (&PartyPanel{}).RaidSlotPosition(w, h, slot)
}

func (p *PartyPanel) RaidSlotPosition(w, h int, slot uint16) (int, int) {
	x, y := p.RaidOrigin(w, h)
	return x + 20 + int(slot)/6*RaidSlotWidth, y + 68 + int(slot)%6*RaidSlotHeight
}

func (p *PartyPanel) RaidMemberAt(w, h, mouseX, mouseY int, slots map[uint16]uint32) uint32 {
	for slot, id := range slots {
		x, y := p.RaidSlotPosition(w, h, slot)
		if id != 0 && mouseX >= x && mouseX < x+RaidSlotWidth && mouseY >= y && mouseY < y+RaidSlotHeight {
			return id
		}
	}
	return 0
}

type RaidAction uint8

const (
	RaidNone RaidAction = iota
	RaidClose
	RaidApply
	RaidGroupLoot
	RaidRandomLoot
	RaidMasterLoot
	RaidAutoJoin
	RaidCompact
	RaidGroupSelect
)

func (p *PartyPanel) SetRaidSettings(division uint32, auto bool) {
	if !p.raidDraftReady || !p.raidDraftDirty {
		p.raidDraftDivision, p.raidDraftAuto, p.raidDraftReady = division, auto, true
		p.raidDraftCompact, p.raidDraftGroups, p.raidDraftOpacity = p.raidCompact, p.raidGroups, p.raidOpacity
	}
}
func (p *PartyPanel) RaidSettings() (uint32, bool) { return p.raidDraftDivision, p.raidDraftAuto }
func (p *PartyPanel) ResetRaidSettings()           { p.raidDraftReady, p.raidDraftDirty = false, false }
func (p *PartyPanel) ApplyRaidViewSettings() {
	p.raidCompact, p.raidGroups, p.raidOpacity = p.raidDraftCompact, p.raidDraftGroups, p.raidDraftOpacity
}

func (p *PartyPanel) UpdateRaid(in *platform.Input, w, h int, slots map[uint16]uint32, leader bool) (uint32, RaidAction, bool) {
	x, y := p.RaidOrigin(w, h)
	id := p.RaidMemberAt(w, h, in.MouseX, in.MouseY, slots)
	action := RaidNone
	if leader {
		for i, box := range [][4]int{{30, 311, 112, 16}, {142, 311, 130, 16}, {30, 343, 112, 16}, {306, 311, 125, 16}} {
			if in.MouseX >= x+box[0] && in.MouseX < x+box[0]+box[2] && in.MouseY >= y+box[1] && in.MouseY < y+box[1]+box[3] {
				action = RaidAction(int(RaidGroupLoot) + i)
				break
			}
		}
	}
	for i, box := range [][4]int{{439, 311, 132, 16}, {439, 337, 132, 16}} {
		if in.MouseX >= x+box[0] && in.MouseX < x+box[0]+box[2] && in.MouseY >= y+box[1] && in.MouseY < y+box[1]+box[3] {
			action = RaidAction(int(RaidCompact) + i)
			break
		}
	}
	if in.Pressed && in.MouseX >= x+300 && in.MouseX <= x+555 && in.MouseY >= y+353 && in.MouseY <= y+383 {
		p.raidSliderHeld = true
	}
	if p.raidSliderHeld {
		if in.MouseDown || in.Clicked {
			p.raidDraftOpacity = max(0, min(1, float32(in.MouseX-x-308)/242))
			p.raidDraftDirty = true
		}
		if in.Clicked || !in.MouseDown {
			p.raidSliderHeld = false
		}
	}
	if in.Pressed {
		p.held = id
		p.raidHeld = int(action)
	}
	selected := uint32(0)
	if in.Clicked {
		if p.held == id {
			selected = id
		}
		p.held = 0
		if p.raidHeld != int(action) {
			action = RaidNone
		}
		p.raidHeld = 0
	} else {
		action = RaidNone
	}
	if action != RaidNone {
		switch action {
		case RaidGroupLoot:
			p.raidDraftDivision = 1
		case RaidRandomLoot:
			p.raidDraftDivision = 2
		case RaidMasterLoot:
			p.raidDraftDivision = 4
		case RaidAutoJoin:
			p.raidDraftAuto = !p.raidDraftAuto
		case RaidCompact:
			p.raidDraftCompact = !p.raidDraftCompact
		case RaidGroupSelect:
			if p.raidDraftCompact {
				p.raidDraftGroups = !p.raidDraftGroups
			}
		}
		p.raidDraftDirty, action = true, RaidNone
	}
	for i, button := range []*Button{&p.raidClose, &p.raidOK, &p.raidCancel} {
		button.OriginX, button.OriginY = x, y
		if button.Update(in) {
			if i == 1 {
				action = RaidApply
			} else {
				action = RaidClose
			}
		}
	}
	inside := in.MouseX >= x && in.MouseX < x+RaidWidth && in.MouseY >= y && in.MouseY < y+RaidHeight
	return selected, action, inside && (in.Pressed || in.Clicked || in.MouseDown || in.RightPressed || in.RightDown || in.Wheel != 0)
}

func (p *PartyPanel) DrawRaid(s *render.Sprites, f *Font, w, h int, members []proto.PartyMember, slots map[uint16]uint32, leader, target uint32) {
	x, y := p.RaidOrigin(w, h)
	s.Draw(p.raidFrame, float32(x), float32(y), RaidWidth, RaidHeight, 0, 0, float32(RaidWidth)/1024, float32(RaidHeight)/512, white)
	for slot := uint16(0); slot < 30; slot++ {
		id := slots[slot]
		sx, sy := p.RaidSlotPosition(w, h, slot)
		v := float32(0)
		if id != 0 && id == target {
			v = .5
		}
		s.Draw(p.raidRows, float32(sx), float32(sy), 128, 64, 0, v, 1, v+.25, white)
		for _, m := range members {
			if m.ID == id {
				color := [4]float32{.65, .62, .62, 1}
				if id == leader {
					color = [4]float32{.93, .93, .89, 1}
				}
				f.Draw(s, fmt.Sprint(m.Level), sx+4, sy+8, color, 24)
				f.Draw(s, m.Name, sx+25, sy+8, color, 72)
				if m.MaxHP > 0 {
					width := 88 * float32(min(m.HP, m.MaxHP)) / float32(m.MaxHP)
					s.Draw(p.graph, float32(sx+4), float32(sy+24), width, 8, 0, 0, 0, .25, white)
				}
				break
			}
		}
	}
	for i, active := range []bool{p.raidDraftDivision == 1, p.raidDraftDivision == 2, p.raidDraftDivision == 4, p.raidDraftAuto, p.raidDraftCompact, p.raidDraftCompact && p.raidDraftGroups} {
		if active {
			pos := [][2]int{{30, 311}, {142, 311}, {30, 343}, {306, 311}, {439, 311}, {439, 337}}[i]
			s.DrawImage(p.raidCheck, float32(x+pos[0]), float32(y+pos[1]))
		}
	}
	s.Draw(p.raidSlider, float32(x+308)+p.raidDraftOpacity*242-8, float32(y+354), 16, 28, 0, 0, .5, .875, white)
	for _, button := range []*Button{&p.raidClose, &p.raidOK, &p.raidCancel} {
		button.OriginX, button.OriginY = x, y
		button.Draw(s)
	}
}

func (p *PartyPanel) drawMember(s *render.Sprites, f *Font, x, rowY int, m proto.PartyMember, leader, target uint32) {
	p.drawMemberOpacity(s, f, x, rowY, m, leader, target, 1)
}
func (p *PartyPanel) drawMemberOpacity(s *render.Sprites, f *Font, x, rowY int, m proto.PartyMember, leader, target uint32, opacity float32) {
	tint := [4]float32{1, 1, 1, opacity}
	v := float32(0)
	if m.ID == target {
		v = .5
	}
	s.Draw(p.frame, float32(x), float32(rowY), 256, 64, 0, v, 1, v+.5, tint)
	col := 0
	if m.Class < 6 {
		col = [...]int{2, 5, 1, 0, 3, 4}[m.Class]
	}
	s.Draw(p.icons, float32(x+2), float32(rowY+2), 32, 32, float32(col)/8, 0, float32(col+1)/8, 1, tint)
	color := [4]float32{.93, .93, .89, opacity}
	if m.ID == leader {
		color = [4]float32{.96, .84, .74, opacity}
	}
	f.Draw(s, fmt.Sprint(m.Level), x+28, rowY+10, color, 24)
	f.Draw(s, m.Name, x+50, rowY+10, color, 91)
	for j, pool := range [][2]uint32{{m.HP, m.MaxHP}, {m.MP, m.MaxMP}, {m.SP, m.MaxSP}} {
		if pool[1] == 0 {
			continue
		}
		width := float32(120) * float32(min(pool[0], pool[1])) / float32(pool[1])
		dy := [...]int{27, 36, 43}[j]
		s.Draw(p.graph, float32(x+21), float32(rowY+dy), width, 8, 0, float32(j)/4, 0, float32(j+1)/4, tint)
	}
}
