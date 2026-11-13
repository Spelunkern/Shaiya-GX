package ui

import (
	"github.com/Spelunkern/Shaiya-GX/internal/textwire"
	"strconv"
	"strings"
	"unicode/utf8"
)

type chatGlyph struct {
	char  rune
	color [4]float32
}
type chatRun struct {
	text  string
	color [4]float32
	x     int
}

// Only trusted native notices interpret color markup. Player chat is literal.
func wrapChatMarkup(f *Font, text string, base [4]float32, markup bool) ([][]chatRun, []string) {
	var glyphs []chatGlyph
	color := base
	for len(text) > 0 {
		if markup {
			if strings.HasPrefix(text, "{/c}") || strings.HasPrefix(text, "{/r}") {
				color = base
				text = text[4:]
				continue
			}
			if strings.HasPrefix(text, "{r}") {
				color = RGBColor(0xff8700)
				text = text[3:]
				continue
			}
			if strings.HasPrefix(text, "{c") {
				if end := strings.IndexByte(text, '}'); end > 2 {
					if n, err := strconv.Atoi(text[2:end]); err == nil {
						colors := map[int]uint32{0: 0xff00ff, 1: 0x80ffff, 2: 0x00ff00, 3: 0xffff00, 4: 0xff0000, 5: 0xa19e9c, 50: 0xffcc00, 51: 0xccff33}
						rgb, ok := colors[n]
						if !ok {
							rgb = 0xffffff
						}
						color = RGBColor(rgb)
						text = text[end+1:]
						continue
					}
				}
			}
		}
		r, size := utf8.DecodeRuneInString(text)
		glyphs = append(glyphs, chatGlyph{r, color})
		text = text[size:]
	}
	var rows [][]chatRun
	var lines []string
	for len(glyphs) > 0 {
		cut, space := 0, -1
		var remaining strings.Builder
		for _, g := range glyphs {
			if g.char == '\n' {
				break
			}
			remaining.WriteRune(g.char)
		}
		candidate := ""
		for _, cluster := range textwire.Graphemes(remaining.String()) {
			if cut > 0 && f.Width(candidate+cluster) > 316 {
				break
			}
			if cluster == " " {
				space = cut
			}
			candidate += cluster
			cut += len([]rune(cluster))
		}
		if cut < len(glyphs) && glyphs[cut].char != '\n' && space > 0 {
			cut = space
		}
		row := glyphs[:cut]
		var runs []chatRun
		var plain strings.Builder
		x := 0
		for start := 0; start < len(row); {
			end := start + 1
			for end < len(row) && row[start].color == row[end].color {
				end++
			}
			var b strings.Builder
			for _, g := range row[start:end] {
				b.WriteRune(g.char)
			}
			value := b.String()
			runs = append(runs, chatRun{value, row[start].color, x})
			plain.WriteString(value)
			x += f.Width(value)
			start = end
		}
		rows = append(rows, runs)
		lines = append(lines, plain.String())
		glyphs = glyphs[cut:]
		if len(glyphs) > 0 && glyphs[0].char == '\n' {
			glyphs = glyphs[1:]
		} else {
			for len(glyphs) > 0 && glyphs[0].char == ' ' {
				glyphs = glyphs[1:]
			}
		}
	}
	return rows, lines
}
