package ui

import (
	"fmt"
	"math"

	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

const SkillBookWidth, SkillBookHeight, SkillBookRows = 333, 489, 18

// Delivery anchor. Frame/row geometry comes from Game.exe 0x501000/0x5020e0.
func SkillBookOrigin(w, h int) (int, int) {
	return max(0, w-SkillBookWidth-16), min(82, max(0, h-SkillBookHeight))
}
func SkillBookRow(row int) (x, y int, ok bool) {
	if row < 0 || row >= SkillBookRows {
		return
	}
	return 45 + 135*(row%2), 79 + 43*(row/2), true
}
func SkillBookHit(x, y int) (int, bool) {
	for row := 0; row < SkillBookRows; row++ {
		cx, cy, _ := SkillBookRow(row)
		if x >= cx && x < cx+123 && y >= cy && y < cy+32 {
			return row, true
		}
	}
	return 0, false
}

type SkillBookAction struct {
	Category, Offset int
	Learn, Select    int
	Close            bool
}
type SkillBook struct {
	Position          WindowPosition
	frame, scroll     *render.Texture
	tabs              [2]*render.Texture
	learn             [SkillBookRows]Button
	arrows            [2]Button
	close             Button
	tabHeld, tabHover int
	rowHeld           int
	scrolling         bool
}

func NewSkillBook(t Textures) (*SkillBook, error) {
	ld := &loader{t: t}
	b := &SkillBook{frame: ld.get("skill.tga"), tabHeld: -1, tabHover: -1, rowHeld: -1}
	b.tabs = [2]*render.Texture{ld.get("skill_select_button1.tga"), ld.get("skill_select_button2.tga")}
	tex := ld.get("skill_button.tga")
	for i := range b.learn {
		button := &b.learn[i]
		*button = Button{Tex: tex, OffX: 61 + 135*(i%2), OffY: 95 + 43*(i/2), DrawW: 16, DrawH: 16, HitW: 16, HitH: 16}
		for state, col := range []int{0, 1, 2, 2, 3} {
			button.UV[state] = UV{float32(col) / 4, 0, float32(col+1) / 4, 1}
		}
	}
	b.scroll = ld.get("scrollbar.tga")
	for i := range b.arrows {
		button := &b.arrows[i]
		*button = Button{Tex: b.scroll, OffX: 302, OffY: 68 + i*379, DrawW: 16, DrawH: 32, HitW: 13, HitH: 19}
		for state, col := range []int{0, 1, 1, 1, 2} {
			button.UV[state] = UV{float32(col) / 8, float32(i) / 2, float32(col+1) / 8, float32(i+1) / 2}
		}
	}
	b.close = nativeClose(ld.get("talk1_close_button.tga"), 303, 8)
	return b, ld.err
}
func (b *SkillBook) Update(in *platform.Input, w, h, category, offset, count int, enabled [SkillBookRows]bool) SkillBookAction {
	x, y := b.Origin(w, h)
	last := max(0, ((count+1)/2-9)*2)
	a := SkillBookAction{Category: category, Offset: max(0, min(offset, last)), Learn: -1, Select: -1}
	b.close.OriginX, b.close.OriginY = x, y
	a.Close = b.close.Update(in)
	b.tabHover = -1
	if in.MouseX >= x && in.MouseX < x+26 && in.MouseY >= y+70 && in.MouseY < y+318 {
		b.tabHover = (in.MouseY - y - 70) / 62
	}
	if in.Pressed {
		b.tabHeld = b.tabHover
		b.rowHeld = -1
		if row, ok := SkillBookHit(in.MouseX-x, in.MouseY-y); ok {
			b.rowHeld = row
		}
	}
	if in.Clicked {
		if b.tabHover >= 0 && b.tabHover == b.tabHeld {
			a.Category = b.tabHover
			a.Offset = 0
		}
		b.tabHeld = -1
		if row, ok := SkillBookHit(in.MouseX-x, in.MouseY-y); ok && row == b.rowHeld {
			a.Select = row
		}
		b.rowHeld = -1
	}
	for i := range b.learn {
		button := &b.learn[i]
		button.OriginX, button.OriginY = x, y
		button.Disabled = !enabled[i]
		if button.Update(in) {
			a.Learn = i
		}
	}
	for i := range b.arrows {
		button := &b.arrows[i]
		button.OriginX, button.OriginY = x, y
		button.Disabled = (i == 0 && a.Offset == 0) || (i == 1 && a.Offset == last)
		if button.Update(in) {
			a.Offset += i*2 - 1
		}
	}
	if in.Pressed && in.MouseX >= x+302 && in.MouseX < x+315 && in.MouseY >= y+87 && in.MouseY < y+447 {
		b.scrolling = true
	}
	if b.scrolling && in.MouseDown {
		a.Offset = int(math.Round(float64(in.MouseY-y-91) * float64(last) / 352))
	}
	if !in.MouseDown {
		b.scrolling = false
	}
	if in.MouseX >= x && in.MouseX < x+SkillBookWidth && in.MouseY >= y && in.MouseY < y+SkillBookHeight {
		if in.Wheel < 0 {
			a.Offset += 2
		}
		if in.Wheel > 0 {
			a.Offset -= 2
		}
	}
	a.Offset = max(0, min(a.Offset, last))
	return a
}
func (b *SkillBook) Draw(s *render.Sprites, f *Font, w, h, category, offset, count int, points uint16, enabled [SkillBookRows]bool) {
	x, y := b.Origin(w, h)
	s.Draw(b.frame, float32(x), float32(y), 333, 489, 0, 0, 333.0/512, 489.0/512, white)
	// Reuse the authored empty-slot region for Phoenix's second column.
	for row := 0; row < 9; row++ {
		s.Draw(b.frame, float32(x+173), float32(y+74+43*row), 41, 40, 38.0/512, 74.0/512, 79.0/512, 114.0/512, white)
	}
	for i := 0; i < 4; i++ {
		sheet, dx, v, height := 0, 0, float32(0), float32(256)
		if i == category {
			sheet, dx, height = 1, 2, 128
		} else if i == b.tabHover {
			v = .5
		}
		s.Draw(b.tabs[sheet], float32(x+dx), float32(y+70+i*62), 32, 64, float32(i)/4, v, float32(i+1)/4, v+64/height, white)
	}
	b.close.OriginX, b.close.OriginY = x, y
	b.close.Draw(s)
	f.Draw(s, fmt.Sprint(points), x+245, y+49, white, 58)
	for i := range b.learn {
		b.learn[i].OriginX, b.learn[i].OriginY = x, y
		b.learn[i].Disabled = !enabled[i]
		if enabled[i] {
			b.learn[i].Draw(s)
		}
	}
	for i := range b.arrows {
		b.arrows[i].OriginX, b.arrows[i].OriginY = x, y
		b.arrows[i].Disabled = (i == 0 && offset == 0) || (i == 1 && offset >= max(0, ((count+1)/2-9)*2))
		b.arrows[i].Draw(s)
	}
	thumb := 87
	if count > SkillBookRows {
		thumb += int(float64(offset) * 352 / float64(count-SkillBookRows))
	}
	s.Draw(b.scroll, float32(x+302), float32(y+thumb), 13, 8, .5, 0, .5+13.0/128, 8.0/64, white)
}

func (b *SkillBook) Origin(w, h int) (int, int) {
	x, y := SkillBookOrigin(w, h)
	if b == nil {
		return x, y
	}
	return b.Position.Origin(w, h, SkillBookWidth, SkillBookHeight, x, y)
}

func (b *SkillBook) CancelHeld() { b.rowHeld = -1 }
