package ui

import (
	"fmt"
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

const GatekeeperWidth, GatekeeperHeight = 512, 366

type Gatekeeper struct {
	Position              WindowPosition
	frame                 *render.Texture
	close                 Button
	moves                 [3]Button
	welcomeScroll         panelScroll
	textOffset, textLines int
	Currency              string
}

func NewGatekeeper(t Textures) (*Gatekeeper, error) {
	ld := &loader{t: t}
	v := &Gatekeeper{frame: ld.get("gatekeeper.png")}
	v.welcomeScroll = newPanelScroll(ld.get("scrollbar.png"), 228, 82, 167)
	v.close = nativeClose(ld.get("talk1_close_button.png"), 483, 3)
	tex := ld.get("gatekeeper_button.png")
	for i := range v.moves {
		v.moves[i] = Button{Tex: tex, OffX: 434, OffY: 80 + i*101, DrawW: 64, DrawH: 64, HitW: 58, HitH: 40,
			UV: [5]UV{{0, 0, .25, 1}, {.25, 0, .5, 1}, {.5, 0, .75, 1}, {.5, 0, .75, 1}, {.75, 0, 1, 1}}}
	}
	return v, ld.err
}
func (v *Gatekeeper) Origin(w, h int) (int, int) {
	return v.Position.Origin(w, h, GatekeeperWidth, GatekeeperHeight, min(32, max(0, w-GatekeeperWidth)), min(82, max(0, h-GatekeeperHeight)))
}
func (v *Gatekeeper) ResetWelcomeScroll() { v.textOffset = 0 }
func gatePresent(g sdata.NPCGate) bool    { return g.MapID != 0 || g.X != 0 || g.Z != 0 }
func (v *Gatekeeper) controls(w, h int, gates [3]sdata.NPCGate, busy bool) {
	x, y := v.Origin(w, h)
	v.close.OriginX, v.close.OriginY = x, y
	for i := range v.moves {
		v.moves[i].OriginX, v.moves[i].OriginY = x, y
		v.moves[i].Disabled = busy || !gatePresent(gates[i])
	}
}
func (v *Gatekeeper) Update(in *platform.Input, w, h int, gates [3]sdata.NPCGate, busy bool) (int, bool) {
	v.controls(w, h, gates, busy)
	x, y := v.Origin(w, h)
	if in.MouseX >= x+16 && in.MouseX < x+242 && in.MouseY >= y+82 && in.MouseY < y+249 && in.Wheel != 0 {
		if in.Wheel < 0 {
			v.textOffset += 3
		} else {
			v.textOffset = max(0, v.textOffset-3)
		}
	}
	v.textOffset = v.welcomeScroll.Update(in, x, y, v.textLines, 10, v.textOffset)
	closed := v.close.Update(in)
	for i := range v.moves {
		if v.moves[i].Update(in) {
			return i, closed
		}
	}
	return -1, closed
}
func (v *Gatekeeper) Draw(s *render.Sprites, f *Font, w, h int, name, welcome string, portrait *render.Texture, gates [3]sdata.NPCGate, busy bool) {
	v.controls(w, h, gates, busy)
	x, y := v.Origin(w, h)
	s.Draw(v.frame, float32(x), float32(y), 512, 366, 0, 0, 1, 366.0/512, white)
	if portrait != nil {
		s.Draw(portrait, float32(x+7), float32(y+7), 64, 64, 0, 0, 1, 1, white)
	}
	f.Draw(s, name, x+74+(164-min(164, f.Width(name)))/2, y+22, white, 164)
	lines := wrapLines(f, welcome, 193)
	v.textLines = len(lines)
	v.textOffset = min(v.textOffset, max(0, len(lines)-10))
	for i, line := range lines[v.textOffset:min(len(lines), v.textOffset+10)] {
		f.Draw(s, line, x+26, y+92+i*15, white, 193)
	}
	v.welcomeScroll.Draw(s, x, y, len(lines), 10, v.textOffset)
	for i, g := range gates {
		if !gatePresent(g) {
			continue
		}
		label := g.Name
		if label == "" {
			label = fmt.Sprintf("Map %d", g.MapID)
		}
		f.Draw(s, label, x+292, y+84+i*101, white, 137)
		currency := v.Currency
		if currency == "" {
			currency = "gold"
		}
		f.Draw(s, fmt.Sprintf("%d %s", g.Cost, currency), x+342, y+107+i*101, white, 87)
		v.moves[i].Draw(s)
	}
	v.close.Draw(s)
}
