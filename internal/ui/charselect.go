package ui

import (
	"fmt"
	"strconv"

	"github.com/Spelunkern/Shaiya-GX/internal/data/text"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

// CharSelect is the character window (constructor 0x46ce30, draw
// 0x46a960 / 0x469870 / 0x4697d0). Layout floats at 0x708f80..0x70901c.
type CharSelect struct {
	panel, chain                 *render.Texture
	Slots                        [5]*Button
	Create, Delete, Option, Exit *Button
	Start                        *Button
	Navigation                   PreviewNavigation
	Chars                        [5]*proto.CharacterSlot
	Selected                     int
	Light                        bool // DAT_021fd8d6: picks the class name set
	msg                          text.SysMsg
	Busy                         bool
	lastClick                    float64
	clock                        float64
}

// Column UVs of character_select_button.tga (512x256, 4 rows of 64).
func columnUV(c int) [5]UV {
	u0, u1 := float32(c)*.25, float32(c+1)*.25
	return [5]UV{{u0, 0, u1, .25}, {u0, .25, u1, .5}, {u0, .5, u1, .75}, {u0, .5, u1, .75}, {u0, .75, u1, 1}}
}

func NewCharSelect(t Textures, msg text.SysMsg) (*CharSelect, error) {
	c := &CharSelect{msg: msg, Selected: -1}
	ld := &loader{t: t}
	load := ld.get
	c.panel = load("character_select.tga")
	c.Navigation = newPreviewNavigation(ld)
	c.chain = load("character_chain.tga")
	{
		slot := load("character_select_button2.tga")
		btn := load("character_select_button.tga")
		start := load("character_select_start.tga")
		// Slots: (43, 244 + 36*i), drawn 256x32, clickable 162x25.
		slotUV := [5]UV{{0, 0, 1, .25}, {0, .25, 1, .5}, {0, .5, 1, .75}, {0, .25, 1, .5}, {0, .75, 1, 1}}
		for i := range c.Slots {
			c.Slots[i] = &Button{Tex: slot, OffX: 43, OffY: 244 + 36*i, DrawW: 256, DrawH: 32, HitW: 162, HitH: 25, UV: slotUV, Toggle: true}
		}
		mk := func(x, y, col int) *Button {
			return &Button{Tex: btn, OffX: x, OffY: y, DrawW: 0x80, DrawH: 0x40, HitW: 0x58, HitH: 0x20, UV: columnUV(col)}
		}
		c.Create, c.Delete, c.Option, c.Exit = mk(29, 440, 0), mk(29, 478, 2), mk(136, 440, 1), mk(136, 478, 3)
		c.Start = &Button{Tex: start, OffX: 62, OffY: 527, DrawW: 0x100, DrawH: 0x40, HitW: 131, HitH: 43, UV: FourRowUV}
	}
	return c, ld.err
}

// loader keeps the first texture error so constructors read linearly.
type loader struct {
	t   Textures
	err error
}

func (l *loader) get(name string) *render.Texture {
	if l.err != nil {
		return nil
	}
	tex, err := l.t.Interface(name)
	if err != nil {
		l.err = err
	}
	return tex
}

// Origins from 0x46a960.
func charPanelOrigin(w, h int) (int, int) { return w / 16, h/2 - 0x127 }
func charNavOrigin(w, h int) (int, int)   { return w - w/12 - 0x82, h - h/6 - 100 }

type CharAction int

const (
	CharNone CharAction = iota
	CharStart
	CharCreate
	CharDelete
	CharExit
)

// SetChars fills the slots from the 0x0101 list.
func (c *CharSelect) SetChars(slots map[uint8]proto.CharacterSlot) {
	var previousID uint32
	if c.Selected >= 0 && c.Chars[c.Selected] != nil {
		previousID = c.Chars[c.Selected].ID
	}
	for i := range c.Chars {
		if s, ok := slots[uint8(i)]; ok {
			s := s
			c.Chars[i] = &s
		} else {
			c.Chars[i] = nil
		}
	}
	if c.Selected < 0 || c.Chars[c.Selected] == nil {
		c.Selected = -1
		for i, ch := range c.Chars {
			if ch != nil {
				c.Selected = i
				break
			}
		}
	}
	if c.Selected < 0 || c.Chars[c.Selected].ID != previousID {
		c.Navigation.Reset()
	}
}

func (c *CharSelect) controls(w, h int) {
	px, py := charPanelOrigin(w, h)
	for _, b := range []*Button{c.Create, c.Delete, c.Option, c.Exit, c.Start} {
		b.OriginX, b.OriginY, b.Disabled = px, py, c.Busy
	}
	c.Start.Disabled = c.Busy || c.Selected < 0 || c.Chars[c.Selected].PendingDelete
	c.Delete.Disabled = c.Busy || c.Selected < 0
	c.Option.Disabled = true // NO VERIFICADO: options window not ported
	for i, b := range c.Slots {
		b.OriginX, b.OriginY, b.Disabled = px, py, c.Busy || c.Chars[i] == nil
		b.Checked = i == c.Selected
	}
}

func (c *CharSelect) Update(in *platform.Input, w, h int, dt float64) CharAction {
	c.clock += dt
	c.controls(w, h)
	nx, ny := charNavOrigin(w, h)
	c.Navigation.Update(in, nx, ny, !c.Busy && c.Selected >= 0, false)
	act := CharNone
	for i, b := range c.Slots {
		if b.Update(in) {
			if i == c.Selected && c.clock-c.lastClick < 0.4 && !c.Chars[i].PendingDelete {
				act = CharStart // NO VERIFICADO: double click enters
			}
			if c.Selected != i {
				c.Navigation.Reset()
			}
			c.Selected, c.lastClick = i, c.clock
		}
		b.Checked = i == c.Selected
	}
	if c.Start.Update(in) {
		act = CharStart
	}
	if c.Create.Update(in) {
		act = CharCreate
	}
	if c.Delete.Update(in) {
		act = CharDelete
	}
	c.Option.Update(in)
	if c.Exit.Update(in) {
		act = CharExit
	}
	if !c.Start.Disabled && in.HasKey(platform.KeyEnter) {
		act = CharStart
	}
	return act
}

func (c *CharSelect) className(class uint8) string {
	return ClassName(c.msg, c.Light, class)
}

func ClassName(msg text.SysMsg, lightSide bool, class uint8) string {
	light := []int{0x13ba, 0x13bb, 0x13c0, 0x13c1, 0x13c2, 0x13bc}
	fury := []int{0x13b7, 0x13b8, 0x13bd, 0x13b9, 0x13be, 0x13bf}
	set := fury
	if lightSide {
		set = light
	}
	if int(class) < len(set) {
		return msg.Get(set[class])
	}
	return ""
}

func (c *CharSelect) mapName(id uint16) string {
	if m, ok := mapNameMsg[id]; ok {
		return c.msg.Get(m)
	}
	return strconv.Itoa(int(id))
}

func (c *CharSelect) Draw(s *render.Sprites, f *Font, w, h int) {
	c.controls(w, h)
	px, py := charPanelOrigin(w, h)
	// Chains hanging the panel (0x46aacf..0x46ab0b).
	for _, p := range [][2]int{{0x20, -0x136}, {0xd0, -0x136}, {0x20, -0x9b}, {0xd0, -0x9b}} {
		s.DrawImage(c.chain, float32(px+p[0]), float32(py+p[1]))
	}
	s.DrawImage(c.panel, float32(px), float32(py))
	if c.Selected >= 0 && c.Chars[c.Selected] != nil {
		ch := c.Chars[c.Selected]
		// Name centred around x+0x7e at 3 px per character (0x469870).
		f.Draw(s, ch.Name, px+0x7e-len(ch.Name)*3, py+0x28, white, 0)
		x := px + 0x1f
		f.Draw(s, fmt.Sprintf("%s : %d", c.msg.Get(0x13c9), ch.Level), x, py+0x49, white, 0)
		f.Draw(s, fmt.Sprintf("%s : %s", c.msg.Get(0x13ca), c.className(ch.Class)), x, py+0x5b, white, 0)
		modes := []int{0x1d6, 0x1d7, 0x1d8, 0x1d9}
		mode := ""
		if int(ch.Mode) < len(modes) {
			mode = c.msg.Get(modes[ch.Mode])
		}
		f.Draw(s, fmt.Sprintf("%s : %s", c.msg.Get(0x13cb), mode), x, py+0x6d, white, 0)
		f.Draw(s, fmt.Sprintf("%s : %s", c.msg.Get(0x13d3), c.mapName(ch.MapID)), x, py+0x7f, white, 0)
	}
	for i, b := range c.Slots {
		b.Draw(s)
		if ch := c.Chars[i]; ch != nil {
			// Caption centred in the clickable rect (0x543eb0).
			tw := f.Width(ch.Name)
			f.Draw(s, ch.Name, b.X()+(b.HitW-tw)/2, b.Y()+(b.HitH-f.Height)/2, white, 0)
		}
	}
	for _, b := range []*Button{c.Create, c.Delete, c.Option, c.Exit, c.Start} {
		b.Draw(s)
	}
	nx, ny := charNavOrigin(w, h)
	c.Navigation.Draw(s, nx, ny)
}
