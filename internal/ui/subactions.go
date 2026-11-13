package ui

import (
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

const SubActionWidth, SubActionHeight = 275, 466
const SubActionClose = -2

// SubActions uses the original Sub.png frame and the 4x4 icon_sub atlas.
// Coordinates follow 0x507960/0x507C30. Click activation is a temporary
// alternative to the original drag-to-action-bar flow.
type SubActions struct {
	Position           WindowPosition
	frame, icons, tabs *render.Texture
	close              Button
	held, hover        int
}

func NewSubActions(t Textures) (*SubActions, error) {
	l := &loader{t: t}
	p := &SubActions{frame: l.get("Sub.png"), icons: l.get("icon/icon_sub.png"), tabs: l.get("sub_select_button.png"), held: -1, hover: -1}
	p.close = nativeClose(l.get("talk1_close_button.png"), 250, 1)
	return p, l.err
}
func SubActionsOrigin(w, _ int) (int, int) { return max(0, w-297), 0 }
func (p *SubActions) Origin(w, h int) (int, int) {
	x, y := SubActionsOrigin(w, h)
	return p.Position.Origin(w, h, SubActionWidth, SubActionHeight, x, y)
}
func SubActionCell(index int) (int, int) { return 46 + (index%2)*113, 58 + (index/2)*44 }
func subActionEnabled(i int) bool        { return i >= 0 && i <= 14 }
func (p *SubActions) CellAt(mx, my, w, h int) int {
	x, y := p.Origin(w, h)
	for i := 0; i < 15; i++ {
		cx, cy := SubActionCell(i)
		if mx >= x+cx && mx < x+cx+32 && my >= y+cy && my < y+cy+32 {
			return i
		}
	}
	return -1
}
func (p *SubActions) CancelHeld() { p.held = -1 }
func (p *SubActions) Update(in *platform.Input, w, h int) (int, bool) {
	x, y := p.Origin(w, h)
	p.close.OriginX, p.close.OriginY = x, y
	close := p.close.Update(in)
	p.hover = -1
	for i := 0; i < 16; i++ {
		cx, cy := SubActionCell(i)
		if in.MouseX >= x+cx && in.MouseX < x+cx+32 && in.MouseY >= y+cy && in.MouseY < y+cy+32 {
			p.hover = i
			break
		}
	}
	if in.Pressed {
		p.held = p.hover
	}
	selected := -1
	if in.Clicked {
		if p.hover >= 0 && p.hover == p.held && subActionEnabled(p.hover) {
			selected = p.hover
		}
		p.held = -1
	}
	if close {
		selected = SubActionClose
	}
	captured := in.MouseX >= x && in.MouseX < x+SubActionWidth && in.MouseY >= y && in.MouseY < y+SubActionHeight && (in.Pressed || in.Clicked || in.MouseDown || in.RightPressed || in.RightDown || in.Wheel != 0)
	return selected, captured
}
func (p *SubActions) Draw(s *render.Sprites, w, h int, walking, autoRun bool) {
	x, y := p.Origin(w, h)
	s.Draw(p.frame, float32(x), float32(y), SubActionWidth, SubActionHeight, 0, 0, 275.0/512, 466.0/512, white)
	s.Draw(p.tabs, float32(x), float32(y+60), 32, 128, .5, 0, 1, .25, [4]float32{.45, .45, .45, .65})
	s.Draw(p.tabs, float32(x), float32(y+150), 32, 128, 0, 0, .5, .25, white)
	p.close.OriginX, p.close.OriginY = x, y
	p.close.Draw(s)
	for i := 0; i < 16; i++ {
		cx, cy := SubActionCell(i)
		col := white
		if !subActionEnabled(i) {
			col = [4]float32{.45, .45, .45, .65}
		} else if i == 1 && walking || i == 2 && autoRun {
			col = [4]float32{1, 1, .55, 1}
		}
		u, v := float32(i%4)/4, float32(i/4)/4
		s.Draw(p.icons, float32(x+cx), float32(y+cy), 32, 32, u, v, u+.25, v+.25, col)
	}
}
