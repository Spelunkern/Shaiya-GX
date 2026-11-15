package ui

import (
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

// GraphicsOption is one row of the Advanced graphics window: a label and a
// list of values stored in ClientSettings. To add an option, append an entry
// to AdvancedGraphicsOptions and give the setting a field (with a valid range
// in ClientSettings.Valid and a default in DefaultClientSettings); the window
// lays out, edits and resets it without further UI code.
type GraphicsOption struct {
	Label  string
	Values func(m *GameMenu) []string
	Get    func(m *GameMenu) int // index into Values
	Set    func(m *GameMenu, index int)
}

func onOff() []string { return []string{"Off", "On"} }

func boolIndex(v bool) int {
	if v {
		return 1
	}
	return 0
}

// AdvancedGraphicsOptions lists the rows of the Advanced graphics window.
var AdvancedGraphicsOptions = []GraphicsOption{
	{
		Label: "Antialias",
		Values: func(m *GameMenu) []string {
			rows := m.antialiasRows()
			labels := make([]string, len(rows))
			for i, level := range rows {
				labels[i] = antialiasLabels[level]
			}
			return labels
		},
		Get: func(m *GameMenu) int { return m.antialiasSelected() },
		Set: func(m *GameMenu, i int) { m.Settings.Antialias = m.antialiasRows()[i] },
	},
	{
		Label:  "VSync",
		Values: func(*GameMenu) []string { return onOff() },
		Get:    func(m *GameMenu) int { return boolIndex(!m.Settings.NoVSync) },
		Set:    func(m *GameMenu, i int) { m.Settings.NoVSync = i == 0 },
	},
}

// The window reuses option_main's frame (256x470), its title band covered
// and relabelled, beside the graphics page.
const (
	advancedW, advancedH             = 256, 470
	advancedGap                      = 4
	advancedRowY, advancedRowStep    = 56, 30
	advancedLabelX                   = 22
	advancedBoxX, advancedBoxW       = 128, 82
	advancedBoxH                     = 18
	advancedArrowX                   = advancedBoxX + advancedBoxW - 2
	advancedPopupX, advancedPopupW   = advancedBoxX - 4, advancedBoxW + 16
	advancedButtonX, advancedButtonY = 199, 80
	advancedTitle                    = "Advanced Graphics"
	advancedFooterY                  = 360
)

// advancedButtonFace is the visible height of talk_blank_button's 32px
// cell; the rest is its shadow. At y 80 its face centres on the
// Resolution box (y 85-99), and x 199 centres it in the free right half.
const advancedButtonFace = 25

// advancedRows fit between the title and the frame's lower separator.
func advancedRows() int {
	return min(len(AdvancedGraphicsOptions), (advancedFooterY-advancedRowY)/advancedRowStep)
}

type advancedGraphics struct {
	frame         *render.Texture
	open          bool
	button, close Button
	arrows        []Button
	list          int // row whose value list is open, -1 none
	hover, held   int
}

func (m *GameMenu) initAdvanced(ld *loader) {
	a := &m.advanced
	a.frame = ld.get("option_main.png")
	a.button = Button{Tex: ld.get("talk_blank_button.png"), OffX: advancedButtonX, OffY: advancedButtonY, DrawW: 71, DrawH: 32, HitW: 71, HitH: 24}
	for i := range a.button.UV {
		col := []int{0, 1, 2, 2, 3}[i]
		a.button.UV[i] = UV{float32(col*128) / 512, 0, float32(col*128+71) / 512, .5}
	}
	a.close = nativeClose(ld.get("talk1_close_button.png"), 228, 5)
	a.arrows = make([]Button, len(AdvancedGraphicsOptions))
	for i := range a.arrows {
		a.arrows[i] = m.displayButton
		a.arrows[i].OffX, a.arrows[i].OffY = advancedArrowX, advancedRowY+i*advancedRowStep-3
	}
	a.list, a.hover, a.held = -1, -1, -1
}

func (m *GameMenu) resetAdvanced() {
	a := &m.advanced
	a.open, a.list, a.hover, a.held = false, -1, -1, -1
	a.button.held, a.close.held = false, false
	for i := range a.arrows {
		a.arrows[i].held = false
	}
}

// AdvancedRect is the Advanced window's screen rectangle while it is open:
// right of the graphics page, or left of it when the screen is too narrow.
func (m *GameMenu) AdvancedRect(w, h int) (x, y, width, height int, ok bool) {
	if m == nil || m.Page != MenuGraphics || !m.advanced.open {
		return 0, 0, 0, 0, false
	}
	gx, gy := m.Origin(w, h)
	gw, _ := m.Size()
	x = gx + gw + advancedGap
	if x+advancedW > w {
		x = max(0, gx-advancedW-advancedGap)
	}
	y = min(max(gy, 0), max(0, h-advancedH))
	return x, y, advancedW, advancedH, true
}

func (m *GameMenu) advancedListOpen() bool { return m.advanced.open && m.advanced.list >= 0 }

func (m *GameMenu) advancedPopupY(y, row int) int {
	return y + advancedRowY + row*advancedRowStep + advancedBoxH + 2
}

// updateAdvanced handles the Advanced button and window; handled reports
// that the input belonged to them.
func (m *GameMenu) updateAdvanced(in *platform.Input, w, h, gx, gy int) (bool, int) {
	a := &m.advanced
	a.button.OriginX, a.button.OriginY = gx, gy
	if !a.open {
		if a.button.Update(in) {
			a.open, a.list = true, -1
			return true, MenuNone
		}
		return false, MenuNone
	}
	x, y, width, height, _ := m.AdvancedRect(w, h)
	a.close.OriginX, a.close.OriginY = x, y
	for i := range a.arrows {
		a.arrows[i].OriginX, a.arrows[i].OriginY = x, y
	}
	if a.list >= 0 {
		opt := AdvancedGraphicsOptions[a.list]
		values := opt.Values(m)
		if in.HasKey(platform.KeyEscape) || in.HasKey(platform.KeyEnter) {
			a.list = -1
			return true, MenuNone
		}
		if in.HasKey(platform.KeyUp) || in.HasKey(platform.KeyDown) {
			n := opt.Get(m) - 1
			if in.HasKey(platform.KeyDown) {
				n = opt.Get(m) + 1
			}
			opt.Set(m, max(0, min(len(values)-1, n)))
			return true, MenuChanged
		}
		row := PlayerMenuRowAt(in.MouseX, in.MouseY, x+advancedPopupX, m.advancedPopupY(y, a.list), advancedPopupW, len(values))
		a.hover = row
		if in.Pressed {
			a.held = row
			if row < 0 {
				a.list = -1
			}
		}
		if in.Clicked {
			if row >= 0 && a.held == row {
				opt.Set(m, row)
				a.list, a.held = -1, -1
				return true, MenuChanged
			}
			a.held = -1
		}
		return true, MenuNone
	}
	if a.close.Update(in) || in.HasKey(platform.KeyEscape) || a.button.Update(in) {
		a.open = false
		return true, MenuNone
	}
	for i := 0; i < advancedRows(); i++ {
		if a.arrows[i].Update(in) {
			a.list, a.hover, a.held = i, -1, -1
			return true, MenuNone
		}
	}
	inside := in.MouseX >= x && in.MouseX < x+width && in.MouseY >= y && in.MouseY < y+height
	if inside && (in.Pressed || in.Clicked || in.MouseDown) {
		return true, MenuNone
	}
	return false, MenuNone
}

func (m *GameMenu) drawAdvancedButton(s *render.Sprites, f *Font, solid *render.Texture, gx, gy int) {
	// Cover the frame's baked "Color Bit" label and box; the button takes the slot.
	if solid != nil {
		s.Draw(solid, float32(gx+152), float32(gy+80), 138, 25, 0, 0, 1, 1, [4]float32{11.0 / 255, 10.0 / 255, 9.0 / 255, 1})
	}
	b := &m.advanced.button
	b.OriginX, b.OriginY = gx, gy
	b.Draw(s)
	label := "Advanced"
	tx := gx + advancedButtonX + (71-f.Width(label))/2
	ty := gy + advancedButtonY + (advancedButtonFace-f.Height)/2
	if b.held {
		tx, ty = tx+1, ty+1
	}
	f.Draw(s, label, tx, ty, white, 69)
}

func (m *GameMenu) drawAdvanced(s *render.Sprites, f *Font, solid *render.Texture, w, h int) {
	x, y, _, _, ok := m.AdvancedRect(w, h)
	a := &m.advanced
	if !ok || a.frame == nil {
		return
	}
	s.Draw(a.frame, float32(x), float32(y), advancedW, advancedH, 0, 0, advancedW/256.0, advancedH/512.0, white)
	// Paint the baked "Option (O)" title over with a clean slice of the same band.
	const bandU0, bandU1, bandV0, bandV1 = 62.0 / 256, 70.0 / 256, 6.0 / 512, 30.0 / 512
	s.Draw(a.frame, float32(x+58), float32(y+6), 140, 24, bandU0, bandV0, bandU1, bandV1, white)
	title := f
	if m.TitleFont != nil {
		title = m.TitleFont
	}
	tx := x + (advancedW-title.Width(advancedTitle))/2
	ty := y + 6 + (24-title.Height)/2
	// Dark shadow under a doubled face, like the baked window captions.
	for _, o := range [][2]int{{-1, 0}, {2, 0}, {0, -1}, {1, -1}, {0, 1}, {1, 1}, {1, 2}, {2, 2}} {
		title.Draw(s, advancedTitle, tx+o[0], ty+o[1], [4]float32{0, 0, 0, .85}, advancedW)
	}
	for dx := 0; dx < 2; dx++ {
		title.Draw(s, advancedTitle, tx+dx, ty, white, advancedW)
	}
	a.close.OriginX, a.close.OriginY = x, y
	a.close.Draw(s)
	for i := 0; i < advancedRows(); i++ {
		opt := AdvancedGraphicsOptions[i]
		ry := y + advancedRowY + i*advancedRowStep
		for dx := 0; dx < 2; dx++ {
			f.Draw(s, opt.Label, x+advancedLabelX+dx, ry+(advancedBoxH-f.Height)/2, white, advancedBoxX-advancedLabelX-6)
		}
		if solid != nil {
			s.Draw(solid, float32(x+advancedBoxX), float32(ry), advancedBoxW, advancedBoxH, 0, 0, 1, 1, [4]float32{46.0 / 255, 46.0 / 255, 45.0 / 255, 1})
		}
		values := opt.Values(m)
		value := min(max(opt.Get(m), 0), len(values)-1)
		f.Draw(s, values[value], x+advancedBoxX+4, ry+(advancedBoxH-f.Height)/2, white, advancedBoxW-6)
		a.arrows[i].OriginX, a.arrows[i].OriginY = x, y
		a.arrows[i].Draw(s)
	}
}

func (m *GameMenu) drawAdvancedPopup(s *render.Sprites, f *Font, solid *render.Texture, w, h int) {
	x, y, _, _, ok := m.AdvancedRect(w, h)
	a := &m.advanced
	if !ok || a.list < 0 || solid == nil {
		return
	}
	opt := AdvancedGraphicsOptions[a.list]
	hover := a.hover
	if hover < 0 {
		hover = opt.Get(m)
	}
	m.displayPopup.Draw(s, f, solid, x+advancedPopupX, m.advancedPopupY(y, a.list), advancedPopupW, opt.Values(m), hover)
}
