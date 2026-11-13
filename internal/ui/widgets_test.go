package ui

import (
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"testing"
)

func TestUnicodeFieldBudgetAndGraphemeBackspace(t *testing.T) {
	f := TextField{Max: 127}
	f.Edit(&platform.Input{Chars: []rune("你好 👨‍👩‍👧‍👦")})
	if f.Text != "你好 👨‍👩‍👧‍👦" {
		t.Fatal(f.Text)
	}
	f.Edit(&platform.Input{Keys: []platform.Key{platform.KeyBackspace}})
	if f.Text != "你好 " {
		t.Fatal("emoji split by backspace", f.Text)
	}
	f = TextField{Max: 4}
	f.Edit(&platform.Input{Chars: []rune("e\u0301x")})
	if f.Text != "e\u0301x" {
		t.Fatal(f.Text)
	}
	f.Edit(&platform.Input{Keys: []platform.Key{platform.KeyBackspace}})
	f.Edit(&platform.Input{Keys: []platform.Key{platform.KeyBackspace}})
	if f.Text != "" {
		t.Fatal("combining mark split", f.Text)
	}
}
