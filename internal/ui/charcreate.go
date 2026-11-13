package ui

import (
	"strings"

	"github.com/Spelunkern/Shaiya-GX/internal/data/text"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

// CharCreate uses the original frame (0x468980), anchored as in 0x467170.
// Native atlas controls; packet fields follow 0x5980e0.
type Creation struct {
	Navigation                     PreviewNavigation
	panel                          *render.Texture
	races, sexes                   [2]Button
	jobs                           [3]Button
	arrows                         [6]Button
	modes                          [4]Button
	capabilities                   [6]*render.Texture
	msg                            text.SysMsg
	description, descriptionOffset int
	OK, Cancel                     *Button
	Name                           TextField
	Draft                          proto.NewCharacter
	MaxMode                        uint8
	Busy                           bool
}

func NewCharCreate(t Textures, msg text.SysMsg) (*Creation, error) {
	ld := &loader{t: t}
	c := &Creation{msg: msg, panel: ld.get("create.png"), Name: TextField{Max: 18}}
	c.OK = &Button{Tex: ld.get("ok_button.png"), OffX: 37, OffY: 622, DrawW: 128, DrawH: 32, HitW: 67, HitH: 27, UV: FourRowUV}
	c.Cancel = &Button{Tex: ld.get("cancel_button.png"), OffX: 150, OffY: 622, DrawW: 128, DrawH: 32, HitW: 67, HitH: 27, UV: FourRowUV}
	c.loadControls(ld)
	c.Navigation = newPreviewNavigation(ld)
	return c, ld.err
}

// Family/job mask: CUser::PacketCharacter, Rapture rules.go (0x46bf20).
var familyJobs = [4][]uint8{{0, 1, 5}, {2, 3, 4}, {2, 4, 5}, {0, 1, 3}}

func (c *Creation) Open(slot, faction, maxMode uint8) {
	c.Draft = proto.NewCharacter{Slot: slot, Race: faction * 2, Mode: 1, Height: 2}
	if faction == 1 {
		c.Draft.Race = 3
	}
	c.Draft.Class = familyJobs[c.Draft.Race][0]
	c.MaxMode = min(maxMode, 3)
	c.Draft.Mode = min(c.Draft.Mode, c.MaxMode)
	c.Name.Text, c.Busy = "", false
	c.Navigation.Reset()
	c.description, c.descriptionOffset = 0, 0
	for _, b := range c.buttons() {
		b.held = false
		b.state = StateNormal
	}
}

func (c *Creation) Update(in *platform.Input, w, h int) CharAction {
	c.controls(w, h)
	nx, ny := CreationNavigationOrigin(w, h)
	c.Navigation.Update(in, nx, ny, !c.Busy, true)
	if c.Cancel.Update(in) || (!c.Busy && in.HasKey(platform.KeyEscape)) {
		return CharExit
	}
	if c.Busy {
		return CharNone
	}
	c.Name.Edit(in)
	c.OK.Disabled = len(c.cleanName()) < 2
	d := &c.Draft
	for i := range c.races {
		if c.races[i].Update(in) {
			race := uint8(i)
			if d.Race/2 == 1 {
				race = 3 - uint8(i)
			}
			if d.Race != race {
				d.Race = race
				d.Class = familyJobs[race][0]
				d.Hair, d.Face = 0, 0
			}
			c.description, c.descriptionOffset = 0, 0
		}
	}
	for i := range c.sexes {
		if c.sexes[i].Update(in) {
			d.Gender = uint8(i)
			d.Hair, d.Face = 0, 0
		}
	}
	for i := range c.jobs {
		if c.jobs[i].Update(in) {
			d.Class = familyJobs[d.Race][i]
			c.description, c.descriptionOffset = 1, 0
		}
	}
	for i := range c.arrows {
		if c.arrows[i].Update(in) {
			fields := []*uint8{&d.Hair, &d.Face, &d.Height}
			field := fields[i/2]
			*field = uint8((int(*field) + 4 + (i%2)*2) % 5)
		}
	}
	for i := range c.modes {
		if c.modes[i].Update(in) {
			d.Mode = uint8(i)
			c.description, c.descriptionOffset = 2, 0
		}
	}
	x, y := CreationOrigin(w, h)
	if in.MouseX >= x+24 && in.MouseX < x+242 && in.MouseY >= y+28 && in.MouseY < y+195 {
		if in.Wheel < 0 {
			c.descriptionOffset++
		}
		if in.Wheel > 0 {
			c.descriptionOffset = max(0, c.descriptionOffset-1)
		}
	}

	if c.OK.Update(in) || (!c.OK.Disabled && in.HasKey(platform.KeyEnter)) {
		c.Draft.Name = c.cleanName()
		return CharCreate
	}
	return CharNone
}

func (c *Creation) Draw(s *render.Sprites, f *Font, w, h int) {
	c.controls(w, h)
	x, y := CreationOrigin(w, h)
	s.Draw(c.panel, float32(x), float32(y), 250, 670, 0, 0, 250.0/256, 670.0/1024, white)
	f.Draw(s, c.Name.Text+"_", x+72, y+253, white, 135)
	for _, b := range c.buttons() {
		b.Draw(s)
	}
	id := []int{0x139a, 0x139b, 0x139d, 0x139c}[c.Draft.Race]
	if c.description == 1 {
		id = 0x139e + int(c.Draft.Class)
		if c.Draft.Race >= 2 {
			id += 6
		}
	}
	if c.description == 2 {
		id = 0x13aa + int(c.Draft.Mode)
	}
	lines := wrapLines(f, c.msg.Get(id), 201)
	c.descriptionOffset = max(0, min(c.descriptionOffset, max(0, len(lines)-9)))
	for i, line := range lines[c.descriptionOffset:min(len(lines), c.descriptionOffset+9)] {
		f.Draw(s, line, x+24, y+40+i*16, white, 201)
	}
	nx, ny := CreationNavigationOrigin(w, h)
	c.Navigation.Draw(s, nx, ny)
	s.DrawImage(c.capabilities[c.Draft.Class], float32(w-w/14-256), float32(h-h/8-256))
}

func (c *Creation) cleanName() string { return strings.ReplaceAll(c.Name.Text, " ", "") }
