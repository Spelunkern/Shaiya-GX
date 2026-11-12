package ui

import (
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"testing"
)

func TestCreationNativeChoicesAndSubmission(t *testing.T) {
	for faction := uint8(0); faction < 2; faction++ {
		c, err := NewCharCreate(marketTextures{}, nil)
		if err != nil {
			t.Fatal(err)
		}
		c.Open(3, faction, 2)
		x, y := CreationOrigin(1024, 768)
		click := func(cx, cy int) CharAction {
			c.Update(&platform.Input{Pressed: true, MouseX: x + cx, MouseY: y + cy}, 1024, 768)
			return c.Update(&platform.Input{Clicked: true, MouseX: x + cx, MouseY: y + cy}, 1024, 768)
		}
		for race := 0; race < 2; race++ {
			click(71+race*78, 291)
			if c.Draft.Race/2 != faction {
				t.Fatal("race escaped account faction")
			}
			for job := 0; job < 3; job++ {
				click(23+job*75, 390)
				if c.Draft.Class != familyJobs[c.Draft.Race][job] {
					t.Fatal("native job button selected wrong class")
				}
			}
		}
		click(149, 329)
		if c.Draft.Gender != 1 {
			t.Fatal("female button did not change sex")
		}
		click(98, 441) // previous hair wraps to the last appearance
		if c.Draft.Hair != 4 {
			t.Fatal("previous hair did not wrap")
		}
		click(190, 441)
		if c.Draft.Hair != 0 {
			t.Fatal("next hair did not wrap")
		}
		click(190, 474)
		click(190, 508)
		if c.Draft.Face != 1 || c.Draft.Height != 3 {
			t.Fatal("appearance controls", c.Draft)
		}
		click(71, 584)
		if c.Draft.Mode != 2 {
			t.Fatal("hard mode not selected")
		}
		click(149, 584)
		if c.Draft.Mode != 2 {
			t.Fatal("unavailable ultimate mode selected")
		}
		c.Name.Text = "A"
		if got := click(38, 623); got != CharNone {
			t.Fatal("short name submitted")
		}
		c.Name.Text = "New Hero"
		if got := click(38, 623); got != CharCreate || c.Draft.Name != "NewHero" || c.Draft.Slot != 3 {
			t.Fatal("incorrect native submission", got, c.Draft)
		}
		c.Busy = true
		before := c.Draft
		click(71, 291)
		if c.Draft != before || c.Update(&platform.Input{Keys: []platform.Key{platform.KeyEnter}}, 1024, 768) != CharNone {
			t.Fatal("pending creation allowed edits/submission")
		}
	}
}
func TestCreationReleaseMustMatchPressedControl(t *testing.T) {
	c, err := NewCharCreate(marketTextures{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	c.Open(0, 0, 3)
	x, y := CreationOrigin(1024, 768)
	c.Update(&platform.Input{Pressed: true, MouseX: x + 71, MouseY: y + 291}, 1024, 768)
	c.Update(&platform.Input{Clicked: true, MouseX: x + 149, MouseY: y + 291}, 1024, 768)
	if c.Draft.Race != 0 {
		t.Fatal("drag activated another race")
	}
}
