package ui

import (
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

// Antialias levels shown in the slot that used to read "Color Bit".
const (
	AntialiasOff = iota
	AntialiasFXAA
	AntialiasMSAA2
	AntialiasMSAA4
	AntialiasMSAA8
	AntialiasMSAA16
	antialiasLevels
)

var antialiasLabels = [antialiasLevels]string{"Off", "FXAA", "MSAA 2x", "MSAA 4x", "MSAA 8x", "MSAA 16x"}

// AntialiasSamples is the MSAA sample count for a level (0 for Off and FXAA).
func AntialiasSamples(level int) int {
	switch level {
	case AntialiasMSAA2:
		return 2
	case AntialiasMSAA4:
		return 4
	case AntialiasMSAA8:
		return 8
	case AntialiasMSAA16:
		return 16
	}
	return 0
}

// Value box and popup placement, relative to the graphics page origin.
const (
	antialiasBoxX, antialiasBoxY = 207, 82
	antialiasPopupX              = 205
	antialiasPopupY              = 103
	antialiasPopupW              = 86
)

// SetAntialiasMaxSamples offers only the MSAA levels the GPU supports.
func (m *GameMenu) SetAntialiasMaxSamples(maxSamples int) {
	m.antialiasChoices = m.antialiasChoices[:0]
	for level := range antialiasLevels {
		if s := AntialiasSamples(level); s == 0 || s <= maxSamples {
			m.antialiasChoices = append(m.antialiasChoices, level)
		}
	}
	m.antialiasOpen, m.antialiasHeld, m.antialiasHover = false, -1, -1
}

func (m *GameMenu) antialiasRows() []int {
	if len(m.antialiasChoices) == 0 {
		return []int{AntialiasOff, AntialiasFXAA, AntialiasMSAA2, AntialiasMSAA4, AntialiasMSAA8}
	}
	return m.antialiasChoices
}

func (m *GameMenu) antialiasSelected() int {
	for i, level := range m.antialiasRows() {
		if level == m.Settings.Antialias {
			return i
		}
	}
	return 0
}

// updateAntialias mirrors the resolution dropdown: the arrow toggles a list,
// a click picks a level, and the open list owns pointer and keyboard input.
func (m *GameMenu) updateAntialias(in *platform.Input, x, y int) (bool, int) {
	rows := m.antialiasRows()
	if m.antialiasButton.Update(in) {
		m.antialiasOpen = !m.antialiasOpen
		m.displayOpen = false
		m.antialiasHeld = -1
		return true, MenuNone
	}
	if !m.antialiasOpen {
		return false, MenuNone
	}
	if in.HasKey(platform.KeyEscape) || in.HasKey(platform.KeyEnter) {
		m.antialiasOpen = false
		return true, MenuNone
	}
	if in.HasKey(platform.KeyUp) || in.HasKey(platform.KeyDown) {
		n := m.antialiasSelected()
		if in.HasKey(platform.KeyUp) {
			n--
		} else {
			n++
		}
		m.Settings.Antialias = rows[max(0, min(len(rows)-1, n))]
		return true, MenuChanged
	}
	px, py := x+antialiasPopupX, y+antialiasPopupY
	row := PlayerMenuRowAt(in.MouseX, in.MouseY, px, py, antialiasPopupW, len(rows))
	m.antialiasHover = row
	if in.Pressed {
		m.antialiasHeld = row
		if row < 0 {
			m.antialiasOpen = false
		}
	}
	if in.Clicked {
		if row >= 0 && m.antialiasHeld == row {
			m.Settings.Antialias = rows[row]
			m.antialiasOpen, m.antialiasHeld = false, -1
			return true, MenuChanged
		}
		m.antialiasHeld = -1
	}
	return true, MenuNone
}

func (m *GameMenu) drawAntialias(s *render.Sprites, f *Font, solid *render.Texture, x, y int) {
	// The frame art bakes "Color Bit" in; paint it over with the panel's
	// background and label the slot "Antialias" instead.
	if solid != nil {
		s.Draw(solid, float32(x+156), float32(y+86), 49, 14, 0, 0, 1, 1, [4]float32{11.0 / 255, 10.0 / 255, 9.0 / 255, 1})
	}
	label := "Antialias"
	f.Draw(s, label, x+202-f.Width(label), y+87, white, 60)
	f.Draw(s, antialiasLabels[min(max(m.Settings.Antialias, 0), antialiasLevels-1)], x+211, y+87, white, 56)
	m.antialiasButton.Draw(s)
}

func (m *GameMenu) drawAntialiasPopup(s *render.Sprites, f *Font, solid *render.Texture, x, y int) {
	if m.Page != MenuGraphics || !m.antialiasOpen || solid == nil {
		return
	}
	rows := m.antialiasRows()
	labels := make([]string, len(rows))
	for i, level := range rows {
		labels[i] = antialiasLabels[level]
	}
	hover := m.antialiasHover
	if hover < 0 {
		hover = m.antialiasSelected()
	}
	m.displayPopup.Draw(s, f, solid, x+antialiasPopupX, y+antialiasPopupY, antialiasPopupW, labels, hover)
}
