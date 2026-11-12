package ui

import (
	"fmt"
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

const GatekeeperWidth, GatekeeperHeight = 512, 366

type Gatekeeper struct {
	Position WindowPosition
	frame    *render.Texture
	close    Button
	moves    [3]Button
}

func NewGatekeeper(t Textures) (*Gatekeeper, error) {
	ld := &loader{t: t}
	v := &Gatekeeper{frame: ld.get("gatekeeper.tga")}
	v.close = nativeClose(ld.get("talk1_close_button.tga"), 483, 3)
	tex := ld.get("gatekeeper_button.tga")
	for i := range v.moves {
		v.moves[i] = Button{Tex: tex, OffX: 434, OffY: 80 + i*101, DrawW: 64, DrawH: 64, HitW: 58, HitH: 40,
			UV: [5]UV{{0, 0, .25, 1}, {.25, 0, .5, 1}, {.5, 0, .75, 1}, {.5, 0, .75, 1}, {.75, 0, 1, 1}}}
	}
	return v, ld.err
}
func (v *Gatekeeper) Origin(w, h int) (int, int) {
	return v.Position.Origin(w, h, GatekeeperWidth, GatekeeperHeight, min(32, max(0, w-GatekeeperWidth)), min(82, max(0, h-GatekeeperHeight)))
}
func gatePresent(g sdata.NPCGate) bool { return g.MapID != 0 || g.X != 0 || g.Z != 0 }
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
	f.Draw(s, name, x+76, y+22, white, 150)
	for i, line := range wrapLines(f, welcome, 205) {
		if i == 8 {
			break
		}
		f.Draw(s, line, x+25, y+100+i*15, [4]float32{.8, .8, .8, 1}, 205)
	}
	for i, g := range gates {
		if !gatePresent(g) {
			continue
		}
		label := g.Name
		if label == "" {
			label = fmt.Sprintf("Map %d", g.MapID)
		}
		f.Draw(s, label, x+292, y+80+i*101, white, 137)
		f.Draw(s, fmt.Sprintf("%d gold", g.Cost), x+349, y+104+i*101, white, 80)
		v.moves[i].Draw(s)
	}
	v.close.Draw(s)
}
