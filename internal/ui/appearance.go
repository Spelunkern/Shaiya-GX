package ui

import (
	"fmt"

	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

// AppearanceEditor is a provisional layout using the original UI textures.
type AppearanceEditor struct {
	frame         *render.Texture
	arrows        [8]Button
	ok, cancel    Button
	Values        [4]uint8
	GenderEnabled bool
}

func NewAppearanceEditor(t Textures) (*AppearanceEditor, error) {
	l := &loader{t: t}
	e := &AppearanceEditor{frame: l.get("message.tga")}
	tex := l.get("market_button.tga")
	for i := range e.arrows {
		side := i % 2
		b := &e.arrows[i]
		*b = Button{Tex: tex, OffX: 142 + side*60, OffY: 25 + i/2*28, DrawW: 32, DrawH: 32, HitW: 23, HitH: 23}
		for state, row := range []int{0, 1, 2, 2, 3} {
			b.UV[state] = UV{float32(side) / 2, float32(row) / 4, float32(side+1) / 2, float32(row+1) / 4}
		}
	}
	e.ok = Button{Tex: l.get("ok_button.tga"), OffX: 48, OffY: 150, DrawW: 128, DrawH: 32, HitW: 65, HitH: 25, UV: FourRowUV}
	e.cancel = Button{Tex: l.get("cancel_button.tga"), OffX: 143, OffY: 150, DrawW: 128, DrawH: 32, HitW: 65, HitH: 25, UV: FourRowUV}
	return e, l.err
}

func (e *AppearanceEditor) Open(values [4]uint8, genderEnabled bool) {
	e.Values, e.GenderEnabled = values, genderEnabled
	for i := range e.arrows {
		e.arrows[i].held, e.arrows[i].state = false, StateNormal
	}
	e.ok.held, e.cancel.held = false, false
}

func appearanceEditorOrigin(w, h int) (int, int) { return max(0, w/2-270), max(0, h/2-128) }

func (e *AppearanceEditor) Update(in *platform.Input, w, h int) int {
	x, y := appearanceEditorOrigin(w, h)
	for i := range e.arrows {
		b := &e.arrows[i]
		b.OriginX, b.OriginY, b.Disabled = x, y, i/2 == 3 && !e.GenderEnabled
		if b.Update(in) {
			field, count := i/2, 5
			if field == 3 {
				count = 2
			}
			e.Values[field] = uint8((int(e.Values[field]) + count + i%2*2 - 1) % count)
		}
	}
	e.ok.OriginX, e.ok.OriginY, e.cancel.OriginX, e.cancel.OriginY = x, y, x, y
	if in.HasKey(platform.KeyEscape) || e.cancel.Update(in) {
		return MessageCancel
	}
	if in.HasKey(platform.KeyEnter) || e.ok.Update(in) {
		return MessageOK
	}
	return MessageNone
}

func (e *AppearanceEditor) Draw(s *render.Sprites, f *Font, w, h int) {
	x, y := appearanceEditorOrigin(w, h)
	// Leave room for four controls and the confirmation buttons within the
	// original message frame; the source image includes transparent padding.
	s.Draw(e.frame, float32(x), float32(y), float32(e.frame.W), float32(e.frame.H)*1.25, 0, 0, 1, 1, white)
	for i, label := range []string{"Hair", "Face", "Height", "Sex"} {
		f.Draw(s, label, x+55, y+30+i*28, white, 85)
		f.Draw(s, fmt.Sprintf("%d", e.Values[i]+1), x+181, y+30+i*28, white, 20)
	}
	for i := range e.arrows {
		e.arrows[i].OriginX, e.arrows[i].OriginY = x, y
		e.arrows[i].Disabled = i/2 == 3 && !e.GenderEnabled
		e.arrows[i].Draw(s)
	}
	e.ok.OriginX, e.ok.OriginY, e.cancel.OriginX, e.cancel.OriginY = x, y, x, y
	e.ok.Draw(s)
	e.cancel.Draw(s)
}
