package ui

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExternalFontLazyCoverageAndEmojiClusters(t *testing.T) {
	dir := os.Getenv("SHAIYA_DATA")
	if dir == "" {
		t.Skip("external resources")
	}
	src, err := LoadFontSource(func(name string) ([]byte, error) { return os.ReadFile(filepath.Join(dir, name)) })
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range src.entries {
		if entry.attempted {
			t.Fatal("fallback loaded before it was needed")
		}
	}
	u, err := newUnicodeFont(src.primaryData, 13, 16, 12)
	if err != nil {
		t.Fatal(err)
	}
	u.source = src
	u.primary = src.primary
	for _, value := range []string{"Español áéíóú ñ", "Ελληνικά Русский", "مرحبا بالعالم", "שלום עולם", "नमस्ते", "你好 日本語 한글", "😀 👨‍👩‍👧‍👦 🇦🇷 1️⃣ ❤️"} {
		l := u.layout(value)
		if l.width <= 0 {
			t.Fatal("zero width", value)
		}
		for _, run := range l.runs {
			for _, glyph := range run.Glyphs {
				if glyph.GlyphID == 0 && glyph.Advance > 0 {
					t.Fatalf("missing glyph in %q", value)
				}
			}
		}
	}
	l := u.layout("1️⃣")
	if len(l.runs) != 1 || !src.color[l.runs[0].Face] || len(l.runs[0].Glyphs) != 1 {
		t.Fatalf("keycap not shaped as a single colour glyph: %+v", l.runs)
	}
	loaded := 0
	for _, entry := range src.entries {
		if entry.attempted {
			loaded++
		}
	}
	if loaded >= len(src.entries)/2 {
		t.Fatal("fallback pack was loaded eagerly", loaded)
	}
	t.Logf("%d of %d fallbacks loaded for all sample scripts and emoji", loaded, len(src.entries))
}
