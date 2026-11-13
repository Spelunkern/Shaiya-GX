package ui

import (
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"testing"
)

func TestQuestDescriptionScrollIsIndependentOfList(t *testing.T) {
	q := &QuestLog{held: -1, textLines: 100}
	x, y := q.Origin(1024, 768)
	_, list, _, _ := q.Update(&platform.Input{MouseX: x + 300, MouseY: y + 150, Wheel: -120}, 1024, 768, 12, 2, true)
	if q.textOffset != 3 || list != 2 {
		t.Fatalf("description scroll moved list: text=%d list=%d", q.textOffset, list)
	}
	_, list, _, _ = q.Update(&platform.Input{MouseX: x + 30, MouseY: y + 150, Wheel: -120}, 1024, 768, 12, 2, true)
	if q.textOffset != 3 || list != 3 {
		t.Fatalf("list scroll moved text: text=%d list=%d", q.textOffset, list)
	}
	q.Update(&platform.Input{MouseX: x + 300, MouseY: y + 150, Wheel: 120}, 1024, 768, 12, 2, true)
	if q.textOffset != 0 {
		t.Fatal("wheel up did not return to top")
	}
	q.textOffset = 6
	q.ResetDescriptionScroll()
	if q.textOffset != 0 {
		t.Fatal("description position survived selection change")
	}
}

func TestQuestScrollArrowDoesNotSelectQuestUnderneath(t *testing.T) {
	q := &QuestLog{held: -1, listScroll: newPanelScroll(nil, 227, 52, 237)}
	x, y := q.Origin(1024, 768)
	in := platform.Input{MouseX: x + 230, MouseY: y + 275, Pressed: true, MouseDown: true}
	q.Update(&in, 1024, 768, 20, 0, true)
	in.Pressed, in.MouseDown, in.Clicked = false, false, true
	row, scroll, _, _ := q.Update(&in, 1024, 768, 20, 0, true)
	if row != -1 || scroll != 1 {
		t.Fatal("scroll arrow selected a row or did not scroll", row, scroll)
	}
}
