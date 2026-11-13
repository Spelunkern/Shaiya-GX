package ui

import (
	"strings"
	"testing"
)

func TestLiberationSansUnicodeShaping(t *testing.T) {
	u, err := newUnicodeFont(defaultFontData, 13, 16, 12)
	if err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{"Español áéíóú ñ", "Ελληνικά Русский", "שלום עולם", "e\u0301"} {
		l := u.layout(text)
		if l.width <= 0 {
			t.Fatalf("zero width: %q", text)
		}
		for _, run := range l.runs {
			for _, g := range run.Glyphs {
				if g.GlyphID == 0 && g.Advance > 0 {
					t.Fatalf("missing Liberation Sans glyph: %q", text)
				}
			}
		}
	}
}

func TestUnicodeWrappingPreservesGraphemes(t *testing.T) {
	u, err := newUnicodeFont(defaultFontData, 13, 16, 12)
	if err != nil {
		t.Fatal(err)
	}
	f := &Font{unicode: u}
	cluster := "👨‍👩‍👧‍👦"
	text := strings.Repeat(cluster, 80)
	_, lines := wrapChatMarkup(f, text, white, false)
	if strings.Join(lines, "") != text {
		t.Fatal("chat wrapping lost text")
	}
	for _, line := range lines {
		if f.Width(line) > 316 || strings.ReplaceAll(line, cluster, "") != "" {
			t.Fatal("chat split emoji", line)
		}
	}
	lines = wrapLines(f, text, 40)
	if strings.Join(lines, "") != text {
		t.Fatal("tooltip wrapping lost text")
	}
	for _, line := range lines {
		if strings.ReplaceAll(line, cluster, "") != "" {
			t.Fatal("tooltip split emoji", line)
		}
	}
}
