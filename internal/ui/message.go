package ui

import (
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

const MessageNone, MessageOK, MessageCancel = 0, 1, 2

// Message uses the native type 0/1 dialogs: 0x4c4d10, 0x4c49d0, 0x4c4ae0.
// The caller owns modality and any action associated with confirmation.
type Message struct {
	frame            *render.Texture
	ok, cancel       Button
	Text             string
	Visible, Confirm bool
}

func NewMessage(t Textures) (*Message, error) {
	ld := &loader{t: t}
	m := &Message{frame: ld.get("message.tga")}
	m.ok = Button{Tex: ld.get("ok_button.tga"), DrawW: 128, DrawH: 32, HitW: 65, HitH: 25, UV: FourRowUV}
	m.cancel = Button{Tex: ld.get("cancel_button.tga"), DrawW: 128, DrawH: 32, HitW: 65, HitH: 25, UV: FourRowUV}
	return m, ld.err
}

func (m *Message) Open(text string, confirm bool) {
	m.Text, m.Confirm, m.Visible = text, confirm, true
	for _, b := range []*Button{&m.ok, &m.cancel} {
		b.held, b.state = false, StateNormal
	}
}

func (m *Message) controls(w, h int) {
	x, y := w/2-128, h/2-128
	m.ok.OriginX, m.ok.OriginY, m.ok.OffX, m.ok.OffY = x, y, 95, 114
	if m.Confirm {
		m.ok.OffX = 48
	}
	m.cancel.OriginX, m.cancel.OriginY, m.cancel.OffX, m.cancel.OffY = x, y, 143, 114
}

func (m *Message) Update(in *platform.Input, w, h int) int {
	if !m.Visible {
		return MessageNone
	}
	m.controls(w, h)
	result := MessageNone
	if in.HasKey(platform.KeyEscape) || m.Confirm && m.cancel.Update(in) {
		result = MessageCancel
	} else if m.ok.Update(in) || in.HasKey(platform.KeyEnter) {
		result = MessageOK
	}
	if result != MessageNone {
		m.Visible = false
	}
	return result
}

func (m *Message) Draw(s *render.Sprites, f *Font, w, h int) {
	if !m.Visible {
		return
	}
	m.controls(w, h)
	x, y := w/2-128, h/2-128
	s.DrawImage(m.frame, float32(x), float32(y))
	lines := wrapLines(f, m.Text, 216)
	for i, line := range lines[:min(4, len(lines))] {
		f.Draw(s, line, x+25, y+53+i*14, white, 216)
	}
	m.ok.Draw(s)
	if m.Confirm {
		m.cancel.Draw(s)
	}
}
