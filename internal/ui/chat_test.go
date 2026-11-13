package ui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/platform"
)

func chatTestFont() *Font {
	f := &Font{}
	for i := range f.glyphs {
		f.glyphs[i].adv = 6
	}
	return f
}

func TestOriginalChatModeButtonSelectsChannel(t *testing.T) {
	c, err := NewChatPanel(marketTextures{})
	if err != nil {
		t.Fatal(err)
	}
	c.Typing = true
	y := 768 - 82
	click := func(x, y int) {
		c.Update(&platform.Input{MouseX: x, MouseY: y, Pressed: true, MouseDown: true}, 768)
		c.Update(&platform.Input{MouseX: x, MouseY: y, Clicked: true}, 768)
	}
	click(20, y+8)
	if !c.ModeOpen {
		t.Fatal("mode button did not open channels")
	}
	click(20, y-154+1*22+8)
	if c.Channel != 3 || c.ModeOpen {
		t.Fatal("party selection failed", c.Channel, c.ModeOpen)
	}
}

func TestChatWrapIncludesUnbrokenMessages(t *testing.T) {
	f := chatTestFont()
	text := strings.Repeat("x", 121)
	lines := wrapLines(f, text, 60)
	if strings.Join(lines, "") != text || len(lines) != 13 {
		t.Fatal("long message lost characters", lines)
	}
	for _, line := range lines {
		if f.Width(line) > 60 {
			t.Fatal("line exceeds chat width", line)
		}
	}
	if got := wrapLines(f, "one two\n\nthree", 42); strings.Join(got, "|") != "one two||three" {
		t.Fatal("word/newline wrapping changed", got)
	}
}

func TestChatPanesScrollIndependentlyAndKeepNewMessages(t *testing.T) {
	c, f := &ChatPanel{Rows: [2]int{5, 5}}, chatTestFont()
	entries := []ChatEntry{{Text: "System", System: true}}
	for i := range 20 {
		entries = append(entries, ChatEntry{Text: fmt.Sprint(i)})
	}
	c.SetEntries(f, entries)
	if len(c.panes[0].lines) != 1 || len(c.panes[1].lines) != 20 {
		t.Fatal("system and player histories were mixed")
	}
	in := platform.Input{MouseX: 40, MouseY: ChatOrigin(734) + 130, Wheel: 120}
	if !c.Update(&in, 734) || c.panes[1].offset != 3 || c.panes[0].offset != 0 {
		t.Fatal("wheel did not scroll only the pointed pane")
	}
	entries = append(entries, ChatEntry{Text: "new"})
	c.SetEntries(f, entries)
	if c.panes[1].offset != 4 {
		t.Fatal("new message moved an older view")
	}
	c.SetEntries(f, entries)
	if c.panes[1].offset != 4 {
		t.Fatal("unchanged history moved the view")
	}
	in.Wheel, in.Pressed, in.MouseDown = 0, true, true
	in.MouseX, in.MouseY = 8, 734-228+20
	c.Update(&in, 734)
	if c.panes[1].offset != 13 {
		t.Fatal("scroll track did not reach oldest messages", c.panes[1].offset)
	}
	in.MouseY += 96
	c.Update(&in, 734)
	if c.panes[1].offset != 0 {
		t.Fatal("drag did not return to newest messages")
	}
	in.MouseDown, in.Pressed, in.Clicked = false, false, true
	in.MouseX = 500
	if !c.Update(&in, 734) {
		t.Fatal("scroll release escaped to world")
	}
	// Another panel may consume a release. A subsequent world press must not
	// continue that old scrollbar drag.
	in = platform.Input{MouseX: 8, MouseY: ChatOrigin(734) + 150, Pressed: true, MouseDown: true}
	c.Update(&in, 734)
	in.MouseX = 500
	if c.Update(&in, 734) || c.panes[1].dragging {
		t.Fatal("new press continued an interrupted scrollbar drag")
	}
}

func TestPhoenixChatRetainsSystemMarkupAndLiteralPlayerText(t *testing.T) {
	c, f := &ChatPanel{Rows: [2]int{5, 5}}, chatTestFont()
	entries := []ChatEntry{{Text: "damage", System: true, Style: 0}, {Text: "acquired {c3}item{/c}", System: true, Style: 3, Markup: true}, {Text: "literal {c3}message", Style: 9}}
	c.SetEntries(f, entries)
	if c.panes[0].lines[1] != "acquired item" || len(c.panes[0].runs[1]) != 2 || c.panes[1].lines[0] != "literal {c3}message" {
		t.Fatal("system markup or literal chat changed", c.panes)
	}
}

func TestPhoenixChatGeometryAndZeroRows(t *testing.T) {
	c := &ChatPanel{Rows: [2]int{5, 5}}
	for i, want := range [][3]int{{424, 112, 6}, {540, 144, 8}} {
		y, height, capacity := c.paneGeometry(768, i)
		if [3]int{y, height, capacity} != want {
			t.Fatal(i, y, height, capacity)
		}
	}
	c.Rows = [2]int{0, 0}
	for i, want := range [][3]int{{584, 32, 1}, {620, 64, 3}} {
		y, height, capacity := c.paneGeometry(768, i)
		if [3]int{y, height, capacity} != want {
			t.Fatal("collapsed history", i, y, height, capacity)
		}
	}
}

func TestNativeChatScrollAnchorSurvivesHistoryLimit(t *testing.T) {
	c, f := &ChatPanel{Rows: [2]int{5, 5}}, chatTestFont()
	var entries []ChatEntry
	for i := range 100 {
		entries = append(entries, ChatEntry{Text: fmt.Sprint(i), Style: 9})
	}
	c.SetEntries(f, entries)
	c.panes[1].offset = 10
	entries = append(entries[1:], ChatEntry{Text: "100", Style: 9})
	c.SetEntries(f, entries)
	if c.panes[1].offset != 11 {
		t.Fatal("new message moved historical view", c.panes[1].offset)
	}
}
