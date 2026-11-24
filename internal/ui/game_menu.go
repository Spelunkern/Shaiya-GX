package ui

import (
	"fmt"
	"math"

	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

type MenuPage int

const (
	MenuMain MenuPage = iota
	MenuGraphics
	MenuSound
	MenuUser
	MenuExtraKeys
	MenuKeys
	MenuHelp
)

const MenuNone, MenuClose, MenuCharacters, MenuExit, MenuChanged, MenuSave = 0, 1, 2, 3, 4, 5

// Game.exe 0x4fbbb0 constructs the widgets and 0x4fa800 places subpage
// controls. These coordinates take precedence over Phoenix's compact menu.
type GameMenu struct {
	displayButton              Button
	displayScroll              panelScroll
	displayHover               int
	displayOpen                bool
	displayOffset, displayHeld int
	DisplayModes               []platform.DisplayMode
	DisplayError               string
	displayPopup               *PlayerMenu
	antialiasButton            Button
	antialiasOpen              bool
	antialiasHover             int
	antialiasHeld              int
	antialiasChoices           []int
	Position                   WindowPosition
	Page                       MenuPage
	frames                     [7]*render.Texture
	buttons                    [8]Button
	close, back                Button
	ok, cancel, reset          Button
	slider, check              *render.Texture
	help                       *menuHelp
	// CharSelect is the character-select variant of the options window
	// (Game.exe state 3 of 0x4fbbb0/0x4f9bf0): docked at the top right, and a
	// frameless main page with only Graphics, Sound and User buttons.
	CharSelect   bool
	bindingSlot  int // one-based, zero while no binding is being edited
	dragSlider   int
	pressedCheck *bool
	Settings     ClientSettings
	original     ClientSettings
}

// Only settings with a runtime consumer are editable. Other original controls
// remain visible in their original art until their subsystems are implemented.
type ClientSettings struct {
	DisplayMode                               platform.DisplayMode
	ChatRows                                  [2]int
	ChatFilters                               uint8 `json:",omitempty"`
	ChatTransparent                           bool  `json:",omitempty"`
	Music, Effects, Ambient                   float64
	CharacterSound, RiskSensor                bool
	Glow, Brightness, Details, VisualDistance int
	HideCloak, HideHelmet                     bool
	MouseSensitivity                          int
	UseMouse, InvertY, SwapMouse              bool
	Bindings                                  platform.KeyBindings
	OwnName, PlayerNames, MonsterNames        bool
	BannedWords                               bool `json:",omitempty"`
	Shadows, Water, SkillEffects              bool
	Antialias                                 int `json:",omitempty"`
	// "Other Setup" of the User Option page. The original's All Windows and
	// Re-open UI checkboxes had no effect and were removed.
	WarningMsg                             bool
	RejectBattle, RejectParty, RejectTrade bool
	ShakeScreen, WordBalloon, HideDamage   bool
}

func DefaultClientSettings() ClientSettings {
	return ClientSettings{Antialias: AntialiasOff, Glow: 2, Brightness: 2, Details: 2, VisualDistance: 2, ChatRows: [2]int{5, 5}, Music: 1, Effects: 1, Ambient: 1, CharacterSound: true, RiskSensor: true, MouseSensitivity: 3, UseMouse: true,
		OwnName: true, PlayerNames: true, MonsterNames: true, Shadows: true, Water: true, SkillEffects: true,
		WarningMsg: true, ShakeScreen: true, WordBalloon: true}
}

func (c ClientSettings) Valid() bool {
	for _, rows := range c.ChatRows {
		if rows < 0 || rows > 20 {
			return false
		}
	}
	return c.Bindings.Valid() && c.DisplayMode.Valid() && c.Antialias >= 0 && c.Antialias < antialiasLevels && c.Glow >= 0 && c.Glow <= 2 && c.Brightness >= 0 && c.Brightness <= 4 && c.Details >= 0 && c.Details <= 2 && c.VisualDistance >= 0 && c.VisualDistance <= 2 && !math.IsNaN(c.Music) && !math.IsNaN(c.Effects) && c.Music >= 0 && c.Music <= 1 && c.Effects >= 0 && c.Effects <= 1 && !math.IsNaN(c.Ambient) && c.Ambient >= 0 && c.Ambient <= 1 && c.MouseSensitivity >= 0 && c.MouseSensitivity <= 6
}

func NewGameMenu(t Textures) (*GameMenu, error) {
	ld := &loader{t: t}
	m := &GameMenu{Settings: DefaultClientSettings(), dragSlider: -1}
	for i, name := range []string{"option_main", "option_graphic", "option_sound", "option_user", "option_plus_key", "option_key", "option_help"} {
		m.frames[i] = ld.get(name + ".png")
	}
	tex := ld.get("option_main_button.png")
	for i := range m.buttons {
		b := &m.buttons[i]
		*b = Button{Tex: tex, OffX: 64, OffY: 77 + 32*i, DrawW: 128, DrawH: 32, HitW: 128, HitH: 22}
		for state, row := range []int{0, 32, 64, 64, 96} {
			x, y := (i%4)*128, (i/4)*128+row
			b.UV[state] = UV{float32(x) / 512, float32(y) / 256, float32(x+128) / 512, float32(y+32) / 256}
		}
	}
	m.help = newMenuHelp(ld)
	m.close = nativeClose(ld.get("talk1_close_button.png"), 228, 5)
	m.back = Button{Tex: ld.get("option_main_button2.png"), DrawW: 128, DrawH: 32, HitW: 128, HitH: 22, UV: FourRowUV}
	m.ok = Button{Tex: ld.get("ok_button.png"), DrawW: 128, DrawH: 32, HitW: 65, HitH: 25, UV: FourRowUV}
	m.cancel = Button{Tex: ld.get("cancel_button.png"), DrawW: 128, DrawH: 32, HitW: 65, HitH: 25, UV: FourRowUV}
	m.reset = Button{Tex: ld.get("option_reset_button.png"), DrawW: 128, DrawH: 32, HitW: 81, HitH: 25, UV: FourRowUV}
	m.DisplayModes = []platform.DisplayMode{{Width: 1024, Height: 768}}
	m.displayPopup = &PlayerMenu{frame: ld.get("TipBox.png")}
	m.displayScroll = newPanelScroll(ld.get("scrollbar.png"), 90, 8, displayModeRows*PlayerMenuRowHeight+8)
	m.displayButton = Button{Tex: ld.get("option_graphic_resol.png"), OffX: 135, OffY: 81, DrawW: 32, DrawH: 32, HitW: 20, HitH: 20}
	for i, state := range []int{0, 1, 2, 2, 3} {
		m.displayButton.UV[i] = UV{float32(state) / 4, 0, float32(state+1) / 4, 1}
	}
	m.antialiasButton = m.displayButton
	m.antialiasButton.OffX = 268 // right end of the old "Color Bit" box
	m.antialiasHover, m.antialiasHeld = -1, -1
	m.slider, m.check = ld.get("option_graphic_slider.png"), ld.get("chat_check.png")
	return m, ld.err
}

func (m *GameMenu) Size() (int, int) {
	switch m.Page {
	case MenuGraphics:
		return 308, 540
	case MenuSound:
		return 256, 418
	case MenuUser, MenuKeys:
		return 380, 446
	case MenuExtraKeys:
		return 380, 306
	case MenuHelp:
		return 454, 448
	default:
		return 256, 468
	}
}

func (m *GameMenu) Origin(w, h int) (int, int) {
	width, height := m.Size()
	if m.CharSelect {
		return w - width - 5, 10
	}
	return m.Position.Origin(w, h, width, height, (w-width)/2, (h-height)/2)
}

func (m *GameMenu) Open(page MenuPage) {
	if page == MenuHelp {
		m.help.reset()
	}
	if page != MenuMain {
		m.original = m.Settings
	}
	m.Page = page
	m.bindingSlot = 0
	m.displayOpen = false
	m.displayHeld = -1
	m.displayButton.held = false
	m.displayHover = -1
	m.antialiasButton.held = false
	m.antialiasOpen, m.antialiasHover, m.antialiasHeld = false, -1, -1
	m.displayScroll.dragging = false
	for i := range m.displayScroll.arrows {
		m.displayScroll.arrows[i].held = false
	}
	m.dragSlider = -1
	m.pressedCheck = nil
	for i := range m.buttons {
		m.buttons[i].held = false
	}
	for _, b := range []*Button{&m.ok, &m.cancel, &m.reset, &m.close, &m.back} {
		b.held = false
	}
}

func (m *GameMenu) SavedSettings() ClientSettings {
	if m.Page != MenuMain {
		return m.original
	}
	return m.Settings
}

func (m *GameMenu) DiscardPage() {
	if m.Page != MenuMain {
		m.Settings = m.original
	}
	m.Open(MenuMain)
}

func (m *GameMenu) controls(w, h int) (int, int, int, int) {
	x, y := m.Origin(w, h)
	width, height := m.Size()
	m.close.OriginX, m.close.OriginY = x, y
	m.close.OffX = width - 28
	m.back.OriginX, m.back.OriginY = x, y
	m.back.OffX, m.back.OffY = 74, 388
	positions := [3][2]int{}
	switch m.Page {
	case MenuGraphics:
		positions = [3][2]int{{149, 488}, {219, 488}, {21, 488}}
	case MenuSound:
		positions = [3][2]int{{105, 362}, {175, 362}, {15, 362}}
	case MenuUser, MenuKeys:
		positions = [3][2]int{{220, 397}, {290, 397}, {27, 397}}
	case MenuExtraKeys:
		positions = [3][2]int{{220, 244}, {290, 244}, {27, 244}}
	case MenuHelp:
		positions = [3][2]int{{268, 397}, {349, 397}, {0, 0}}
	}
	for i, b := range []*Button{&m.ok, &m.cancel, &m.reset} {
		b.OriginX, b.OriginY = x, y
		b.OffX, b.OffY = positions[i][0], positions[i][1]
	}
	m.displayButton.OriginX, m.displayButton.OriginY = x, y
	m.displayButton.Disabled = len(m.DisplayModes) < 2
	m.antialiasButton.OriginX, m.antialiasButton.OriginY = x, y
	for i := range m.buttons {
		m.buttons[i].OriginX, m.buttons[i].OriginY = x, y
		m.buttons[i].OffX, m.buttons[i].OffY = 64, 77+32*i
		if m.CharSelect && i < 3 {
			m.buttons[i].OffX, m.buttons[i].OffY = 64+0x19, 0x11+32*i
		}
	}
	return x, y, width, height
}

func (m *GameMenu) Update(in *platform.Input, w, h int) int {
	x, y, _, _ := m.controls(w, h)
	if m.updateBinding(in, x, y) {
		return MenuNone
	}
	if m.Page == MenuGraphics {
		if !m.displayOpen {
			if handled, result := m.updateAntialias(in, x, y); handled {
				return result
			}
		}
		if handled, result := m.updateDisplayModes(in, w, h, x, y); handled {
			m.antialiasOpen = false
			return result
		}
	}
	if in.Pressed {
		m.pressedCheck = nil
		m.dragSlider = -1
	}
	if m.CharSelect && m.Page == MenuMain {
		if in.HasKey(platform.KeyEscape) {
			return MenuClose
		}
		for i := 0; i < 3; i++ {
			if m.buttons[i].Update(in) {
				m.Open(MenuPage(i + 1))
				return MenuNone
			}
		}
		return MenuNone
	}
	if m.close.Update(in) || in.HasKey(platform.KeyEscape) {
		if m.Page != MenuMain {
			m.Settings = m.original
			m.Open(MenuMain)
			return MenuChanged
		}
		return MenuClose
	}
	if m.Page == MenuMain {
		if m.back.Update(in) {
			return MenuClose
		}
		for i := range m.buttons {
			if !m.buttons[i].Update(in) {
				continue
			}
			if i == 6 {
				return MenuCharacters
			}
			if i == 7 {
				return MenuExit
			}
			m.Open(MenuPage(i + 1))
			break
		}
		return MenuNone
	}
	if m.Page == MenuHelp {
		m.help.update(in, x, y)
	}
	if m.cancel.Update(in) {
		m.Settings = m.original
		m.Open(MenuMain)
		return MenuChanged
	}
	if m.ok.Update(in) {
		m.Open(MenuMain)
		return MenuSave
	}
	if m.Page != MenuHelp && m.reset.Update(in) {
		defaults := DefaultClientSettings()
		switch m.Page {
		case MenuSound:
			m.Settings.Music, m.Settings.Effects, m.Settings.Ambient = defaults.Music, defaults.Effects, defaults.Ambient
			m.Settings.CharacterSound, m.Settings.RiskSensor = defaults.CharacterSound, defaults.RiskSensor
		case MenuUser:
			m.Settings.UseMouse, m.Settings.InvertY, m.Settings.SwapMouse = defaults.UseMouse, defaults.InvertY, defaults.SwapMouse
			m.Settings.MouseSensitivity = defaults.MouseSensitivity
			m.Settings.OwnName, m.Settings.PlayerNames, m.Settings.MonsterNames = true, true, true
			m.Settings.BannedWords = defaults.BannedWords
			m.Settings.WarningMsg = defaults.WarningMsg
			m.Settings.RejectBattle, m.Settings.RejectParty, m.Settings.RejectTrade = defaults.RejectBattle, defaults.RejectParty, defaults.RejectTrade
			m.Settings.ShakeScreen, m.Settings.WordBalloon, m.Settings.HideDamage = defaults.ShakeScreen, defaults.WordBalloon, defaults.HideDamage
		case MenuKeys, MenuExtraKeys:
			start, end := m.bindingRange()
			for slot := start; slot < end; slot++ {
				m.Settings.Bindings.Bind(slot, defaults.Bindings.At(slot))
			}
		case MenuGraphics:
			m.Settings.Shadows, m.Settings.Water, m.Settings.SkillEffects = true, true, true
			m.Settings.Glow, m.Settings.Brightness, m.Settings.Details, m.Settings.VisualDistance = 2, 2, 2, 2
			m.Settings.HideCloak, m.Settings.HideHelmet = false, false
			m.Settings.DisplayMode = platform.DisplayMode{}
			m.Settings.Antialias = defaults.Antialias
		}
		return MenuChanged
	}
	changed := false
	if m.Page == MenuGraphics {
		values := []*int{&m.Settings.Glow, &m.Settings.Brightness, &m.Settings.Details, &m.Settings.VisualDistance}
		limits := []int{2, 4, 2, 2}
		for i, p := range values {
			sy := 184 + i*50
			if in.Pressed && in.MouseX >= x+27 && in.MouseX <= x+283 && in.MouseY >= y+sy-14 && in.MouseY < y+sy+14 {
				m.dragSlider = i
			}
			if m.dragSlider == i {
				value := int(math.Round(min(1, max(0, float64(in.MouseX-x-35)/240)) * float64(limits[i])))
				if *p != value {
					*p = value
					changed = true
				}
				if in.Clicked || !in.MouseDown {
					m.dragSlider = -1
				}
			}
		}
	}
	if m.Page == MenuSound || m.Page == MenuUser {
		count := 3
		if m.Page == MenuUser {
			count = 1
		}
		for i := 0; i < count; i++ {
			sx, sy, length := 35, 106+50*i, 185
			padX, padY := 5, 10
			if m.Page == MenuUser {
				sx, sy = 36, 140
				padX, padY = 7, 14
			}
			if in.Pressed && in.MouseX >= x+sx-padX && in.MouseX <= x+sx+length+padX && in.MouseY >= y+sy-padY && in.MouseY <= y+sy+padY {
				m.dragSlider = i
			}
			if m.dragSlider == i {
				value := min(1.0, max(0.0, float64(in.MouseX-x-sx)/float64(length)))
				if m.Page == MenuSound {
					if i == 0 {
						m.Settings.Music = math.Round(value*6) / 6
					} else if i == 1 {
						m.Settings.Effects = math.Round(value*6) / 6
					} else {
						m.Settings.Ambient = math.Round(value*6) / 6
					}
				} else {
					m.Settings.MouseSensitivity = int(math.Round(value * 6))
				}
				changed = true
				if in.Clicked || !in.MouseDown {
					m.dragSlider = -1
				}
			}
		}
	}
	for _, c := range m.checks() {
		over := in.MouseX >= x+c.x && in.MouseX < x+c.x+18 && in.MouseY >= y+c.y && in.MouseY < y+c.y+18
		if in.Pressed && over {
			m.pressedCheck = c.value
		}
		if in.Clicked && over && c.value != nil && m.pressedCheck == c.value {
			*c.value = !*c.value
			changed = true
		}
	}
	if in.Clicked || !in.MouseDown {
		m.pressedCheck = nil
	}
	if changed {
		return MenuChanged
	}
	return MenuNone
}

type menuCheck struct {
	x, y  int
	value *bool
}

func (m *GameMenu) checkSelected(c menuCheck) bool {
	if c.value == nil {
		return false
	}
	if c.value == &m.Settings.HideCloak || c.value == &m.Settings.HideHelmet {
		return !*c.value
	}
	return *c.value
}

func (m *GameMenu) checks() []menuCheck {
	s := &m.Settings
	switch m.Page {
	case MenuSound:
		return []menuCheck{{33, 284, &s.CharacterSound}, {33, 314, &s.RiskSensor}}
	case MenuUser:
		return []menuCheck{{34, 83, &s.UseMouse}, {137, 83, &s.SwapMouse}, {241, 83, &s.InvertY}, {34, 202, &s.OwnName}, {137, 202, &s.PlayerNames}, {34, 228, &s.MonsterNames}, {137, 228, &s.BannedWords},
			// Other Setup: the original grid without All Windows and Re-open UI.
			{34, 306, &s.RejectBattle}, {137, 306, &s.RejectParty}, {241, 306, &s.WarningMsg},
			{34, 332, &s.RejectTrade}, {137, 332, &s.ShakeScreen}, {241, 332, &s.WordBalloon},
			{34, 358, &s.HideDamage}}
	case MenuGraphics:
		return []menuCheck{{35, 404, &s.Shadows}, {156, 404, &s.Water}, {35, 429, &s.HideCloak}, {156, 429, &s.HideHelmet}, {35, 451, &s.SkillEffects}}
	}
	return nil
}

func (m *GameMenu) Draw(s *render.Sprites, f *Font, w, h int, solid ...*render.Texture) {
	x, y, width, height := m.controls(w, h)
	if m.CharSelect && m.Page == MenuMain {
		for i := 0; i < 3; i++ {
			m.buttons[i].Draw(s)
		}
		return
	}
	tex := m.frames[m.Page]
	if tex != nil {
		s.Draw(tex, float32(x), float32(y), float32(width), float32(height), 0, 0, float32(width)/float32(tex.W), float32(height)/float32(tex.H), white)
	}
	m.close.Draw(s)
	if m.Page == MenuMain {
		for i := range m.buttons {
			m.buttons[i].Draw(s)
		}
		m.back.Draw(s)
	} else {
		m.ok.Draw(s)
		m.cancel.Draw(s)
		if m.Page != MenuHelp {
			m.reset.Draw(s)
		}
	}
	knob := func(px, py int, value float64, length int) {
		// 0x4fbbb0: 15x28 thumb, [0,.5]x[0,.84375] in the 32x32 sheet.
		s.Draw(m.slider, float32(x+px+int(value*float64(length))-7), float32(y+py-14), 15, 28, 0, 0, .5, .84375, white)
	}
	switch m.Page {
	case MenuSound:
		knob(35, 106, m.Settings.Music, 185)
		knob(35, 156, m.Settings.Effects, 185)
		knob(35, 206, m.Settings.Ambient, 185)
	case MenuUser:
		knob(36, 140, float64(m.Settings.MouseSensitivity)/6, 185)
	case MenuGraphics:
		for i, v := range []int{m.Settings.Glow, m.Settings.Brightness, m.Settings.Details, m.Settings.VisualDistance} {
			limit := []int{2, 4, 2, 2}[i]
			knob(35, 184+i*50, float64(v)/float64(limit), 240)
		}
		modeLabel := m.Settings.DisplayMode.Label()
		if modeLabel == "" {
			modeLabel = fmt.Sprintf("%dx%d", w, h)
		}
		f.Draw(s, modeLabel, x+74, y+87, white, 65)
		m.displayButton.Draw(s)
		if m.DisplayError != "" {
			f.Draw(s, m.DisplayError, x+16, y+110, RGBColor(0xffa080), 275)
		}
		m.drawAntialias(s, f, x, y)
	case MenuKeys, MenuExtraKeys:
		first, end := m.bindingRange()
		for slot := first; slot < end; slot++ {
			bx, by := bindingOrigin(slot - first)
			key := platform.BindingName(m.Settings.Bindings.At(slot))
			if m.bindingSlot == slot+1 {
				key = Tr(12287)
			}
			f.Draw(s, key, x+bx+5, y+by+5, white, 50)
		}
	case MenuHelp:
		m.help.draw(s, x, y)
	}
	for _, c := range m.checks() {
		if m.checkSelected(c) {
			s.Draw(m.check, float32(x+c.x), float32(y+c.y), 16, 16, 0, 0, 1, 1, white)
		}
	}
	if len(solid) > 0 {
		m.drawDisplayModes(s, f, solid[0], x, y)
		m.drawAntialiasPopup(s, f, solid[0], x, y)
	}
}

func (m *GameMenu) SetChatRows(rows [2]int) { m.Settings.ChatRows = rows; m.original.ChatRows = rows }

func (m *GameMenu) SetChatOptions(filters uint8, transparent bool) {
	m.Settings.ChatFilters, m.Settings.ChatTransparent = filters, transparent
	m.original.ChatFilters, m.original.ChatTransparent = filters, transparent
}

func (m *GameMenu) BindingCapture() bool { return m.bindingSlot != 0 }

func (m *GameMenu) bindingRange() (int, int) {
	switch m.Page {
	case MenuKeys:
		return 0, 20
	case MenuExtraKeys:
		return 20, 30
	}
	return 0, 0
}

func bindingOrigin(index int) (int, int) { return 114 + index%2*170, 63 + index/2*31 }

func (m *GameMenu) updateBinding(in *platform.Input, x, y int) bool {
	if m.BindingCapture() {
		for _, key := range in.KeyPresses {
			if key == platform.PhysicalEscape || m.Settings.Bindings.Bind(m.bindingSlot-1, key) {
				m.bindingSlot = 0
				return true
			}
		}
		if len(in.KeyPresses) == 0 && in.HasKey(platform.KeyEscape) {
			m.bindingSlot = 0
			return true
		}
		if !in.Pressed {
			return true
		}
		m.bindingSlot = 0
	}
	first, end := m.bindingRange()
	if in.Pressed {
		for slot := first; slot < end; slot++ {
			bx, by := bindingOrigin(slot - first)
			if in.MouseX >= x+bx && in.MouseX < x+bx+55 && in.MouseY >= y+by && in.MouseY < y+by+23 {
				m.bindingSlot = slot + 1
				return true
			}
		}
	}
	return false
}
