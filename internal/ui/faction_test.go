package ui

import (
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"testing"
)

func TestFactionRequiresSelectionAndBothConfirmations(t *testing.T) {
	for country := 0; country < 2; country++ {
		v, err := NewFaction(marketTextures{}, nil)
		if err != nil {
			t.Fatal(err)
		}
		click := func(x, y int) int {
			v.Update(&platform.Input{MouseX: x, MouseY: y, Pressed: true}, 1024, 768)
			return v.Update(&platform.Input{MouseX: x, MouseY: y, Clicked: true}, 1024, 768)
		}
		if got := v.Update(&platform.Input{Keys: []platform.Key{platform.KeyEnter}}, 1024, 768); got != FactionNone || v.step != 0 {
			t.Fatal("unselected faction submitted")
		}
		x, y := FactionPanel(1024, 768, country)
		click(x+30, y+100)
		if v.Selected != country {
			t.Fatal("wrong selected faction", v.Selected)
		}
		if got := click(1024/2-148+36, 768/2-267+479); got != FactionNone || v.step != 1 {
			t.Fatal("selection skipped first confirmation")
		}
		if got := click(1024/2-77, 768/2+198); got != FactionNone || v.step != 2 {
			t.Fatal("first confirmation skipped second")
		}
		if got := click(1024/2-77, 768/2+80); got != country {
			t.Fatal("confirmed wrong faction", got)
		}
		v.Busy = true
		if got := click(1024/2-77, 768/2+80); got != FactionNone {
			t.Fatal("duplicate submission")
		}
	}
}
func TestFactionCancellationDoesNotChoose(t *testing.T) {
	v, err := NewFaction(marketTextures{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	v.Update(&platform.Input{Chars: []rune{'f'}, Keys: []platform.Key{platform.KeyEnter}}, 1024, 768)
	if got := v.Update(&platform.Input{Keys: []platform.Key{platform.KeyEscape}}, 1024, 768); got != FactionNone || v.step != 0 {
		t.Fatal("dialog cancel chose a faction")
	}
	if got := v.Update(&platform.Input{Keys: []platform.Key{platform.KeyEscape}}, 1024, 768); got != FactionExit {
		t.Fatal("base screen cancel did not exit")
	}
	v.Reset()
	if v.Selected != -1 || v.step != 0 || v.Busy {
		t.Fatal("stale faction selection after reconnect")
	}
}
