package app

import (
	"strings"

	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

type tooltipSocket struct{ definition *sdata.Item }
type tooltipRun struct {
	text   string
	color  [4]float32
	x, y   int
	shadow bool
}
type tooltipLayout struct {
	runs            []tooltipRun
	height, socketY int
}

func wrapTooltip(f *ui.Font, text string, color [4]float32) [][]tooltipGlyph {
	var lines [][]tooltipGlyph
	for _, paragraph := range strings.Split(text, "\n") {
		glyphs := tooltipGlyphs(paragraph, color)
		if len(glyphs) == 0 {
			lines = append(lines, nil)
			continue
		}
		for len(glyphs) > 0 {
			pixels, cut, space := 0, 0, -1
			for cut < len(glyphs) {
				advance := f.Width(string(glyphs[cut].char))
				if cut > 0 && pixels+advance > 185 {
					break
				}
				if glyphs[cut].char == ' ' {
					space = cut
				}
				pixels += advance
				cut++
			}
			if cut < len(glyphs) && space > 0 {
				cut = space
			}
			lines = append(lines, glyphs[:cut])
			glyphs = glyphs[cut:]
			for len(glyphs) > 0 && glyphs[0].char == ' ' {
				glyphs = glyphs[1:]
			}
		}
	}
	return lines
}

// The two Phoenix classes have distinct block spacing, not a single list of
// evenly spaced rows. Height is measured with the actual reference font.
func layoutTooltip(f *ui.Font, rows []itemDetailLine, item bool, description string, sockets int) tooltipLayout {
	layout := tooltipLayout{}
	block := func(text string, color [4]float32, y int, shadow bool) (height, lines int) {
		wrapped := wrapTooltip(f, text, color)
		for i, line := range wrapped {
			x := 10
			for start := 0; start < len(line); {
				end := start + 1
				for end < len(line) && line[end].color == line[start].color {
					end++
				}
				var text strings.Builder
				for _, g := range line[start:end] {
					text.WriteRune(g.char)
				}
				value := text.String()
				layout.runs = append(layout.runs, tooltipRun{value, line[start].color, x, y + i*15, shadow})
				x += f.Width(value)
				start = end
			}
		}
		return f.Height + max(0, len(wrapped)-1)*15, len(wrapped)
	}
	if len(rows) == 0 {
		return layout
	}
	height, _ := block(rows[0].text, rows[0].color, 10, false)
	y := 10 + height + 5
	if item {
		y = max(40, y)
		for _, row := range rows[1:] {
			_, count := block(row.text, row.color, y, strings.Contains(row.text, "{r}"))
			y += max(15, count*15)
		}
	} else if len(rows) > 1 {
		// Phoenix's stats Label is one multiline block, with a single 5px gap.
		top := y
		count := 0
		for _, row := range rows[1:] {
			_, n := block(row.text, row.color, top+count*15, false)
			count += n
		}
		y = top + f.Height + max(0, count-1)*15 + 5
	}
	if description != "" {
		if item {
			y += 5
		}
		height, _ := block(description, white, y, false)
		y += height
	}
	layout.socketY = y + 3
	if item && sockets > 0 {
		y += 24
	}
	layout.height = y + 10
	return layout
}
