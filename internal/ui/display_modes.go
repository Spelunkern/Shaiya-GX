package ui

import (
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

const displayModeRows = 8

func (m *GameMenu) SetDisplayModes(modes []platform.DisplayMode) {
	m.DisplayModes = platform.NormalizeDisplayModes(modes)
	m.displayOpen = false
	m.displayHeld = -1
}
func (m *GameMenu) DisplayPopupOpen() bool {
	return m != nil && m.Page == MenuGraphics && (m.displayOpen || m.antialiasOpen)
}
func (m *GameMenu) displaySelected() int {
	for i, mode := range m.DisplayModes {
		if mode == m.Settings.DisplayMode {
			return i
		}
	}
	return 0
}
func (m *GameMenu) displayRows() int { return min(displayModeRows, len(m.DisplayModes)) }
func (m *GameMenu) updateDisplayModes(in *platform.Input, w, h, x, y int) (bool, int) {
	if m.displayButton.Update(in) {
		m.displayOpen = !m.displayOpen
		m.displayHeld = -1
		m.displayOffset = max(0, min(max(0, len(m.DisplayModes)-m.displayRows()), m.displaySelected()-m.displayRows()+1))
		return true, MenuNone
	}
	if !m.displayOpen {
		return false, MenuNone
	}
	if in.HasKey(platform.KeyEscape) {
		m.displayOpen = false
		return true, MenuNone
	}
	if in.HasKey(platform.KeyUp) || in.HasKey(platform.KeyDown) {
		n := m.displaySelected()
		if in.HasKey(platform.KeyUp) {
			n--
		} else {
			n++
		}
		n = max(0, min(len(m.DisplayModes)-1, n))
		m.Settings.DisplayMode = m.DisplayModes[n]
		m.DisplayError = ""
		if n < m.displayOffset {
			m.displayOffset = n
		}
		if n >= m.displayOffset+m.displayRows() {
			m.displayOffset = n - m.displayRows() + 1
		}
		return true, MenuChanged
	}
	if in.HasKey(platform.KeyEnter) {
		m.displayOpen = false
		return true, MenuChanged
	}
	px, py := x+68, y+103
	row := PlayerMenuRowAt(in.MouseX, in.MouseY, px, py, 110, m.displayRows())
	if len(m.DisplayModes) > m.displayRows() {
		owned := m.displayScroll.dragging || m.displayScroll.arrows[0].held || m.displayScroll.arrows[1].held
		m.displayOffset = m.displayScroll.Update(in, px, py, len(m.DisplayModes), m.displayRows(), m.displayOffset)
		over := in.MouseX >= px+90 && in.MouseX < px+106 && in.MouseY >= py+8 && in.MouseY < py+8+m.displayScroll.height+13
		if owned || over {
			m.displayHeld = -1
			m.displayHover = -1
			return true, MenuNone
		}
	}
	m.displayHover = row
	if row >= 0 && in.Wheel != 0 {
		step := 1
		if in.Wheel > 0 {
			step = -1
		}
		m.displayOffset = max(0, min(len(m.DisplayModes)-m.displayRows(), m.displayOffset+step))
		m.displayHeld = -1
	}
	if in.Pressed {
		m.displayHeld = -1
		if row >= 0 {
			m.displayHeld = m.displayOffset + row
		} else {
			m.displayOpen = false
		}
	}
	if in.Clicked {
		if row >= 0 && m.displayHeld == m.displayOffset+row {
			m.Settings.DisplayMode = m.DisplayModes[m.displayHeld]
			m.DisplayError = ""
			m.displayOpen = false
			m.displayHeld = -1
			return true, MenuChanged
		}
		m.displayHeld = -1
	}
	return true, MenuNone // dropdown owns its pointer and navigation until dismissed
}
func (m *GameMenu) drawDisplayModes(s *render.Sprites, f *Font, solid *render.Texture, x, y int) {
	if m.Page != MenuGraphics || !m.displayOpen || solid == nil {
		return
	}
	labels := make([]string, 0, m.displayRows())
	for _, mode := range m.DisplayModes[m.displayOffset : m.displayOffset+m.displayRows()] {
		labels = append(labels, mode.Label())
	}
	hover := m.displayHover
	if hover < 0 {
		hover = m.displaySelected() - m.displayOffset
	}
	m.displayPopup.Draw(s, f, solid, x+68, y+103, 110, labels, hover)
	if len(m.DisplayModes) > m.displayRows() {
		m.displayScroll.Draw(s, x+68, y+103, len(m.DisplayModes), m.displayRows(), m.displayOffset)
	}
}

func (m *GameMenu) SetCurrentDisplayMode(mode platform.DisplayMode) {
	mode = platform.WindowMode(mode)
	if mode.Width == 0 {
		return
	}
	m.Settings.DisplayMode, m.original.DisplayMode = mode, mode
}
