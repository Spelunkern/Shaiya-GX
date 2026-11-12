package ui

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

func TestMallSelectionScrollingAndGiftClaim(t *testing.T) {
	m, err := NewMall(marketTextures{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	x, y := m.Origin(1024, 768)
	click := func(dx, dy, count int, busy bool) MallAction {
		m.Update(&platform.Input{Pressed: true, MouseDown: true, MouseX: x + dx, MouseY: y + dy}, 1024, 768, count, busy)
		a, _ := m.Update(&platform.Input{Clicked: true, MouseX: x + dx, MouseY: y + dy}, 1024, 768, count, busy)
		return a
	}
	if a := click(295, 410, 20, false); a != MallNone {
		t.Fatal("buy without selection", a)
	}
	click(30, 175, 20, false)
	if m.Selected != 0 {
		t.Fatal("product selection", m.Selected)
	}
	if a := click(295, 410, 20, true); a != MallNone {
		t.Fatal("busy purchase accepted")
	}
	if a := click(295, 410, 20, false); a != MallBuy {
		t.Fatal("purchase action", a)
	}
	m.Update(&platform.Input{MouseX: x + 50, MouseY: y + 200, Wheel: -120}, 1024, 768, 20, false)
	if m.Scroll != 1 {
		t.Fatal("wheel did not scroll", m.Scroll)
	}
	m.Update(&platform.Input{Pressed: true, MouseDown: true, MouseX: x + 568, MouseY: y + 330}, 1024, 768, 20, false)
	m.Update(&platform.Input{Clicked: true, MouseX: x + 568, MouseY: y + 330}, 1024, 768, 20, false)
	if m.Scroll < 5 {
		t.Fatal("scrollbar did not seek", m.Scroll)
	}
	if a := click(35, 410, 20, false); a != MallShowGifts || !m.GiftOpen {
		t.Fatal("gift box action", a)
	}
	gx, gy := m.GiftOrigin(1024, 768)
	stored := map[uint8]proto.MallStoredItem{0: {Slot: 0, Type: 25, TypeID: 4, Count: 1}, 44: {Slot: 44, Type: 25, TypeID: 4, Count: 2}}
	dx, dy := MallGiftCell(0)
	if slot, _, captured := m.UpdateGifts(&platform.Input{RightPressed: true, RightDown: true, MouseX: gx + dx + 1, MouseY: gy + dy + 1}, 1024, 768, stored, 1, false); slot != 0 || !captured {
		t.Fatal("right click claim", slot, captured)
	}
	if slot, _, _ := m.UpdateGifts(&platform.Input{RightPressed: true, MouseX: gx + dx + 1, MouseY: gy + dy + 1}, 1024, 768, stored, 1, true); slot != -1 {
		t.Fatal("busy gift claimed")
	}
	tabX, tabY := gx+10+52+5, gy+60
	m.UpdateGifts(&platform.Input{Pressed: true, MouseDown: true, MouseX: tabX, MouseY: tabY}, 1024, 768, stored, 2, false)
	m.UpdateGifts(&platform.Input{Clicked: true, MouseX: tabX, MouseY: tabY}, 1024, 768, stored, 2, false)
	if m.GiftPage != 1 {
		t.Fatal("gift page", m.GiftPage)
	}
	dx, dy = MallGiftCell(44)
	if slot, _, _ := m.UpdateGifts(&platform.Input{RightPressed: true, MouseX: gx + dx + 1, MouseY: gy + dy + 1}, 1024, 768, stored, 3, false); slot != 44 {
		t.Fatal("paged gift claim", slot)
	}
}
