package ui

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

func TestMessageRequiresItsOwnPressAndSupportsCancel(t *testing.T) {
	m, err := NewMessage(marketTextures{})
	if err != nil {
		t.Fatal(err)
	}
	m.Open("Delete?", true)
	x, y := 1024/2-128, 768/2-128
	ok := platform.Input{MouseX: x + 49, MouseY: y + 115, Clicked: true}
	if m.Update(&ok, 1024, 768) != MessageNone || !m.Visible {
		t.Fatal("the opening release confirmed the dialog")
	}
	ok.Pressed, ok.Clicked = true, false
	m.Update(&ok, 1024, 768)
	ok.Pressed, ok.Clicked, ok.MouseX = false, true, x+144
	if m.Update(&ok, 1024, 768) != MessageNone {
		t.Fatal("cross-button release activated the dialog")
	}
	if m.Update(&platform.Input{Keys: []platform.Key{platform.KeyEscape}}, 1024, 768) != MessageCancel || m.Visible {
		t.Fatal("Escape did not cancel")
	}
	m.Open("Failure", false)
	if m.Update(&platform.Input{Keys: []platform.Key{platform.KeyEnter}}, 1024, 768) != MessageOK || m.Visible {
		t.Fatal("Enter did not dismiss the notice")
	}
}

func TestDeletedCharacterCannotStartByClickDoubleClickOrEnter(t *testing.T) {
	c, err := NewCharSelect(marketTextures{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	c.Selected = 0
	c.Chars[0] = &proto.CharacterSlot{ID: 10, PendingDelete: true}
	x, y := charPanelOrigin(1024, 768)
	for i := 0; i < 2; i++ {
		for _, pressed := range []bool{true, false} {
			if c.Update(&platform.Input{MouseX: x + 44, MouseY: y + 245, Pressed: pressed, Clicked: !pressed}, 1024, 768, .01) != CharNone {
				t.Fatal("deleted character entered by double click")
			}
		}
	}
	if c.Update(&platform.Input{Keys: []platform.Key{platform.KeyEnter}}, 1024, 768, .01) != CharNone || !c.Start.Disabled {
		t.Fatal("deleted character entered by Enter")
	}
}
