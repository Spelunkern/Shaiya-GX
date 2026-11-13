package ui

import (
	"github.com/Spelunkern/Shaiya-GX/internal/textwire"
	"slices"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

type ChatEntry struct {
	Text   string
	System bool
	Color  [4]float32
	Style  uint8
	Markup bool
}

type chatPane struct {
	lines    []string
	runs     [][]chatRun
	offset   int
	arrows   [2]Button
	dragging bool
	active   time.Time // last new line or interaction; drives auto-hide
}

// Idle chat hides: a pane shows while typing, or for chatHold after a new
// line or interaction, then fades out over chatFade. The input row shows only
// while typing.
const (
	chatHold = 10 * time.Second
	chatFade = time.Second
)

var chatNow = time.Now

func (c *ChatPanel) paneAlpha(i int, now time.Time) float32 {
	if c.Typing || c.ModeOpen || c.resizing || c.panes[i].dragging {
		return 1
	}
	idle := now.Sub(c.panes[i].active)
	if c.panes[i].active.IsZero() || idle >= chatHold+chatFade {
		return 0
	}
	if idle <= chatHold {
		return 1
	}
	return 1 - float32(idle-chatHold)/float32(chatFade)
}

func (c *ChatPanel) inputShown() bool { return c.Typing || c.ModeOpen }

// ShowHistory reveals both history panes and restarts their hide timer.
func (c *ChatPanel) ShowHistory() {
	now := chatNow()
	for i := range c.panes {
		c.panes[i].active = now
	}
}

// ChatPanel follows Phoenix chat_hud.gd and ep4_chat.gd: flat persistent
// histories, 16px rows, a 64px selector and a 286px nine-patch input.
type ChatPanel struct {
	FocusRequested                   bool
	Resized, ChannelChanged          bool
	scroll, input                    *render.Texture
	ChannelNames                     [6]string
	Recipient                        string
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
	c := &ChatPanel{scroll: ld.get("scrollbar.png"), ChannelNames: chatChannelNames, input: ld.get("chat.png"), modeHeld: -1, Rows: [2]int{5, 5}}
	c.modeButton = Button{Tex: c.input, DrawW: 64, DrawH: 22, HitW: 64, HitH: 22}
	for state, row := range []int{0, 32, 64, 64, 96} {
		c.modeButton.UV[state] = UV{0, float32(row) / 256, 64.0 / 512, float32(row+22) / 256}
	}
	return c, ld.err
}

func ChatOrigin(h int) int { return max(0, h-348) }

func (c *ChatPanel) paneGeometry(h, index int) (y, height, capacity int) {
	rows := c.Rows[index]
	rows = max(0, min(20, rows))
	lowerRows := c.Rows[1]
	lowerRows = max(0, min(20, lowerRows))
	height = (rows + 2 + 2*index) * 16
	y = h - 84 - (lowerRows+4)*16
	if index == 0 {
		y -= 8 + height
	}
	return max(0, y), height, max(1, height/16)
}

func (c *ChatPanel) SetEntries(f *Font, entries []ChatEntry) {
	if slices.Equal(c.entries, entries) {
		return
	}
	overlap := min(len(c.entries), len(entries))
	for overlap > 0 && !slices.Equal(c.entries[len(c.entries)-overlap:], entries[:overlap]) {
		overlap--
	}
	added := [2]int{}
	c.entries = append(c.entries[:0], entries...)
	for i := range c.panes {
		c.panes[i].lines = c.panes[i].lines[:0]
		clear(c.panes[i].runs)
		c.panes[i].runs = c.panes[i].runs[:0]
	}
	for entryIndex, entry := range entries {
		index := 1
		if entry.System {
			index = 0
		}
		pane := &c.panes[1]
		if entry.System {
			pane = &c.panes[0]
		}
		runs, rows := wrapChatMarkup(f, entry.Text, entry.Color, entry.Markup)
		if entryIndex >= overlap {
			added[index] += len(rows)
		}
		pane.runs = append(pane.runs, runs...)
		pane.lines = append(pane.lines, rows...)
	}
	if added[0]+added[1] > 0 {
		c.ShowHistory() // any new line brings up both panes together
	}
	for i := range c.panes {
		p := &c.panes[i]
		if p.offset > 0 {
			p.offset += added[i]
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

// Contains keeps world picking behind the visible chat panels from handling
// the same pointer gesture. Geometry follows the current resized histories.
func (c *ChatPanel) Contains(mx, my, h int) bool {
	if mx < 0 || mx >= 350 {
		return false
	}
	if c.inputShown() && my >= h-82 && my < h-60 {
		return true
	}
	now := chatNow()
	for i := range c.panes {
		if c.paneAlpha(i, now) == 0 {
			continue
		}
		y, height, _ := c.paneGeometry(h, i)
		if my >= y && my < y+height {
			return true
		}
	}
	return false
}

func (c *ChatPanel) Update(in *platform.Input, h int) bool {
	c.Resized = false
	c.ChannelChanged = false
	c.FocusRequested = false
	if c.resizing {
		c.Rows[c.resizePane] = min(20, max(0, c.resizeStart+(c.resizeY-in.MouseY)/16))
		if in.Clicked || !in.MouseDown {
			c.resizing = false
			c.Resized = true
		}
		return true
	}
	now := chatNow()
	if in.Pressed && in.MouseX >= 25 && in.MouseX < 350 {
		for pane := range c.panes {
			if c.paneAlpha(pane, now) == 0 {
				continue
			}
			y, _, _ := c.paneGeometry(h, pane)
			if in.MouseY >= y-4 && in.MouseY < y+5 {
				c.resizing = true
				c.resizePane, c.resizeY, c.resizeStart = pane, in.MouseY, c.Rows[pane]
				return true
			}
		}
	}
	c.controls(h)
	inputY := h - 82
	c.modeButton.OriginY = inputY
	if c.inputShown() && c.modeButton.Update(in) {
		c.ModeOpen = !c.ModeOpen
	}
	if c.ModeOpen {
		row := -1
		if in.MouseX >= 0 && in.MouseX < 120 && in.MouseY >= inputY-len(chatChannelOrder)*22 && in.MouseY < inputY {
			row = (in.MouseY - (inputY - len(chatChannelOrder)*22)) / 22
		}
		if in.Pressed {
			c.modeHeld = row
		}
		if in.Clicked && row >= 0 && c.modeHeld == row {
			c.Channel = chatChannelOrder[row]
			c.ChannelChanged = true
			c.FocusRequested = true
			c.Recipient = ""
			c.ModeOpen = false
		}
		if in.Pressed && row < 0 && !c.modeButton.hit(in) {
			c.ModeOpen = false
		}
		return true
	}
	if c.inputShown() && in.MouseX >= 0 && in.MouseX < 350 && in.MouseY >= inputY && in.MouseY < inputY+22 {
		c.FocusRequested = in.Pressed && in.MouseX >= 64
		return true
	}

	captured := false
	for i := range c.panes {
		p := &c.panes[i]
		y, height, capacity := c.paneGeometry(h, i)
		limit := max(0, len(p.lines)-capacity)
		if c.paneAlpha(i, now) == 0 {
			p.dragging = false
			p.offset = max(0, min(p.offset, limit))
			continue
		}
		over := in.MouseX >= 0 && in.MouseX < 350 && in.MouseY >= y && in.MouseY < y+height
		if over {
			p.active = now // keep a visible pane up while the pointer is on it
		}
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
	now := chatNow()
	for i := range c.panes {
		p := &c.panes[i]
		if c.inputShown() {
			p.active = now // closing the input starts the hold timer
		}
		a := c.paneAlpha(i, now)
		if a == 0 {
			continue
		}
		y, height, capacity := c.paneGeometry(h, i)
		s.Draw(blank, 0, float32(y), 350, float32(height), 0, 0, 1, 1, [4]float32{29.0 / 255, 31.0 / 255, 36.0 / 255, 154.0 / 255 * a})
		end := max(0, len(p.lines)-p.offset)
		start := max(0, end-capacity)
		for row := start; row < end; row++ {
			yText := y + 5 + (capacity-(end-start)+row-start)*16
			for _, run := range p.runs[row] {
				tint := run.color
				if tint[3] == 0 {
					tint = white
				}
				tint[3] *= a
				f.Draw(s, run.text, 26+run.x, yText+1, [4]float32{0, 0, 0, tint[3]}, 316-run.x)
				f.Draw(s, run.text, 25+run.x, yText, tint, 316-run.x)
			}
		}
		// Phoenix's bordered scrollbar rail is always visible, alpha 0.82.
		s.Draw(blank, 4, float32(y+4), 16, float32(height-8), 0, 0, 1, 1, [4]float32{.31, .27, .19, .8 * .82 * a})
		s.Draw(blank, 5, float32(y+5), 14, float32(height-10), 0, 0, 1, 1, [4]float32{.037, .035, .031, .7 * .82 * a})
		if a == 1 {
			for arrow := range p.arrows {
				p.arrows[arrow].Draw(s)
			}
		}
	}
	if !c.inputShown() {
		return
	}
	y := h - 82
	c.modeButton.OriginY = y
	c.modeButton.Draw(s)
	label := c.ChannelNames[min(int(c.Channel), 5)]
	scale := float32(11.0 / 12)
	if c.Channel == 5 {
		scale = 10.0 / 12
	}
	f.DrawScaled(s, label, 6, float32(y+4), scale, [4]float32{0, 0, 0, 1}, int(44/scale))
	f.DrawScaled(s, label, 5, float32(y+3), scale, white, int(44/scale))
	// Phoenix CleanFrame: source 272x22, 4px side patches, 284px target.
	s.Draw(c.input, 64, float32(y), 4, 22, 0, .5, 4.0/512, 150.0/256, white)
	s.Draw(c.input, 68, float32(y), 276, 22, 4.0/512, .5, 268.0/512, 150.0/256, white)
	s.Draw(c.input, 344, float32(y), 4, 22, 268.0/512, .5, 272.0/512, 150.0/256, white)
	s.Draw(c.input, 348, float32(y), 2, 22, 290.0/512, .5, 292.0/512, 150.0/256, white)
	shown := text
	if typing {
		shown += "_"
	}
	for len(shown) > 1 && f.Width(shown) > 274 {
		shown = textwire.DropFirst(shown)
	}
	f.Draw(s, shown, 70, y+2, white, 274)
}

var chatChannelNames = [6]string{"Normal", "Trade", "Guild", "Party", "Area", "Whisper"}
var chatChannelOrder = [6]uint8{0, 3, 2, 5, 4, 1}

func (c *ChatPanel) capacity(index int) int { _, _, n := c.paneGeometry(768, index); return n }

// DrawOverlay keeps the Phoenix channel picker above movable HUD bars.
func (c *ChatPanel) DrawOverlay(s *render.Sprites, f *Font, blank *render.Texture, h int) {
	if !c.ModeOpen {
		return
	}
	top := h - 82 - len(chatChannelOrder)*22
	s.Draw(blank, 0, float32(top), 120, float32(len(chatChannelOrder)*22), 0, 0, 1, 1, [4]float32{.114, .122, .141, .97})
	for i, channel := range chatChannelOrder {
		tint := white
		if channel == c.Channel {
			tint = RGBColor(0xffff00)
		}
		f.Draw(s, c.ChannelNames[channel], 8, top+i*22+3, tint, 104)
	}
}
