package ui

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/text"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

const FactionNone, FactionExit = -1, -2

// FactionPanel is the native country hit area (0x47cb70/0x47d5e0).
func FactionPanel(w, h, country int) (int, int) {
	x := w/5 - 118
	if country == 1 {
		x = w - w/5 - 118
	}
	return x, h/2 - 324
}

// FactionScreen preserves the two native confirmations before 0x0109.
// Flags/Select.EFT and the presentation animation remain pending.
type FactionScreen struct {
	bg, center                 *render.Texture
	panels                     [2]*render.Texture
	dialogs                    [2][2]*render.Texture
	selectButton, cancelButton Button
	ok, cancel                 Button
	msg                        text.SysMsg
	Selected, step, held       int
	Busy                       bool
}

func NewFaction(t Textures, msg text.SysMsg) (*FactionScreen, error) {
	ld := &loader{t: t}
	v := &FactionScreen{bg: ld.get("select_country_bg2.jpg"), center: ld.get("select_country1.tga"), msg: msg}
	v.panels = [2]*render.Texture{ld.get("select_country2.tga"), ld.get("select_country3.tga")}
	for i, name := range []string{"light", "fury"} {
		v.dialogs[i] = [2]*render.Texture{ld.get("choose_" + name + ".tga"), ld.get("choose_submit_" + name + ".tga")}
	}
	tex := ld.get("select_country_button.tga")
	for i, b := range []*Button{&v.selectButton, &v.cancelButton} {
		*b = Button{Tex: tex, OffX: 35 + i*130, OffY: 478, DrawW: 128, DrawH: 32, HitW: 98, HitH: 25}
		for state, row := range []int{0, 1, 2, 2, 3} {
			b.UV[state] = UV{float32(i) / 2, float32(row) / 4, float32(i+1) / 2, float32(row+1) / 4}
		}
	}
	v.ok = Button{Tex: ld.get("ok_button.tga"), DrawW: 128, DrawH: 32, HitW: 65, HitH: 25, UV: FourRowUV}
	v.cancel = Button{Tex: ld.get("cancel_button.tga"), DrawW: 128, DrawH: 32, HitW: 65, HitH: 25, UV: FourRowUV}
	v.Reset()
	return v, ld.err
}
func (v *FactionScreen) Reset() {
	v.Selected, v.held, v.step, v.Busy = -1, -1, 0, false
	for _, b := range []*Button{&v.selectButton, &v.cancelButton, &v.ok, &v.cancel} {
		b.held = false
		b.state = StateNormal
	}
}
func (v *FactionScreen) controls(w, h int) {
	for _, b := range []*Button{&v.selectButton, &v.cancelButton} {
		b.OriginX, b.OriginY = w/2-148, h/2-267
		b.Disabled = v.Busy
	}
	v.selectButton.Disabled = v.Busy || v.Selected < 0
	y := h/2 + 197
	if v.step == 2 {
		y = h/2 + 79
	}
	v.ok.OriginX, v.ok.OriginY, v.ok.Disabled = w/2-78, y, v.Busy
	v.cancel.OriginX, v.cancel.OriginY, v.cancel.Disabled = w/2+35, y, v.Busy
}
func (v *FactionScreen) Update(in *platform.Input, w, h int) int {
	v.controls(w, h)
	if v.Busy {
		return FactionNone
	}
	if v.step > 0 {
		if v.cancel.Update(in) || in.HasKey(platform.KeyEscape) {
			v.step = 0
			return FactionNone
		}
		if v.ok.Update(in) || in.HasKey(platform.KeyEnter) {
			if v.step == 1 {
				v.step = 2
			} else {
				return v.Selected
			}
		}
		return FactionNone
	}
	if v.cancelButton.Update(in) || in.HasKey(platform.KeyEscape) {
		return FactionExit
	}
	hovered := -1
	for i := 0; i < 2; i++ {
		x, y := FactionPanel(w, h, i)
		if in.MouseX >= x && in.MouseX <= x+256 && in.MouseY >= y && in.MouseY <= y+645 {
			hovered = i
		}
	}
	if in.Pressed {
		v.held = hovered
	}
	if in.Clicked {
		if hovered >= 0 && hovered == v.held {
			v.Selected = hovered
		}
		v.held = -1
	}
	for _, r := range in.Chars {
		if r == 'l' || r == 'L' {
			v.Selected = 0
		}
		if r == 'f' || r == 'F' {
			v.Selected = 1
		}
	}
	if v.selectButton.Update(in) || (v.Selected >= 0 && in.HasKey(platform.KeyEnter)) {
		v.step = 1
	}
	return FactionNone
}
func (v *FactionScreen) Draw(s *render.Sprites, f *Font, w, h int) {
	v.controls(w, h)
	s.Draw(v.bg, 0, 0, float32(w), float32(h), 0, 0, 1, 1, white)
	for i := 0; i < 2; i++ {
		x, y := FactionPanel(w, h, i)
		s.Draw(v.panels[i], float32(x), float32(y), 256, 664, 0, 0, 1, 664.0/1024, white)
	}
	x, y := w/2-145, h/2-202
	s.Draw(v.center, float32(x), float32(y), 286, 462, 0, 0, 286.0/512, 462.0/512, white)
	id := 0x1404
	if v.Selected >= 0 {
		id += v.Selected + 1
	}
	top := h/2 - 172
	for _, line := range wrapLines(f, v.msg.Get(id), 230) {
		if top > h/2+175 {
			break
		}
		f.Draw(s, line, w/2-115, top, white, 230)
		top += 16
	}

	v.selectButton.Draw(s)
	v.cancelButton.Draw(s)
	if v.step > 0 && v.Selected >= 0 {
		height, top := 512, h/2-256
		if v.step == 2 {
			height, top = 256, h/2-128
		}
		s.Draw(v.dialogs[v.Selected][v.step-1], float32(w/2-256), float32(top), 512, float32(height), 0, 0, 1, 1, white)
		v.ok.Draw(s)
		v.cancel.Draw(s)
	}
}
