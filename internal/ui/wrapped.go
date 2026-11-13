package ui

import (
	"github.com/Spelunkern/Shaiya-GX/internal/textwire"
	"strings"
)

func wrapLines(f *Font, text string, width int) []string {
	if width <= 0 {
		return []string{text}
	}
	var lines []string
	for _, paragraph := range strings.Split(text, "\n") {
		line := ""
		for _, word := range strings.Fields(paragraph) {
			if line != "" && f.Width(line+" "+word) > width {
				lines = append(lines, line)
				line = ""
			}
			if line != "" {
				line += " "
			}
			for f.Width(word) > width {
				clusters := textwire.Graphemes(word)
				cut := 1
				for cut < len(clusters) && f.Width(strings.Join(clusters[:cut+1], "")) <= width {
					cut++
				}
				lines = append(lines, line+strings.Join(clusters[:cut], ""))
				line, word = "", strings.Join(clusters[cut:], "")
			}
			line += word
		}
		lines = append(lines, line)
	}
	return lines
}
