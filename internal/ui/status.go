package ui

import (
	"fmt"

	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

const StatusWidth, StatusHeight = 277, 486

// Status uses the original stats art and controls (0x502df0). The delivery
// layout starts at the right edge; numeric allocation replaces the graph.
type Status struct {
	Position             WindowPosition
	frames               [2]*render.Texture
	resourceBars, expBar *render.Texture
	close, expand        Button
	apply, cancel        Button
	steps                [12]Button
	Expanded             bool
	Draft                proto.StatPoints
}

type StatusState struct {
	Class, Mode string
	Base        proto.StatPoints
	Points      uint16
	Details     proto.Details
	HP, MP, SP  uint32
	Stats       proto.AdditionalStats
	Known, Busy bool
}

// Display order STR/REC/INT/WIS/DEX/LUC differs from the allocation wire order.
var statOrder = [...]int{0, 2, 3, 4, 1, 5}

func StatusOrigin(w, h int, expanded bool) (int, int) {
	width := StatusWidth
	if expanded {
		width += 132 // Native expanded bounds, 0x502ab0.
	}
	return max(0, w-width), min(114, max(0, h-StatusHeight))
}

func NewStatus(t Textures) (*Status, error) {
	ld := &loader{t: t}
	v := &Status{frames: [2]*render.Texture{ld.get("stats_off.tga"), ld.get("stats_on.tga")}}
	v.resourceBars, v.expBar = ld.get("main_stats_bar.tga"), ld.get("main_bottom_bar.tga")
	v.close = nativeClose(ld.get("talk1_close_button.tga"), 250, 8)
	v.expand = Button{Tex: ld.get("stats_button.tga"), OffX: 0, OffY: 156, DrawW: 32, DrawH: 128, HitW: 27, HitH: 102}
	for i := range v.expand.UV {
		v.expand.UV[i] = UV{0, 0, 1, 1}
	}
	tex := ld.get("stats_on_select_button.tga")
	for i := range v.steps {
		b := &v.steps[i]
		*b = Button{Tex: tex, OffX: 85 + i%2*26, OffY: 128 + i/2*20, DrawW: 32, DrawH: 32, HitW: 22, HitH: 20}
		for state, col := range []int{0, 1, 2, 2, 3} {
			b.UV[state] = UV{float32(col) / 4, float32(i%2) / 2, float32(col+1) / 4, float32(i%2+1) / 2}
		}
	}
	tex = ld.get("stats_on_button.tga")
	for i, b := range []*Button{&v.apply, &v.cancel} {
		*b = Button{Tex: tex, OffX: 15 + i*65, OffY: 269, DrawW: 64, DrawH: 32, HitW: 46, HitH: 20}
		for state, col := range []int{0, 1, 2, 2, 3} {
			b.UV[state] = UV{float32(col) / 4, float32(i) / 2, float32(col+1) / 4, float32(i+1) / 2}
		}
	}
	return v, ld.err
}

func (v *Status) Clear() { v.Draft = proto.StatPoints{}; v.Expanded = false }

func (v *Status) controls(w, h int, state StatusState) {
	x, y := v.Origin(w, h)
	mainX := x
	if v.Expanded {
		mainX += 132
	}
	v.close.OriginX, v.close.OriginY = mainX, y
	v.expand.OriginX, v.expand.OriginY = mainX, y
	for i := range v.steps {
		b := &v.steps[i]
		index := statOrder[i/2]
		b.OriginX, b.OriginY = x, y
		b.Disabled = state.Busy
		if i%2 == 0 {
			b.Disabled = b.Disabled || v.Draft.Total() >= uint32(state.Points) || uint32(state.Base[index])+uint32(v.Draft[index]) >= 65535
		} else {
			b.Disabled = b.Disabled || v.Draft[index] == 0
		}
	}
	for _, b := range []*Button{&v.apply, &v.cancel} {
		b.OriginX, b.OriginY = x, y
		b.Disabled = state.Busy
	}
	v.apply.Disabled = v.apply.Disabled || v.Draft.Total() == 0 || v.Draft.Total() > uint32(state.Points)
}

// Update returns Apply and Close; Draft is never written into confirmed state.
func (v *Status) Update(in *platform.Input, w, h int, state StatusState) (bool, bool) {
	v.controls(w, h, state)
	if in.Pressed {
		for _, b := range v.buttons() {
			b.held = false
		}
	}
	if v.close.Update(in) {
		return false, true
	}
	if v.expand.Update(in) {
		v.Expanded = !v.Expanded
		return false, false
	}
	if !v.Expanded {
		return false, false
	}
	for i := range v.steps {
		if v.steps[i].Update(in) {
			index := statOrder[i/2]
			if i%2 == 0 {
				v.Draft[index]++
			} else {
				v.Draft[index]--
			}
			return false, false
		}
	}
	if v.cancel.Update(in) {
		v.Clear()
		return false, false
	}
	return v.apply.Update(in), false
}

func (v *Status) buttons() []*Button {
	b := []*Button{&v.close, &v.expand, &v.apply, &v.cancel}
	for i := range v.steps {
		b = append(b, &v.steps[i])
	}
	return b
}

func (v *Status) Draw(s *render.Sprites, f *Font, w, h int, state StatusState) {
	v.controls(w, h, state)
	x, y := v.Origin(w, h)
	variant, width, mainX := 0, StatusWidth, x
	if v.Expanded {
		variant, width, mainX = 1, width+132, x+132
	}
	s.Draw(v.frames[variant], float32(x), float32(y), float32(width), StatusHeight, 0, 0, float32(width)/512, StatusHeight/512.0, white)
	// Phoenix stats_hud.gd: EXP (73,132) 183x15; pools (73,265+20i) 68x15.
	xp := progress(state.Details.Exp, state.Details.PrevExp, state.Details.NextExp)
	s.Draw(v.expBar, float32(mainX+73), float32(y+132), 183*xp, 15, 0, 0, 428.0/512*xp, 6.0/16, white)
	for i, pool := range []Resource{{state.HP, state.Details.MaxHP}, {state.MP, state.Details.MaxMP}, {state.SP, state.Details.MaxSP}} {
		ratio := progress(pool.Current, 0, pool.Max)
		s.Draw(v.resourceBars, float32(mainX+73), float32(y+265+20*i), 68*ratio, 15, 0, float32(i)/4, ratio, float32(i)/4+12.0/64, white)
	}
	text := func(value string, dx, dy, width int) { f.Draw(s, value, mainX+dx, y+dy, white, width) }
	text(state.Class, 75, 80, 66)
	text(state.Mode, 190, 80, 68)
	text(fmt.Sprintf("%d / %d", state.Details.Exp, state.Details.NextExp), 78, 132, 180)
	for i, index := range statOrder {
		text(fmt.Sprint(state.Base[index]), 78+i/3*114, 174+i%3*20, 67)
	}
	text(fmt.Sprint(state.Points), 192, 234, 66)
	for i, pool := range []Resource{{state.HP, state.Details.MaxHP}, {state.MP, state.Details.MaxMP}, {state.SP, state.Details.MaxSP}} {
		text(fmt.Sprintf("%d/%d", pool.Current, pool.Max), 74, 266+i*20, 66)
	}
	derived := []string{"-", "-", "-", "-"}
	if state.Known {
		d := state.Stats
		derived = []string{fmt.Sprintf("%d-%d", d.MinAttack, d.MaxAttack), fmt.Sprintf("%d-%d", d.MinMagic, d.MaxMagic), fmt.Sprint(d.Defense), fmt.Sprint(d.Resistance)}
	}
	for i, value := range derived {
		text(value, 192, 269+i*20, 66)
	}
	for i, value := range []uint32{state.Details.Kills, state.Details.Deaths, state.Details.Wins, state.Details.Losses} {
		text(fmt.Sprint(value), 78+i%2*114, 405+i/2*20, 66)
	}
	v.close.Draw(s)
	v.expand.Draw(s)
	if v.Expanded {
		f.Draw(s, fmt.Sprint(uint32(state.Points)-min(uint32(state.Points), v.Draft.Total())), x+82, y+102, white, 58)
		for i, index := range statOrder {
			f.Draw(s, fmt.Sprint(v.Draft[index]), x+52, y+132+i*20, white, 28)
		}
		for i := range v.steps {
			v.steps[i].Draw(s)
		}
		v.apply.Draw(s)
		v.cancel.Draw(s)
	}
}

func (v *Status) Origin(w, h int) (int, int) {
	expanded := v != nil && v.Expanded
	x, y := StatusOrigin(w, h, expanded)
	width := StatusWidth
	if expanded {
		width += 132
	}
	if v == nil {
		return x, y
	}
	return v.Position.Origin(w, h, width, StatusHeight, x, y)
}
