package ui

import (
	"math"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

func TestPreviewNavigationRotationPauseResetAndZoom(t *testing.T) {
	for _, creation := range []bool{false, true} {
		n := newPreviewNavigation(&loader{t: marketTextures{}})
		click := func(index int) {
			b := &n.buttons[index]
			in := platform.Input{MouseX: 100 + b.OffX + 1, MouseY: 200 + b.OffY + 1, Pressed: true}
			n.Update(&in, 100, 200, true, creation)
			in.Pressed, in.Clicked = false, true
			n.Update(&in, 100, 200, true, creation)
		}
		click(0)
		n.Advance(1, 1)
		if n.Angle != -1 {
			t.Fatal("rotation did not continue after the button release")
		}
		click(2)
		n.Advance(1, 1)
		if n.Angle != -1 {
			t.Fatal("pause changed the orientation")
		}
		click(4)
		if !n.Face || creation && n.Angle != 0 || !creation && n.Angle != -1 {
			t.Fatal("zoom did not preserve the screen's orientation policy")
		}
		click(5)
		if n.Face {
			t.Fatal("full-body view not restored")
		}
		click(1)
		n.Advance(2, .2)
		click(3)
		n.Advance(1, 1)
		if n.Angle != 0 {
			t.Fatal("stop did not restore and hold the original orientation")
		}
		click(1)
		n.Advance(100000, 1)
		if math.Abs(n.Angle) > math.Pi {
			t.Fatal("rotation accumulated without a bound")
		}
	}
}

func TestPreviewNavigationPressOwnershipAndPending(t *testing.T) {
	c, err := NewCharCreate(marketTextures{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	c.Open(0, 0, 3)
	x, y := CreationNavigationOrigin(1024, 768)
	draft := c.Draft
	c.Update(&platform.Input{MouseX: x + 9, MouseY: y + 39, Pressed: true}, 1024, 768)
	c.Update(&platform.Input{MouseX: x + 100, MouseY: y + 39, Clicked: true}, 1024, 768)
	c.Navigation.Advance(1, 1)
	if c.Navigation.Angle != 0 || c.Draft != draft {
		t.Fatal("cross-button release activated navigation or modified the draft")
	}
	c.Busy = true
	for _, pressed := range []bool{true, false} {
		c.Update(&platform.Input{MouseX: x + 54, MouseY: y + 8, Pressed: pressed, Clicked: !pressed, Keys: []platform.Key{platform.KeyUp}}, 1024, 768)
	}
	if c.Navigation.Face {
		t.Fatal("pending creation accepted navigation")
	}
	c.Open(0, 0, 3)
	c.Update(&platform.Input{Keys: []platform.Key{platform.KeyUp, platform.KeyLeft}}, 1024, 768)
	c.Navigation.Advance(1, .2)
	if !c.Navigation.Face || c.Navigation.Angle != -.2 {
		t.Fatal("keyboard navigation not connected to creation")
	}
	c.Open(1, 1, 3)
	if c.Navigation.Face || c.Navigation.Angle != 0 {
		t.Fatal("new creation retained the previous camera")
	}
}

func TestSelectionNavigationResetsOnlyWhenCharacterChanges(t *testing.T) {
	c, err := NewCharSelect(marketTextures{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	slots := map[uint8]proto.CharacterSlot{0: {ID: 1}, 1: {ID: 2}}
	c.SetChars(slots)
	c.Navigation.Angle, c.Navigation.Face = .7, true
	c.SetChars(slots)
	if c.Navigation.Angle != .7 || !c.Navigation.Face {
		t.Fatal("roster refresh reset the current preview")
	}
	x, y := charPanelOrigin(1024, 768)
	for _, pressed := range []bool{true, false} {
		c.Update(&platform.Input{MouseX: x + 44, MouseY: y + 281, Pressed: pressed, Clicked: !pressed}, 1024, 768, .01)
	}
	if c.Selected != 1 || c.Navigation.Angle != 0 || c.Navigation.Face {
		t.Fatal("selecting another character kept its predecessor's camera")
	}
	c.Navigation.Face = true
	delete(slots, 1)
	c.SetChars(slots)
	if c.Selected != 0 || c.Navigation.Face {
		t.Fatal("deleted selection retained its camera")
	}
}
