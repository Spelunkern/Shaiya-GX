package ui

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

type marketTextures struct{}

func (marketTextures) Interface(string) (*render.Texture, error) { return nil, nil }

func TestMarketCellBoundaries(t *testing.T) {
	// Coordinates from Game.exe 0x508a20, including the last row/column.
	for _, tc := range []struct{ slot, x, y int }{{0, 15, 112}, {5, 210, 112}, {29, 210, 268}} {
		x, y, ok := MarketCell(tc.slot)
		if !ok || x != tc.x || y != tc.y {
			t.Fatalf("slot %d = %d,%d", tc.slot, x, y)
		}
		for _, p := range [][2]int{{x, y}, {x + 31, y + 31}} {
			if got, ok := MarketHit(p[0], p[1]); !ok || got != tc.slot {
				t.Fatalf("hit %v = %d,%v", p, got, ok)
			}
		}
	}
	for _, p := range [][2]int{{14, 112}, {47, 112}, {15, 144}, {242, 268}, {15, 300}} {
		if _, ok := MarketHit(p[0], p[1]); ok {
			t.Fatalf("gap %v selected a product", p)
		}
	}
	for _, slot := range []int{-1, 30} {
		if _, _, ok := MarketCell(slot); ok {
			t.Fatal("invalid slot", slot)
		}
	}
}

func TestMarketPageLimitsPendingAndClose(t *testing.T) {
	m, err := NewMarket(marketTextures{})
	if err != nil {
		t.Fatal(err)
	}
	x, y := MarketOrigin(1024, 768)
	page := 0
	click := func(cx, cy, count int, busy bool) bool {
		t.Helper()
		m.Update(&platform.Input{MouseX: x + cx, MouseY: y + cy, Pressed: true, MouseDown: true}, 1024, 768, page, count, busy)
		var closed bool
		page, closed = m.Update(&platform.Input{MouseX: x + cx, MouseY: y + cy, Clicked: true}, 1024, 768, page, count, busy)
		return closed
	}
	click(96, 309, 60, false)
	if page != 0 {
		t.Fatal("previous page below zero")
	}
	click(146, 309, 60, false)
	if page != 1 {
		t.Fatal("next page unavailable")
	}
	click(146, 309, 60, false)
	if page != 1 {
		t.Fatal("empty page after an exact multiple of 30")
	}
	click(96, 309, 60, true)
	if page != 1 {
		t.Fatal("pending request allowed page change")
	}
	click(96, 309, 60, false)
	if page != 0 {
		t.Fatal("previous page unavailable")
	}
	click(146, 309, 30, false)
	if page != 0 {
		t.Fatal("single-page catalog changed page")
	}
	if !click(229, 6, 30, true) {
		t.Fatal("pending request blocked window close")
	}
}
