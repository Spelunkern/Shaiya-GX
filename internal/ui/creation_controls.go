package ui

import "github.com/Spelunkern/Shaiya-GX/internal/render"

// Native constructor 0x467170; float coordinates at 0x708e88..0x708f08.
func CreationOrigin(w, h int) (int, int) { return w / 12, h/2 - 335 }

// CreationNavigationOrigin is the native 0x468980 anchor.
func CreationNavigationOrigin(w, h int) (int, int) { return w - w/14 - 256, h - h/14 - 100 }

func creationLabel(tex *render.Texture, x, y, row, rows int) Button {
	b := Button{Tex: tex, OffX: x, OffY: y, DrawW: 128, DrawH: 32, HitW: 68, HitH: 22, Toggle: true}
	for state, col := range []int{0, 1, 2, 3, 2} {
		b.UV[state] = UV{float32(col) / 4, float32(row) / float32(rows), float32(col+1) / 4, float32(row+1) / float32(rows)}
	}
	return b
}
func (c *Creation) controls(w, h int) {
	x, y := CreationOrigin(w, h)
	faction := c.Draft.Race / 2
	races := [2]uint8{0, 1}
	if faction == 1 {
		races = [2]uint8{3, 2}
	}
	for i := range c.races {
		row := []int{2, 3, 1, 0}[races[i]]
		skin := creationLabel(c.races[i].Tex, 70+i*78, 290, row, 4)
		c.races[i].UV = skin.UV
		c.races[i].Checked = c.Draft.Race == races[i]
	}
	for i := range c.sexes {
		c.sexes[i].Checked = c.Draft.Gender == uint8(i)
	}
	for i, job := range familyJobs[c.Draft.Race] {
		row := []int{6, 7, 9, 10, 11, 8}[job]
		if faction == 1 {
			row = []int{0, 1, 3, 2, 4, 5}[job]
		}
		skin := creationLabel(c.jobs[i].Tex, 22+i*75, 389, row, 16)
		c.jobs[i].UV = skin.UV
		c.jobs[i].Checked = c.Draft.Class == job
	}
	for i := range c.modes {
		c.modes[i].Checked = c.Draft.Mode == uint8(i)
	}
	for _, b := range c.buttons() {
		b.OriginX, b.OriginY, b.Disabled = x, y, c.Busy
	}
	for i := range c.modes {
		c.modes[i].Disabled = c.Busy || uint8(i) > c.MaxMode
	}
	c.OK.Disabled = c.Busy || len(c.cleanName()) < 2
}
func (c *Creation) buttons() []*Button {
	out := []*Button{c.OK, c.Cancel}
	for i := range c.races {
		out = append(out, &c.races[i])
	}
	for i := range c.sexes {
		out = append(out, &c.sexes[i])
	}
	for i := range c.jobs {
		out = append(out, &c.jobs[i])
	}
	for i := range c.arrows {
		out = append(out, &c.arrows[i])
	}
	for i := range c.modes {
		out = append(out, &c.modes[i])
	}
	return out
}
func (c *Creation) loadControls(ld *loader) {
	tex := ld.get("create_race_button.tga")
	for i := range c.races {
		c.races[i] = creationLabel(tex, 70+i*78, 290, 2+i, 4)
	}
	tex = ld.get("create_sex_button.tga")
	for i := range c.sexes {
		c.sexes[i] = creationLabel(tex, 70+i*78, 328, i, 2)
	}
	tex = ld.get("create_job_button.tga")
	for i := range c.jobs {
		c.jobs[i] = creationLabel(tex, 22+i*75, 389, 6+i, 16)
	}
	tex = ld.get("market_button.tga")
	for i := range c.arrows {
		side, row := i%2, i/2
		b := &c.arrows[i]
		*b = Button{Tex: tex, OffX: 97 + side*92, OffY: []int{440, 473, 507}[row], DrawW: 32, DrawH: 32, HitW: 23, HitH: 23}
		for state, v := range []int{0, 1, 2, 2, 3} {
			b.UV[state] = UV{float32(side) / 2, float32(v) / 4, float32(side+1) / 2, float32(v+1) / 4}
		}
	}
	tex = ld.get("create_mode_button.tga")
	for i := range c.modes {
		b := &c.modes[i]
		*b = Button{Tex: tex, OffX: 70 + i%2*78, OffY: 554 + i/2*29, DrawW: 128, DrawH: 32, HitW: 68, HitH: 22, Toggle: true}
		for state, row := range []int{0, 1, 2, 3, 4} {
			b.UV[state] = UV{float32(i) / 4, float32(row) / 8, float32(i+1) / 4, float32(row+1) / 8}
		}
	}
	for i, name := range []string{"af", "df", "pr", "sr", "am", "dm"} {
		c.capabilities[i] = ld.get("class_" + name + ".tga")
	}
}
