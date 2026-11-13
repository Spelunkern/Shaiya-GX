// Package gm defines GX's small, shared GM command vocabulary.
package gm

import (
	"strconv"
	"strings"
)

const TurboMultiplier = 12

// Parse recognizes commands independently of chat channel or account privileges.
// Authority is checked exclusively by the world server.
func Parse(text string) (string, []string, bool) {
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return "", nil, false
	}
	command := strings.ToLower(fields[0])
	switch command {
	case "/char", "/attack", "/iclear", "/turbo", "/getitem", "/mmake", "/nmake", "/cmove", "/goto":
		return command, fields[1:], true
	}
	return "", nil, false
}

// GotoCommand formats the same-map teleport sent by a GM World Map click.
// y is the client's terrain height at x/z, which the server does not know.
func GotoCommand(x, z, y float32) string {
	return GotoXZCommand(x, z) + " " + coord(y)
}

// GotoXZCommand keeps the character's current height (maps without terrain).
func GotoXZCommand(x, z float32) string { return "/goto " + coord(x) + " " + coord(z) }

func coord(f float32) string { return strconv.FormatFloat(float64(f), 'f', 2, 32) }
