package ui

import (
	"fmt"
	"math"

	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

const ActionSlots = 10
const ActionBars = 2
const TotalActionSlots = ActionSlots * ActionBars

type ActionSlot struct {
	Icon         *render.Texture
	UV           UV
	Label        string
	Level, Count uint8
	Cooldown     float64
}

// Phoenix ep4_action_bars.gd and main_slot_hud.gd: two independently
// movable bars, background drag handles and centre-preserving orientation.
type ActionBar struct {
	frames                    [2]*render.Texture
	change                    *render.Texture
	Position                  [ActionBars]WindowPosition
	Hover, held               int
	dragging, orientationHeld int // one-based; zero means idle
	offsetX, offsetY          int
}

func NewActionBar(t Textures) (*ActionBar, error) {
	ld := &loader{t: t}
	return &ActionBar{frames: [2]*render.Texture{ld.get("main_slot_1.tga"), ld.get("main_slot_2.tga")}, change: ld.get("main_slot_change.tga"), Hover: -1, held: -1}, ld.err
}
func (b *ActionBar) Size(bar int) (int, int) {
	if b.Position[bar].Vertical {
		return 56, 449
	}
	return 448, 54
}
func ActionBarOrigin(w, h int) (int, int) { return max(0, (w-448)/2), max(0, h-64-8-54*2-6) }
func (b *ActionBar) Origin(bar, w, h int) (int, int) {
	x, y := ActionBarOrigin(w, h)
	y += bar * 60
	width, height := b.Size(bar)
	return b.Position[bar].Origin(w, h, width, height, x, y)
}
func (b *ActionBar) SlotOrigin(index, w, h int) (int, int) {
	bar, cell := index/ActionSlots, index%ActionSlots
	x, y := b.Origin(bar, w, h)
	if b.Position[bar].Vertical {
		return x + 12, y + 31 + 40*cell
	}
	return x + 30 + 40*cell, y + 10
}
func (b *ActionBar) SlotAt(mx, my, w, h int) int {
	for index := TotalActionSlots - 1; index >= 0; index-- {
		x, y := b.SlotOrigin(index, w, h)
		if mx >= x && mx < x+32 && my >= y && my < y+32 {
			return index
		}
	}
	return -1
}

// ActionSlotAt retains the default-layout helper for synthetic fixtures.
func ActionSlotAt(mx, my, w, h int) int { return (&ActionBar{}).SlotAt(mx, my, w, h) }
func (b *ActionBar) changeAt(bar, mx, my, w, h int) bool {
	x, y := b.Origin(bar, w, h)
	if b.Position[bar].Vertical {
		x += 4
		y += 425
	} else {
		x += 424
		y += 30
	}
	return mx >= x && mx < x+22 && my >= y && my < y+24
}
func (b *ActionBar) Contains(mx, my, w, h int) bool {
	for bar := 0; bar < ActionBars; bar++ {
		x, y := b.Origin(bar, w, h)
		bw, bh := b.Size(bar)
		if mx >= x && mx < x+bw && my >= y && my < y+bh {
			return true
		}
	}
	return false
}
func (b *ActionBar) LayoutActive() bool { return b.dragging > 0 || b.orientationHeld > 0 }
func (b *ActionBar) CancelHeld()        { b.held = -1 }

// Layout captures the complete drag/release; changed requests persistence.
func (b *ActionBar) Layout(in *platform.Input, w, h int) (captured, changed bool) {
	if b.dragging > 0 {
		bar := b.dragging - 1
		bw, bh := b.Size(bar)
		b.Position[bar].Move(in.MouseX-b.offsetX, in.MouseY-b.offsetY, w, h, bw, bh)
		if in.Clicked || !in.MouseDown {
			b.dragging = 0
			changed = true
		}
		return true, changed
	}
	if b.orientationHeld > 0 {
		bar := b.orientationHeld - 1
		if in.Clicked {
			if b.changeAt(bar, in.MouseX, in.MouseY, w, h) {
				x, y := b.Origin(bar, w, h)
				bw, bh := b.Size(bar)
				cx, cy := x+bw/2, y+bh/2
				b.Position[bar].Vertical = !b.Position[bar].Vertical
				bw, bh = b.Size(bar)
				b.Position[bar].Move(cx-bw/2, cy-bh/2, w, h, bw, bh)
				changed = true
			}
			b.orientationHeld = 0
		} else if !in.MouseDown {
			b.orientationHeld = 0
		}
		return true, changed
	}
	if !in.Pressed || b.SlotAt(in.MouseX, in.MouseY, w, h) >= 0 {
		return false, false
	}
	for bar := ActionBars - 1; bar >= 0; bar-- {
		x, y := b.Origin(bar, w, h)
		bw, bh := b.Size(bar)
		if in.MouseX < x || in.MouseX >= x+bw || in.MouseY < y || in.MouseY >= y+bh {
			continue
		}
		b.CancelHeld()
		if b.changeAt(bar, in.MouseX, in.MouseY, w, h) {
			b.orientationHeld = bar + 1
		} else {
			b.dragging = bar + 1
			b.offsetX, b.offsetY = in.MouseX-x, in.MouseY-y
		}
		return true, false
	}
	return false, false
}
func (b *ActionBar) Update(in *platform.Input, w, h int) (int, bool) {
	b.Hover = b.SlotAt(in.MouseX, in.MouseY, w, h)
	if in.Pressed {
		b.held = b.Hover
	}
	click := -1
	wasHeld := b.held >= 0
	if in.Clicked {
		if b.Hover >= 0 && b.held == b.Hover {
			click = b.Hover
		}
		b.held = -1
	}
	return click, b.Contains(in.MouseX, in.MouseY, w, h) && (in.Pressed || in.Clicked || in.MouseDown || in.RightPressed || in.RightDown || in.Wheel != 0) || wasHeld && in.Clicked
}
func (b *ActionBar) Draw(s *render.Sprites, f *Font, blank *render.Texture, w, h int, slots [TotalActionSlots]ActionSlot, _ bool) {
	for bar := 0; bar < ActionBars; bar++ {
		x, y := b.Origin(bar, w, h)
		bw, bh := b.Size(bar)
		orientation := 0
		if b.Position[bar].Vertical {
			orientation = 1
		}
		tex := b.frames[orientation]
		if tex != nil {
			s.Draw(tex, float32(x), float32(y), float32(bw), float32(bh), 0, 0, float32(bw)/float32(tex.W), float32(bh)/float32(tex.H), white)
		}
		nx, ny := x+3, y+19
		cx, cy := x+424, y+30
		if orientation == 1 {
			nx, ny = x+19, y+2
			cx, cy = x+4, y+425
		}
		f.Draw(s, fmt.Sprint(bar+1), nx, ny, white, 18)
		if b.change != nil {
			s.Draw(b.change, float32(cx), float32(cy), 22, 24, 0, float32(orientation)*.5, 22.0/float32(b.change.W), float32(orientation)*.5+24.0/float32(b.change.H), white)
		}
	}
	for i, slot := range slots {
		cx, cy := b.SlotOrigin(i, w, h)
		if slot.Icon != nil {
			u := slot.UV
			s.Draw(slot.Icon, float32(cx), float32(cy), 32, 32, u.U0, u.V0, u.U1, u.V1, white)
			f.Draw(s, fmt.Sprint((i+1)%10), cx+23, cy, white, 12)
		}
		label := ""
		if slot.Count > 1 {
			label = fmt.Sprint(slot.Count)
		} else if slot.Level > 0 {
			label = fmt.Sprint(slot.Level)
		}
		f.Draw(s, label, cx+32-f.Width(label), cy+18, [4]float32{1, 1, .3, 1}, 32)
		if slot.Cooldown > 0 {
			s.Draw(blank, float32(cx), float32(cy), 32, 32, 0, 0, 1, 1, [4]float32{0, 0, 0, .65})
			text := fmt.Sprintf("%.0f", math.Ceil(slot.Cooldown))
			f.Draw(s, text, cx+16-f.Width(text)/2, cy+9, white, 32)
		}
	}
}

// SkillIcon matches 0x4a1810: thousands select the sheet, remainder is 1-based.
func SkillIcon(icon uint16) (string, UV, bool) {
	index := int(icon%1000) - 1
	if index < 0 || index >= 256 {
		return "", UV{}, false
	}
	name := "icon/icon_skill.tga"
	if icon >= 1000 {
		name = fmt.Sprintf("icon/icon_skill%d.dds", int(icon)/1000+1)
	}
	u, v := float32(index%16)/16, float32(index/16)/16
	return name, UV{u, v, u + 1.0/16, v + 1.0/16}, true
}
