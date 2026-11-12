package ui

import (
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"testing"
)

func TestSkillBookRowsAndControls(t *testing.T) {
	for _, tc := range []struct{ row, y int }{{0, 79}, {16, 423}} {
		x, y, ok := SkillBookRow(tc.row)
		if !ok || x != 45 || y != tc.y {
			t.Fatal("native row", tc.row, x, y)
		}
		if row, ok := SkillBookHit(x+31, y+31); !ok || row != tc.row {
			t.Fatal("row hit")
		}
		if _, ok := SkillBookHit(x, y+32); ok {
			t.Fatal("row gap selected")
		}
	}
	b, err := NewSkillBook(marketTextures{})
	if err != nil {
		t.Fatal(err)
	}
	x, y := SkillBookOrigin(1024, 768)
	category, offset := 0, 0
	enabled := [SkillBookRows]bool{true}
	click := func(cx, cy int) SkillBookAction {
		b.Update(&platform.Input{MouseX: x + cx, MouseY: y + cy, Pressed: true, MouseDown: true}, 1024, 768, category, offset, 24, enabled)
		a := b.Update(&platform.Input{MouseX: x + cx, MouseY: y + cy, Clicked: true}, 1024, 768, category, offset, 24, enabled)
		category, offset = a.Category, a.Offset
		return a
	}
	if a := click(62, 96); a.Learn != 0 {
		t.Fatal("learn hit", a)
	}
	enabled[0] = false
	if a := click(62, 96); a.Learn != -1 {
		t.Fatal("disabled learn activated", a)
	}
	if a := click(4, 199); a.Category != 2 || a.Offset != 0 {
		t.Fatal("combat category", a)
	}
	a := b.Update(&platform.Input{MouseX: x + 100, MouseY: y + 200, Wheel: -120}, 1024, 768, category, 0, 24, enabled)
	if a.Offset != 2 {
		t.Fatal("scroll should advance one row", a)
	}
	if a := click(304, 9); !a.Close {
		t.Fatal("close did not activate")
	}
}
