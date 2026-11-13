package ui

import (
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

const BlacksmithWidth, BlacksmithHeight = 512, 446

type Blacksmith struct {
	Position                                                  WindowPosition
	frame                                                     *render.Texture
	close, link, extract, repair, repairAll, enchant, compose Button
}

func NewBlacksmith(t Textures) (*Blacksmith, error) {
	ld := &loader{t: t}
	v := &Blacksmith{frame: ld.get("Remake1.png")}
	v.close = nativeClose(ld.get("talk1_close_button.png"), 483, 3)
	v.link = Button{Tex: ld.get("repair_button2.png"), OffX: 292, OffY: 374, DrawW: 128, DrawH: 32, HitW: 124, HitH: 27,
		UV: [5]UV{{0, 0, .5, .25}, {0, .25, .5, .5}, {0, .5, .5, .75}, {0, .5, .5, .75}, {0, .75, .5, 1}}}
	v.extract = Button{Tex: ld.get("repair_button2.png"), OffX: 427, OffY: 374, DrawW: 64, DrawH: 32, HitW: 64, HitH: 27,
		UV: [5]UV{{.5, 0, 1, .25}, {.5, .25, 1, .5}, {.5, .5, 1, .75}, {.5, .5, 1, .75}, {.5, .75, 1, 1}}}
	v.repair = Button{Tex: ld.get("talk_blank_button.png"), OffX: 30, OffY: 300, DrawW: 128, DrawH: 32, HitW: 65, HitH: 26,
		UV: [5]UV{{0, 0, .25, .5}, {.25, 0, .5, .5}, {.5, 0, .75, .5}, {.5, 0, .75, .5}, {.75, 0, 1, .5}}}
	v.repairAll = Button{Tex: ld.get("repair_button1.png"), OffX: 30, OffY: 335, DrawW: 96, DrawH: 32, HitW: 96, HitH: 26,
		UV: [5]UV{{0, 0, .5, .25}, {0, .25, .5, .5}, {0, .5, .5, .75}, {0, .5, .5, .75}, {0, .75, .5, 1}}}
	v.enchant = Button{Tex: ld.get("repair_button4.png"), OffX: 427, OffY: 320, DrawW: 128, DrawH: 32, HitW: 64, HitH: 27, UV: FourRowUV}
	v.compose = Button{Tex: ld.get("talk_blank_button.png"), OffX: 145, OffY: 335, DrawW: 128, DrawH: 32, HitW: 65, HitH: 26,
		UV: [5]UV{{0, 0, .25, .5}, {.25, 0, .5, .5}, {.5, 0, .75, .5}, {.5, 0, .75, .5}, {.75, 0, 1, .5}}}
	return v, ld.err
}

func (v *Blacksmith) Origin(w, h int) (int, int) {
	return v.Position.Origin(w, h, BlacksmithWidth, BlacksmithHeight, min(32, max(0, w-BlacksmithWidth)), min(82, max(0, h-BlacksmithHeight)))
}

// The upper sockets accept equipment, lapis or lapisia, and material; the lower socket accepts an optional hammer.
func BlacksmithSlot(slot int) (int, int) {
	if slot == 0 {
		return 294, 124
	}
	if slot == 1 {
		return 368, 124
	}
	if slot == 2 {
		return 442, 124
	}
	return 368, 320
}

func BlacksmithHit(x, y int) int {
	for slot := 0; slot < 4; slot++ {
		sx, sy := BlacksmithSlot(slot)
		if x >= sx && x < sx+32 && y >= sy && y < sy+32 {
			return slot
		}
	}
	return -1
}

func (v *Blacksmith) Update(in *platform.Input, w, h int, busy, ready, extractReady, repairReady, enchantReady, composeReady bool) (bool, bool, bool, bool, bool, bool, bool) {
	x, y := v.Origin(w, h)
	v.close.OriginX, v.close.OriginY = x, y
	v.link.OriginX, v.link.OriginY, v.link.Disabled = x, y, busy || !ready
	v.extract.OriginX, v.extract.OriginY, v.extract.Disabled = x, y, busy || !extractReady
	v.repair.OriginX, v.repair.OriginY, v.repair.Disabled = x, y, busy || !repairReady
	v.repairAll.OriginX, v.repairAll.OriginY, v.repairAll.Disabled = x, y, busy
	v.enchant.OriginX, v.enchant.OriginY, v.enchant.Disabled = x, y, busy || !enchantReady
	v.compose.OriginX, v.compose.OriginY, v.compose.Disabled = x, y, busy || !composeReady
	return v.close.Update(in), v.link.Update(in), v.extract.Update(in), v.repair.Update(in), v.repairAll.Update(in), v.enchant.Update(in), v.compose.Update(in)
}

func (v *Blacksmith) Draw(s *render.Sprites, f *Font, w, h int, name string, busy, ready, extractReady, repairReady, enchantReady, composeReady bool) {
	x, y := v.Origin(w, h)
	s.DrawImage(v.frame, float32(x), float32(y))
	f.Draw(s, name, x+77, y+20, white, 165)
	f.Draw(s, "Blacksmith", x+329, y+15, white, 160)
	f.Draw(s, "Drag items from your inventory", x+25, y+100, white, 210)
	f.Draw(s, "to the appropriate slots.", x+25, y+116, white, 210)
	f.Draw(s, "Linking or enchanting may fail", x+25, y+148, white, 210)
	f.Draw(s, "and damage or destroy equipment.", x+25, y+164, white, 210)
	f.Draw(s, "For repair or extraction, select", x+25, y+196, white, 210)
	f.Draw(s, "the equipment only.", x+25, y+212, white, 210)
	f.Draw(s, "For Remake, add a rune.", x+25, y+236, white, 210)
	f.Draw(s, "Equipment", x+285, y+170, white, 72)
	f.Draw(s, "Lapis/Lapisia", x+356, y+170, white, 84)
	f.Draw(s, "Mat/Rune", x+436, y+170, white, 65)
	f.Draw(s, "Hammer (optional)", x+340, y+355, white, 112)
	if busy {
		f.Draw(s, "Waiting for server...", x+285, y+330, white, 200)
	}
	v.close.OriginX, v.close.OriginY = x, y
	v.link.OriginX, v.link.OriginY, v.link.Disabled = x, y, busy || !ready
	v.extract.OriginX, v.extract.OriginY, v.extract.Disabled = x, y, busy || !extractReady
	v.repair.OriginX, v.repair.OriginY, v.repair.Disabled = x, y, busy || !repairReady
	v.repairAll.OriginX, v.repairAll.OriginY, v.repairAll.Disabled = x, y, busy
	v.enchant.OriginX, v.enchant.OriginY, v.enchant.Disabled = x, y, busy || !enchantReady
	v.compose.OriginX, v.compose.OriginY, v.compose.Disabled = x, y, busy || !composeReady
	v.compose.Draw(s)
	f.Draw(s, "Remake", x+153, y+341, white, 52)
	v.enchant.Draw(s)
	v.repair.Draw(s)
	f.Draw(s, "Repair", x+38, y+306, white, 52)
	v.repairAll.Draw(s)
	v.close.Draw(s)
	v.link.Draw(s)
	v.extract.Draw(s)
}
