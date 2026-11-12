package ui

import (
	"fmt"

	"github.com/Spelunkern/Shaiya-GX/internal/data/text"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

// ServerList is the server window (constructor 0x4eb520, draw 0x4ea5d0).
type ServerList struct {
	panel, white *render.Texture
	OK, Cancel   *Button
	Servers      []proto.Server
	Selected     int
	Scroll       int
	msg          text.SysMsg
	lastClick    float64
	clock        float64
	Busy         bool
}

// Original layout: floats 0x70a1b4..0x70a1d0, text offset +4 (static
// initializer 0x6b6700), row pitch 0x16, 13 visible rows.
const (
	slOKX, slOKY         = 67, 397
	slCancelX, slCancelY = 180, 397
	slRowX, slRowY       = 31, 78
	slRowW, slRowH       = 240, 19
	slTextPad            = 4
	slPitch              = 0x16
	slVisible            = 0xd
	slStatusX            = 0xe1
)

func NewServerList(t Textures, msg text.SysMsg, white *render.Texture) (*ServerList, error) {
	s := &ServerList{msg: msg, white: white, Selected: -1}
	var err error
	if s.panel, err = t.Interface("login_server.tga"); err != nil {
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
	s.OK = &Button{Tex: ok, OffX: slOKX, OffY: slOKY, DrawW: 0x80, DrawH: 0x20, HitW: 0x41, HitH: 0x19, UV: FourRowUV}
	s.Cancel = &Button{Tex: cancel, OffX: slCancelX, OffY: slCancelY, DrawW: 0x80, DrawH: 0x20, HitW: 0x41, HitH: 0x19, UV: FourRowUV}
	return s, nil
}

func serverOrigin(w, h int) (int, int) { return w/2 - 0x9c, h - h/20 - 0x1c2 }

type ServerAction int

const (
	ServerNone ServerAction = iota
	ServerConnect
	ServerBack
)

func (s *ServerList) Update(in *platform.Input, w, h int, dt float64) ServerAction {
	s.clock += dt
	ox, oy := serverOrigin(w, h)
	for _, b := range []*Button{s.OK, s.Cancel} {
		b.OriginX, b.OriginY = ox, oy
		b.Disabled = s.Busy
	}
	s.OK.Disabled = s.Busy || s.Selected < 0
	act := ServerNone
	if s.OK.Update(in) {
		act = ServerConnect
	}
	if s.Cancel.Update(in) {
		act = ServerBack
	}
	if s.Busy {
		return act
	}
	// NO VERIFICADO: row picking and double click (update routine not ported).
	if in.Pressed {
		for i := 0; i < slVisible && s.Scroll+i < len(s.Servers); i++ {
			x, y := ox+slRowX, oy+slRowY+i*slPitch
			if in.MouseX >= x && in.MouseX < x+slRowW && in.MouseY >= y && in.MouseY < y+slRowH {
				idx := s.Scroll + i
				if idx == s.Selected && s.clock-s.lastClick < 0.4 {
					act = ServerConnect
				}
				s.Selected, s.lastClick = idx, s.clock
			}
		}
	}
	if in.HasKey(platform.KeyEnter) && s.Selected >= 0 {
		act = ServerConnect
	}
	if in.HasKey(platform.KeyEscape) {
		act = ServerBack
	}
	return act
}

func argb(c uint32) [4]float32 {
	return [4]float32{float32(c>>16&0xff) / 255, float32(c>>8&0xff) / 255, float32(c&0xff) / 255, float32(c>>24) / 255}
}

func (s *ServerList) Draw(sp *render.Sprites, f *Font, w, h int) {
	ox, oy := serverOrigin(w, h)
	sp.DrawImage(s.panel, float32(ox), float32(oy))
	for i := 0; i < slVisible && s.Scroll+i < len(s.Servers); i++ {
		idx := s.Scroll + i
		sv := s.Servers[idx]
		rowY := oy + i*slPitch
		if idx == s.Selected {
			sp.Draw(s.white, float32(ox+slRowX), float32(rowY+slRowY), slRowW, slRowH, 0, 0, 1, 1, argb(0x7aa0a0a0))
		}
		f.Draw(sp, fmt.Sprintf("%d.%s", idx+1, sv.Name), ox+slRowX+slTextPad, rowY+slRowY+slTextPad, white, 0)
		status, color := s.status(sv)
		f.Draw(sp, status, ox+slStatusX, rowY+slRowY+slTextPad, argb(color), 0)
	}
	s.OK.Draw(sp)
	s.Cancel.Draw(sp)
}

// status follows 0x4ea820..0x4ea9e2 (locale 6).
func (s *ServerList) status(sv proto.Server) (string, uint32) {
	switch sv.Status {
	case 1:
		return "Lock", 0xfff3a0a0
	case 2:
		return "Closed", 0xfff3a0a0
	case 0:
	default:
		return "", 0
	}
	cap := float32(sv.Capacity)
	users := int(sv.Users)
	switch {
	case users > int(cap*0.85):
		return "Full", 0xfff3a0a0
	case users > int(cap*0.45):
		return s.msg.Get(0x2760), 0xfff3dea0
	case users > int(cap*0.3):
		return s.msg.Get(0x2761), 0xffa8f3a0
	default:
		return s.msg.Get(0x2762), 0xffa0cff3
	}
}
