package ui

import (
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"testing"
)

func TestQuestDescriptionScrollIsIndependentOfList(t *testing.T) {
	q := &QuestLog{held: -1}
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
