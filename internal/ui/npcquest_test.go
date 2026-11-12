package ui

import (
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"testing"
)

func TestNPCQuestRewardChoiceRequiresPressAndRelease(t *testing.T) {
	v := &NPCQuest{held: -1}
	x, y := NPCQuestOrigin(1024, 768)
	p := v.Update(&platform.Input{MouseX: x + 30, MouseY: y + 355, Pressed: true, MouseDown: true}, 1024, 768, 2, 0, true, true, 2)
	if p.Row != -1 {
		t.Fatal("press chose reward")
	}
	p = v.Update(&platform.Input{MouseX: x + 30, MouseY: y + 355, Clicked: true}, 1024, 768, 2, 0, true, true, 2)
	if p.Row != 1 {
		t.Fatalf("choice row=%d", p.Row)
	}
	p = v.Update(&platform.Input{MouseX: x + 30, MouseY: y + 315, Clicked: true}, 1024, 768, 2, 0, true, true, 2)
	if p.Row != -1 {
		t.Fatal("release without press chose reward")
	}
}

func TestNPCQuestDescriptionScrollPreservesListAndRewards(t *testing.T) {
	v := &NPCQuest{held: -1}
	x, y := NPCQuestOrigin(1024, 768)
	a := v.Update(&platform.Input{MouseX: x + 30, MouseY: y + 150, Wheel: -120}, 1024, 768, 10, 2, true, true, 2)
	if v.textOffset != 3 || a.Offset != 2 || a.Row != -1 {
		t.Fatal("description wheel affected list or reward selection", v.textOffset, a)
	}
	v.Update(&platform.Input{MouseX: x + 30, MouseY: y + 350, Wheel: -120}, 1024, 768, 10, 2, true, true, 2)
	if v.textOffset != 3 {
		t.Fatal("reward area moved description")
	}
	v.ResetDescriptionScroll()
	if v.textOffset != 0 {
		t.Fatal("selection did not reset description")
	}
}
