package ui

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/platform"
)

func TestRaidSlotsAndSelection(t *testing.T) {
	slots := map[uint16]uint32{}
	for slot := uint16(0); slot < 30; slot++ {
		slots[slot] = uint32(slot) + 1
	}
	for _, size := range [][2]int{{1024, 768}, {800, 768}} {
		p := &PartyPanel{}
		for slot, id := range slots {
			x, y := RaidSlotPosition(size[0], size[1], slot)
			if x < 0 || y < 0 || x+RaidSlotWidth > size[0] || y+RaidSlotHeight > size[1] {
				t.Fatalf("slot %d outside %v: %d,%d", slot, size, x, y)
			}
			if got := p.RaidMemberAt(size[0], size[1], x+5, y+5, slots); got != id {
				t.Fatalf("slot %d resolved %d, want %d", slot, got, id)
			}
		}
	}
	p := &PartyPanel{}
	x, y := RaidSlotPosition(1024, 768, 23)
	p.UpdateRaid(&platform.Input{Pressed: true, MouseDown: true, MouseX: x + 5, MouseY: y + 5}, 1024, 768, slots, false)
	if id, _, captured := p.UpdateRaid(&platform.Input{Clicked: true, MouseX: x + 5, MouseY: y + 5}, 1024, 768, slots, false); id != 24 || !captured {
		t.Fatalf("raid selection: %d %v", id, captured)
	}
	rx, ry := p.RaidOrigin(1024, 768)
	for _, leader := range []bool{false, true} {
		p.ResetRaidSettings()
		p.SetRaidSettings(1, false)
		p.UpdateRaid(&platform.Input{Pressed: true, MouseDown: true, MouseX: rx + 146, MouseY: ry + 315}, 1024, 768, slots, leader)
		_, action, captured := p.UpdateRaid(&platform.Input{Clicked: true, MouseX: rx + 146, MouseY: ry + 315}, 1024, 768, slots, leader)
		division, _ := p.RaidSettings()
		if !captured || action != RaidNone || leader && division != 2 || !leader && division != 1 {
			t.Fatalf("leader %v: action %v captured %v", leader, action, captured)
		}
	}
	p.RaidPosition.Move(35, 55, 1024, 768, RaidWidth, RaidHeight)
	x, y = p.RaidSlotPosition(1024, 768, 23)
	if got := p.RaidMemberAt(1024, 768, x+5, y+5, slots); got != 24 {
		t.Fatal("moved raid window did not move hit targets", got)
	}
}

func TestCompactRaidGroupSelectionAndDraft(t *testing.T) {
	p := &PartyPanel{raidCompact: true, raidGroups: true, raidOpacity: 1}
	slots := map[uint16]uint32{0: 1, 6: 7, 7: 8, 12: 13, 13: 14}
	x, y := p.RaidCompactOrigin(1024, 768)
	click := func(px, py int) (uint32, bool) {
		p.UpdateRaidCompact(&platform.Input{Pressed: true, MouseDown: true, MouseX: px, MouseY: py}, 1024, 768, 8, slots)
		return p.UpdateRaidCompact(&platform.Input{Clicked: true, MouseX: px, MouseY: py}, 1024, 768, 8, slots)
	}
	if id, captured := click(x+8, y+24+PartyRowHeight+8); id != 8 || !captured {
		t.Fatalf("own group row: %d %v", id, captured)
	}
	if id, captured := click(x+20+2*24+5, y+12); id != 0 || !captured {
		t.Fatalf("group tab: %d %v", id, captured)
	}
	if id, captured := click(x+8, y+24+PartyRowHeight+8); id != 14 || !captured {
		t.Fatalf("selected group row: %d %v", id, captured)
	}
	p.SetRaidSettings(1, false)
	rx, ry := p.RaidOrigin(1024, 768)
	check := func(dx, dy int) {
		p.UpdateRaid(&platform.Input{Pressed: true, MouseDown: true, MouseX: rx + dx, MouseY: ry + dy}, 1024, 768, slots, false)
		p.UpdateRaid(&platform.Input{Clicked: true, MouseX: rx + dx, MouseY: ry + dy}, 1024, 768, slots, false)
	}
	check(445, 315)
	if p.raidDraftCompact || !p.RaidCompactVisible() {
		t.Fatal("view toggle changed before OK")
	}
	p.ResetRaidSettings()
	p.SetRaidSettings(1, false)
	if !p.raidDraftCompact {
		t.Fatal("cancel retained compact toggle")
	}
	p.UpdateRaid(&platform.Input{Pressed: true, MouseDown: true, MouseX: rx + 308, MouseY: ry + 370}, 1024, 768, slots, false)
	p.UpdateRaid(&platform.Input{Clicked: true, MouseX: rx + 429, MouseY: ry + 370}, 1024, 768, slots, false)
	if p.raidDraftOpacity < .49 || p.raidDraftOpacity > .51 || p.raidOpacity != 1 {
		t.Fatal("opacity draft", p.raidDraftOpacity, p.raidOpacity)
	}
	p.ApplyRaidViewSettings()
	if p.raidOpacity < .49 || p.raidOpacity > .51 {
		t.Fatal("opacity not applied")
	}
	p.raidGroups = false
	if id := p.RaidCompactMemberAt(1024, 768, x+8, y+PartyRowHeight+8, 8, slots); id != 8 {
		t.Fatal("disabled group selection did not follow own group", id)
	}
}
