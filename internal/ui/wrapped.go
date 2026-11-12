package ui

import "strings"

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
				runes := []rune(word)
				cut := 1
				for cut < len(runes) && f.Width(string(runes[:cut+1])) <= width {
					cut++
				}
				lines = append(lines, line+string(runes[:cut]))
				line, word = "", string(runes[cut:])
			}
			line += word
		}
		lines = append(lines, line)
	}
	return lines
}
