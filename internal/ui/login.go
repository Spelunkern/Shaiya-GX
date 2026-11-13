package ui

import (
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

// Textures loads interface textures by file name.
type Textures interface {
	Interface(name string) (*render.Texture, error)
}

// Login is the login window (constructor 0x4b8ff0, drawn by 0x4ba650,
// logic in 0x4bd680). Every coordinate is the original's, relative to the
// window origin computed from the screen size.
type Login struct {
	logo, panel        *render.Texture
	OK, Cancel, SaveID *Button
	ID, Password       TextField
	Focus              int // +0x23c: 0 = ID, 1 = password
	Message            string
	caret              float64
	Busy               bool // waiting for the server: input locked
}

// Original offsets (floats at 0x709d40..0x709d74).
const (
	loginOKX, loginOKY         = 37, 159
	loginCancelX, loginCancelY = 122, 159
	loginCheckX, loginCheckY   = 147, 102
	loginIDX, loginIDY         = 84, 56
	loginPwX, loginPwY         = 84, 81
	loginTextClip              = 0x96 // last argument of the 0x543d30 calls
)

func NewLogin(t Textures) (*Login, error) {
	l := &Login{ID: TextField{Max: 18}, Password: TextField{Max: 16, Masked: true}}
	var err error
	if l.panel, err = t.Interface("login.tga"); err != nil {
		return nil, err
	}
	if l.logo, err = t.Interface("shaiyalogo01_new.TGA"); err != nil { // feature 15 (login3) is on for this client
		return nil, err
	}
	ok, err := t.Interface("ok_button.tga")
	if err != nil {
		return nil, err
	}
	cancel, err := t.Interface("cancel_button.tga")
	if err != nil {
		return nil, err
	}
	check, err := t.Interface("chat_check.tga")
	if err != nil {
		return nil, err
	}
	// 0x528830(origin, offset, 128x32 drawn, 65x25 hit, sheet 128x128).
	l.OK = &Button{Tex: ok, OffX: loginOKX, OffY: loginOKY, DrawW: 0x80, DrawH: 0x20, HitW: 0x41, HitH: 0x19, UV: FourRowUV}
	l.Cancel = &Button{Tex: cancel, OffX: loginCancelX, OffY: loginCancelY, DrawW: 0x80, DrawH: 0x20, HitW: 0x41, HitH: 0x19, UV: FourRowUV}
	// 0x528700 type 3: only the pressed/checked states show the mark.
	l.SaveID = &Button{Tex: check, OffX: loginCheckX, OffY: loginCheckY, DrawW: 0x10, DrawH: 0x10, HitW: 0x10, HitH: 0x10, Toggle: true,
		UV: [5]UV{{}, {}, {0, 0, 1, 1}, {0, 0, 1, 1}, {}}}
	return l, nil
}

// Origin is the window origin of 0x4b8ff0/0x4bd680.
func loginOrigin(w, h int) (int, int) { return w/2 - 0x6f, h - h/32 - 0x114 }

// LoginAction is what the player asked for this frame.
type LoginAction int

const (
	LoginNone LoginAction = iota
	LoginSubmit
	LoginQuit
)

func (l *Login) Update(in *platform.Input, w, h int, dt float64) LoginAction {
	ox, oy := loginOrigin(w, h)
	for _, b := range []*Button{l.OK, l.Cancel, l.SaveID} {
		b.OriginX, b.OriginY = ox, oy
		b.Disabled = l.Busy && b != l.SaveID
	}
	// Caret timer (0x8655c0): "%s_" while above 0.5, reset at 1.0.
	l.caret += dt
	if l.caret > 1.0 {
		l.caret = 0
	}
	act := LoginNone
	if l.OK.Update(in) {
		act = LoginSubmit
	}
	if l.Cancel.Update(in) {
		act = LoginQuit
	}
	l.SaveID.Update(in)
	if l.Busy {
		return act
	}
	if in.Pressed {
		// Clicking a field moves the focus (field boxes of login.tga).
		if in.MouseX >= ox+80 && in.MouseX < ox+206 {
			switch {
			case in.MouseY >= oy+52 && in.MouseY < oy+70:
				l.Focus = 0
			case in.MouseY >= oy+77 && in.MouseY < oy+95:
				l.Focus = 1
			}
		}
	}
	// Typed text goes to the field focused before this frame's Tab.
	if l.Focus == 0 {
		l.ID.Edit(in)
	} else {
		l.Password.Edit(in)
	}
	if in.HasKey(platform.KeyTab) {
		l.Focus ^= 1
	}
	if in.HasKey(platform.KeyEnter) {
		if l.Focus == 0 && l.Password.Text == "" {
			l.Focus = 1
		} else {
			act = LoginSubmit
		}
	}
	if in.HasKey(platform.KeyEscape) {
		act = LoginQuit
	}
	return act
}

func (l *Login) Draw(s *render.Sprites, f *Font, w, h int) {
	l.DrawLogo(s, w, h)
	l.drawForm(s, f, w, h)
}

// The logo belongs to the login scenery, including server selection.
func (l *Login) DrawLogo(s *render.Sprites, w, h int) {
	// Logo: x = w/2 - 256, y = h/16 (0x4bb8f0, locale 6).
	s.DrawImage(l.logo, float32(w/2-0x100), float32(h/16))
}

func (l *Login) drawForm(s *render.Sprites, f *Font, w, h int) {
	ox, oy := loginOrigin(w, h)
	s.DrawImage(l.panel, float32(ox), float32(oy))
	l.OK.Draw(s)
	l.Cancel.Draw(s)
	l.SaveID.Draw(s)
	id, pw := l.ID.Text, l.Password.Shown()
	if !l.Busy && l.caret > 0.5 {
		if l.Focus == 0 {
			id += "_"
		} else {
			pw += "_"
		}
	}
	f.Draw(s, id, ox+loginIDX, oy+loginIDY, white, loginTextClip)
	f.Draw(s, pw, ox+loginPwX, oy+loginPwY, white, loginTextClip)
	if l.Message != "" {
		// NO VERIFICADO: the original shows messages in its own box.
		mw := f.Width(l.Message)
		f.Draw(s, l.Message, ox+0x6f-mw/2, oy+0xb8, white, 0)
	}
}
