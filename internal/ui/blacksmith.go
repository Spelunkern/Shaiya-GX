package ui

import (
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

const BlacksmithWidth, BlacksmithHeight = 512, 446
const (
	ForgeLapis = iota
	ForgeEnchant
	ForgeRemake
)

type Blacksmith struct {
	Position                               WindowPosition
	Mode, GemSlot                          int
	frame, lapis, enchantFrame             *render.Texture
	close, link, extract, enchant, compose Button
	tabs                                   [3]Button
}

// Native NPC sheets store disabled before pressed, unlike generic menu sheets.
var npcVerticalUV = [5]UV{{0, 0, 1, .25}, {0, .25, 1, .5}, {0, .75, 1, 1}, {0, .75, 1, 1}, {0, .5, 1, .75}}

func npcHorizontalUV(row int) (uv [5]UV) {
	for i, col := range []int{0, 1, 3, 3, 2} {
		uv[i] = UV{float32(col) / 4, float32(row) / 2, float32(col+1) / 4, float32(row+1) / 2}
	}
	return
}
func NewBlacksmith(t Textures) (*Blacksmith, error) {
	ld := &loader{t: t}
	v := &Blacksmith{frame: ld.get("Remake3.png"), lapis: ld.get("rapis.png"), enchantFrame: ld.get("rapisian.png")}
	v.close = nativeClose(ld.get("talk1_close_button.png"), 483, 3)
	sheet := ld.get("rapis_button2.png")
	v.link = Button{Tex: sheet, OffX: 292, OffY: 374, DrawW: 128, DrawH: 32, HitW: 124, HitH: 27}
	v.extract = Button{Tex: sheet, OffX: 427, OffY: 374, DrawW: 64, DrawH: 32, HitW: 64, HitH: 27}
	for i, row := range []int{0, 1, 3, 3, 2} {
		v.link.UV[i] = UV{0, float32(row) / 4, .5, float32(row+1) / 4}
		v.extract.UV[i] = UV{.5, float32(row) / 4, .75, float32(row+1) / 4}
	}
	v.enchant = Button{Tex: ld.get("repair_button4.png"), OffX: 348, OffY: 374, DrawW: 128, DrawH: 32, HitW: 65, HitH: 27, UV: npcVerticalUV}
	v.compose = Button{Tex: ld.get("Btn_Remodeling.png"), OffX: 348, OffY: 374, DrawW: 128, DrawH: 32, HitW: 65, HitH: 26, UV: npcVerticalUV}
	for i, tex := range []*render.Texture{ld.get("repair_button3.png"), ld.get("repair_button4.png"), ld.get("Btn_Remodeling.png")} {
		width := 65
		if i == 0 {
			width = 96
		}
		v.tabs[i] = Button{Tex: tex, OffX: 78, OffY: 270 + i*30, DrawW: 128, DrawH: 32, HitW: width, HitH: 26, UV: npcVerticalUV}
	}
	return v, ld.err
}
func (v *Blacksmith) Origin(w, h int) (int, int) {
	return v.Position.Origin(w, h, BlacksmithWidth, BlacksmithHeight, min(32, max(0, w-BlacksmithWidth)), min(82, max(0, h-BlacksmithHeight)))
}
func BlacksmithSlot(slot int) (int, int) { return (&Blacksmith{}).Slot(slot) }
func (v *Blacksmith) Slot(slot int) (int, int) {
	if v.Mode == ForgeLapis {
		switch slot {
		case 0:
			return 367, 93
		case 1:
			return 270 + min(5, max(0, v.GemSlot))*39, 206
		case 3:
			return 367, 288
		}
		return -100, -100
	}
	if slot == 0 {
		if v.Mode == ForgeRemake {
			return 370, 276
		}
		return 367, 288
	}
	if v.Mode == ForgeRemake {
		if slot == 2 {
			return 371, 105
		}
		return -100, -100
	}
	if slot == 1 {
		return 309, 131
	}
	if slot == 2 {
		return 427, 131
	}
	return -100, -100
}
func BlacksmithHit(x, y int) int { return (&Blacksmith{}).Hit(x, y) }
func (v *Blacksmith) Hit(x, y int) int {
	for slot := 0; slot < 4; slot++ {
		sx, sy := v.Slot(slot)
		if sx >= 0 && x >= sx && x < sx+32 && y >= sy && y < sy+32 {
			return slot
		}
	}
	return -1
}
func (v *Blacksmith) place(x, y int, busy, ready, extractReady, enchantReady, composeReady bool) {
	for i := range v.tabs {
		b := &v.tabs[i]
		b.OriginX, b.OriginY, b.Disabled = x, y, busy || v.Mode == i
	}
	for _, b := range []*Button{&v.close, &v.link, &v.extract, &v.enchant, &v.compose} {
		b.OriginX, b.OriginY = x, y
	}
	v.link.Disabled = busy || !ready
	v.extract.Disabled = busy || !extractReady
	v.enchant.Disabled = busy || !enchantReady
	v.compose.Disabled = busy || !composeReady
}
func (v *Blacksmith) Update(in *platform.Input, w, h int, busy, ready, extractReady, enchantReady, composeReady bool) (bool, bool, bool, bool, bool) {
	x, y := v.Origin(w, h)
	v.place(x, y, busy, ready, extractReady, enchantReady, composeReady)
	for i := range v.tabs {
		if v.tabs[i].Update(in) {
			v.Mode = i
			return false, false, false, false, false
		}
	}
	return v.close.Update(in), v.Mode == ForgeLapis && v.link.Update(in), v.Mode == ForgeLapis && v.extract.Update(in), v.Mode == ForgeEnchant && v.enchant.Update(in), v.Mode == ForgeRemake && v.compose.Update(in)
}
func (v *Blacksmith) Draw(s *render.Sprites, f *Font, w, h int, name string, busy, ready, extractReady, enchantReady, composeReady bool) {
	x, y := v.Origin(w, h)
	frame := v.frame
	if v.Mode == ForgeLapis {
		frame = v.lapis
	} else if v.Mode == ForgeEnchant {
		frame = v.enchantFrame
	}
	s.Draw(frame, float32(x), float32(y), 512, 446, 0, 0, 1, 446.0/512, white)
	text := "Place equipment and lapis to link. For extraction, place an extraction hammer and click the lapis to recover."
	if v.Mode == ForgeEnchant {
		text = "Place equipment, lapisia and enhancement material. Enhancement can fail."
	}
	if v.Mode == ForgeRemake {
		text = "Place equipment and a recreation rune to recreate its attributes."
	}
	for i, line := range wrapLines(f, text, 202) {
		f.Draw(s, line, x+26, y+94+i*17, RGBColor(0xc8c8c8), 202)
	}
	v.place(x, y, busy, ready, extractReady, enchantReady, composeReady)
	for i := range v.tabs {
		v.tabs[i].Draw(s)
	}
	switch v.Mode {
	case ForgeLapis:
		v.link.Draw(s)
		v.extract.Draw(s)
	case ForgeEnchant:
		v.enchant.Draw(s)
	case ForgeRemake:
		v.compose.Draw(s)
	}
	if busy {
		f.Draw(s, "Waiting for server...", x+26, y+265, white, 208)
	}
	v.close.Draw(s)
}
