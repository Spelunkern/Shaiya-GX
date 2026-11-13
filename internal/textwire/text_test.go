package textwire

import (
	"testing"
	"unicode/utf8"
)

func TestUTF8AndLegacyDecode(t *testing.T) {
	for _, s := range []string{"Español", "你好 👨‍👩‍👧‍👦", "مرحبا", "🇦🇷"} {
		if got := Decode(Encode(s)); got != s {
			t.Fatal(s, got)
		}
	}
	if got := Decode([]byte{0xe9, 0, 0}); got != "é" {
		t.Fatal(got)
	}
}
func TestByteBudgetPreservesGraphemes(t *testing.T) {
	for _, s := range []string{"a你好b", "x👨‍👩‍👧‍👦z", "e\u0301!", "🇦🇷!"} {
		for n := 0; n <= len(s)+1; n++ {
			p := Prefix(s, n)
			if len(p) > n || !utf8.ValidString(p) {
				t.Fatal(s, n, p)
			}
		}
	}
	if got := Prefix("a👨‍👩‍👧‍👦", 8); got != "a" {
		t.Fatal(got)
	}
	if got := DropLast("ae\u0301"); got != "a" {
		t.Fatal(got)
	}
	if got := DropLast("a👨‍👩‍👧‍👦"); got != "a" {
		t.Fatal(got)
	}
}
