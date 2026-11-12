package ui

import (
	"slices"

	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

type ChatEntry struct {
	Text   string
	System bool
	Color  [4]float32
}

type chatPane struct {
	lines    []string
	colors   [][4]float32
	offset   int
	arrows   [2]Button
	dragging bool
}

// ChatPanel uses 0x470120/0x4707d0, TipBox (0x50ff10) and the original
// scroll/input atlases. Seven rows per pane match the observed EP5 layout.
type ChatPanel struct {
	Resized                          bool
	scroll, input                    *render.Texture
	entries                          []ChatEntry
	panes                            [2]chatPane
	modeButton                       Button
	Typing, ModeOpen                 bool
	Rows                             [2]int
	resizePane, resizeY, resizeStart int
	resizing                         bool
	Channel                          uint8
	modeHeld                         int
}

func NewChatPanel(t Textures) (*ChatPanel, error) {
	ld := &loader{t: t}
	c := &ChatPanel{scroll: ld.get("scrollbar.tga"), input: ld.get("chat.tga"), modeHeld: -1, Rows: [2]int{5, 5}}
	c.modeButton = Button{Tex: c.input, DrawW: 64, DrawH: 22, HitW: 64, HitH: 22}
	for state, row := range []int{0, 32, 64, 64, 96} {
		c.modeButton.UV[state] = UV{0, float32(row) / 256, 64.0 / 512, float32(row+22) / 256}
	}
	return c, ld.err
}

func ChatOrigin(h int) int { return max(0, h-344) }

func (c *ChatPanel) paneGeometry(h, index int) (y, height, capacity int) {
	rows := c.Rows[index]
	if rows < 1 {
		rows = 5
	}
	lowerRows := c.Rows[1]
	if lowerRows < 1 {
		lowerRows = 5
	}
	height = (rows + 2 + 2*index) * 16
	y = h - 84 - (lowerRows+4)*16
	if index == 0 {
		y -= 4 + height
	}
	return max(0, y), height, (height - 8) / 16
}

func (c *ChatPanel) SetEntries(f *Font, entries []ChatEntry) {
	if slices.Equal(c.entries, entries) {
		return
	}
	c.entries = append(c.entries[:0], entries...)
	previous := [2]int{len(c.panes[0].lines), len(c.panes[1].lines)}
	for i := range c.panes {
		c.panes[i].lines = c.panes[i].lines[:0]
		c.panes[i].colors = c.panes[i].colors[:0]
	}
	for _, entry := range entries {
		pane := &c.panes[1]
		if entry.System {
			pane = &c.panes[0]
		}
		rows := wrapLines(f, entry.Text, 316)
		pane.lines = append(pane.lines, rows...)
		for range rows {
			pane.colors = append(pane.colors, entry.Color)
		}
	}
	for i := range c.panes {
		p := &c.panes[i]
		if p.offset > 0 {
			p.offset += max(0, len(p.lines)-previous[i])
		}
		p.offset = min(p.offset, max(0, len(p.lines)-c.capacity(i)))
	}
}

func (c *ChatPanel) controls(h int) {
	for i := range c.panes {
		p := &c.panes[i]
		y, height, capacity := c.paneGeometry(h, i)
		limit := max(0, len(p.lines)-capacity)
		for arrow := range p.arrows {
			b := &p.arrows[arrow]
			b.Tex, b.OriginX, b.OriginY = c.scroll, 4, y+4+arrow*(height-24)
			b.DrawW, b.DrawH, b.HitW, b.HitH = 16, 16, 16, 16
			b.Disabled = limit == 0
			for state, col := range []int{0, 1, 1, 1, 2} {
				b.UV[state] = UV{float32(col) / 8, float32(arrow) / 2, float32(col+1) / 8, float32(arrow)/2 + 16.0/64}
			}
		}
	}
}

func (c *ChatPanel) Update(in *platform.Input, h int) bool {
	c.Resized = false
	if c.resizing {
		c.Rows[c.resizePane] = min(20, max(1, c.resizeStart+(c.resizeY-in.MouseY)/16))
		if in.Clicked || !in.MouseDown {
			c.resizing = false
			c.Resized = true
		}
		return true
	}
	if in.Pressed && in.MouseX >= 25 && in.MouseX < 350 {
		for pane := range c.panes {
			y, _, _ := c.paneGeometry(h, pane)
			if in.MouseY >= y-4 && in.MouseY < y+2 {
				if c.Rows[pane] < 1 {
					c.Rows[pane] = 5
				}
				c.resizing = true
				c.resizePane, c.resizeY, c.resizeStart = pane, in.MouseY, c.Rows[pane]
				return true
			}
		}
	}
	c.controls(h)
	inputY := h - 82
	if c.Typing {
		c.modeButton.OriginY = inputY
		if c.modeButton.Update(in) {
			c.ModeOpen = !c.ModeOpen
		}
		if c.ModeOpen {
			row := -1
			if in.MouseX >= 0 && in.MouseX < 100 && in.MouseY >= inputY-5*22 && in.MouseY < inputY {
				row = (in.MouseY - (inputY - 5*22)) / 22
			}
			if in.Pressed {
				c.modeHeld = row
			}
			if in.Clicked && row >= 0 && c.modeHeld == row {
				c.Channel = uint8(row)
				c.ModeOpen = false
			}
			if in.Pressed && row < 0 && !c.modeButton.hit(in) {
				c.ModeOpen = false
			}
			return true
		}
		if in.MouseX >= 0 && in.MouseX < 350 && in.MouseY >= inputY && in.MouseY < inputY+22 {
			return true
		}
	} else {
		c.ModeOpen = false
	}
	captured := false
	for i := range c.panes {
		p := &c.panes[i]
		y, height, capacity := c.paneGeometry(h, i)
		limit := max(0, len(p.lines)-capacity)
		over := in.MouseX >= 0 && in.MouseX < 350 && in.MouseY >= y && in.MouseY < y+height
		if in.Pressed {
			p.dragging = false
		}
		if over {
			if in.Wheel > 0 {
				p.offset += 3
			} else if in.Wheel < 0 {
				p.offset -= 3
			}
			captured = captured || in.Pressed || in.Clicked || in.MouseDown || in.RightPressed || in.RightDown || in.Wheel != 0
		}
		for arrow := range p.arrows {
			if in.Pressed {
				p.arrows[arrow].held = false
			}
			if p.arrows[arrow].Update(in) {
				p.offset += 1 - 2*arrow
			}
		}
		if in.Pressed && in.MouseX >= 4 && in.MouseX < 17 && in.MouseY >= y+20 && in.MouseY < y+height-20 {
			p.dragging = true
		}
		if p.dragging && in.MouseDown {
			track := max(1, height-48)
			p.offset = limit - limit*max(0, min(track, in.MouseY-y-20))/track
			captured = true
		}
		if p.dragging && (in.Clicked || !in.MouseDown) {
			p.dragging = false
			captured = true
		}
		p.offset = max(0, min(p.offset, limit))
	}
	return captured
}

func (c *ChatPanel) Draw(s *render.Sprites, f *Font, blank *render.Texture, h int, typing bool, text string) {
	c.Typing = typing
	c.controls(h)
	for i := range c.panes {
		p := &c.panes[i]
		y, height, capacity := c.paneGeometry(h, i)
		s.Draw(blank, 0, float32(y), 350, float32(height), 0, 0, 1, 1, [4]float32{.114, .122, .141, .54})
		end := max(0, len(p.lines)-p.offset)
		start := max(0, end-capacity)
		for row, line := range p.lines[start:end] {
			tint := p.colors[start+row]
			if tint[3] == 0 {
				tint = white
			}
			f.Draw(s, line, 25, y+height-8-capacity*16+2+(capacity-(end-start)+row)*16, tint, 316)
		}
		for arrow := range p.arrows {
			p.arrows[arrow].Draw(s)
		}
		s.Draw(blank, 9, float32(y+20), 3, float32(height-40), 0, 0, 1, 1, [4]float32{.3, .3, .3, .6})
		track := max(1, height-48)
		thumb := track
		if limit := len(p.lines) - capacity; limit > 0 {
			thumb -= track * p.offset / limit
		}
		s.Draw(c.scroll, 4, float32(y+20+thumb), 13, 8, .5, 0, .5+13.0/128, 8.0/64, white)
	}
	if typing {
		y := h - 82
		// Phoenix ep4_chat.gd: original 64x22 mode button and 286x22 input.
		c.modeButton.OriginY = y
		c.modeButton.Draw(s)
		f.Draw(s, chatChannelNames[min(int(c.Channel), 4)], 5, y+4, white, 43)
		s.Draw(c.input, 64, float32(y), 4, 22, 0, .5, 4.0/512, 150.0/256, white)
		s.Draw(c.input, 68, float32(y), 280, 22, 4.0/512, .5, 268.0/512, 150.0/256, white)
		s.Draw(c.input, 348, float32(y), 2, 22, 290.0/512, .5, 292.0/512, 150.0/256, white)
		shown := []rune(text + "_")
		for len(shown) > 1 && f.Width(string(shown)) > 280 {
			shown = shown[1:]
		}
		f.Draw(s, string(shown), 70, y+4, white, 274)
		if c.ModeOpen {
			s.Draw(blank, 0, float32(y-110), 100, 110, 0, 0, 1, 1, [4]float32{29.0 / 255, 31.0 / 255, 36.0 / 255, .97})
			for i, label := range chatChannelNames {
				tint := white
				if i == int(c.Channel) {
					tint = [4]float32{1, 1, 0, 1}
				}
				f.Draw(s, label, 8, y-110+i*22+4, tint, 84)
			}
		}
	}
}

var chatChannelNames = [5]string{"Normal", "Trade", "Guild", "Party", "Shout"}

func (c *ChatPanel) capacity(index int) int { _, _, n := c.paneGeometry(768, index); return n }
