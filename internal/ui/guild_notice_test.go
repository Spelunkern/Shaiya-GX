package ui

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/platform"
)

func TestGuildNoticeDialogLocksNameAndAllowsEmpty(t *testing.T) {
	d := &GuildCreateDialog{Name: TextField{Max: 24}, Remark: TextField{Max: 64}}
	d.OpenNotice("Guild", "Existing")
	d.Update(&platform.Input{Keys: []platform.Key{platform.KeyTab}, Chars: []rune{'!'}}, 1024, 768)
	if d.Name.Text != "Guild" || d.Remark.Text != "Existing!" {
		t.Fatal("notice edit changed name", d.Name.Text, d.Remark.Text)
	}
	d.Remark.Text = ""
	if d.Update(&platform.Input{Keys: []platform.Key{platform.KeyEnter}}, 1024, 768) != MessageOK {
		t.Fatal("empty notice could not be confirmed")
	}
}
