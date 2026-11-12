package ui

import (
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

const RaidCompactWidth, RaidCompactHeight = 160, 336

func (p *PartyPanel) RaidCompactVisible() bool { return p.raidCompact }
func (p *PartyPanel) RaidCompactOrigin(w, h int) (int, int) {
	height := 6 * PartyRowHeight
	if p.raidGroups {
		height += 24
	}
	return p.RaidCompactPosition.Origin(w, h, RaidCompactWidth, height, 25, 150)
}

func (p *PartyPanel) syncRaidGroup(self uint32, slots map[uint16]uint32) {
	if p.raidGroupInitialized && p.raidGroups {
		return
	}
	for slot, id := range slots {
		if id == self && slot < 30 {
			p.raidGroup, p.raidGroupInitialized = int(slot)/6, true
			return
		}
	}
}

func (p *PartyPanel) RaidCompactMemberAt(w, h, mouseX, mouseY int, self uint32, slots map[uint16]uint32) uint32 {
	if !p.raidCompact {
		return 0
	}
	p.syncRaidGroup(self, slots)
	x, y := p.RaidCompactOrigin(w, h)
	if p.raidGroups {
		y += 24
	}
	if mouseX < x || mouseX >= x+RaidCompactWidth || mouseY < y || mouseY >= y+6*PartyRowHeight {
		return 0
	}
	return slots[uint16(p.raidGroup*6+(mouseY-y)/PartyRowHeight)]
}

// Native compact raid view: 0x514a30, 0x514f70 and 0x5152c0.
func (p *PartyPanel) UpdateRaidCompact(in *platform.Input, w, h int, self uint32, slots map[uint16]uint32) (uint32, bool) {
	if !p.raidCompact {
		return 0, false
	}
	p.syncRaidGroup(self, slots)
	x, y := p.RaidCompactOrigin(w, h)
	height := 6 * PartyRowHeight
	tab := -1
	if p.raidGroups {
		height += 24
		if in.MouseX >= x+20 && in.MouseX < x+140 && in.MouseY >= y+7 && in.MouseY < y+30 {
			tab = (in.MouseX - x - 20) / 24
		}
	}
	id := p.RaidCompactMemberAt(w, h, in.MouseX, in.MouseY, self, slots)
	if in.Pressed {
		p.raidCompactHeld, p.raidTabHeld = id, tab+1
	}
	selected := uint32(0)
	if in.Clicked {
		if p.raidTabHeld == tab+1 && tab >= 0 {
			p.raidGroup, p.raidGroupInitialized = tab, true
		} else if p.raidCompactHeld == id {
			selected = id
		}
		p.raidCompactHeld, p.raidTabHeld = 0, 0
	}
	inside := in.MouseX >= x && in.MouseX < x+RaidCompactWidth && in.MouseY >= y && in.MouseY < y+height
	return selected, inside && (in.Pressed || in.Clicked || in.MouseDown || in.RightPressed || in.RightDown || in.Wheel != 0)
}

func (p *PartyPanel) DrawRaidCompact(s *render.Sprites, f *Font, w, h int, self uint32, members []proto.PartyMember, slots map[uint16]uint32, leader, target uint32) {
	if !p.raidCompact {
		return
	}
	p.syncRaidGroup(self, slots)
	x, y := p.RaidCompactOrigin(w, h)
	if p.raidGroups {
		tint := [4]float32{1, 1, 1, p.raidOpacity}
		for group := 0; group < 5; group++ {
			v := float32(0)
			if group == p.raidGroup {
				v = .25
			}
			s.Draw(p.raidSelector, float32(x+20+group*24), float32(y+7), 32, 32, float32(group)/8, v, float32(group+1)/8, v+.25, tint)
		}
		y += 24
	}
	for row := 0; row < 6; row++ {
		id := slots[uint16(p.raidGroup*6+row)]
		for _, member := range members {
			if member.ID == id && id != 0 {
				p.drawMemberOpacity(s, f, x, y+row*PartyRowHeight, member, leader, target, p.raidOpacity)
				break
			}
		}
	}
}
