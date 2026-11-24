package app

import (
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

func TestAuctionRoutesBoardAndKeepsSelectionOnRefresh(t *testing.T) {
	a := inputApp()
	a.state.Entities[42] = &game.Entity{ID: 42, Kind: game.KindNpc, NpcType: 1, TypeID: 240}
	a.npcDefs[sdata.NPCKey{Type: 1, ID: 240}] = sdata.NPC{MerchantType: proto.AuctionBoard}
	a.shop.npcID = 42
	a.beginAuction(42)
	a.auction.rows = []proto.AuctionRow{{ID: 3}, {ID: 5}}
	a.auction.selected = 1
	a.auction.pending = true
	a.auction.sequence = 2
	a.applyAuction(proto.AuctionPacket(proto.AuctionReply{Sequence: 1, NPC: 42, Rows: []proto.AuctionRow{{ID: 99}}}))
	if !a.auction.pending || a.auction.row().ID != 5 {
		t.Fatal("stale reply replaced the current listing")
	}
	a.applyAuction(proto.AuctionPacket(proto.AuctionReply{Sequence: 2, NPC: 42, Rows: []proto.AuctionRow{{ID: 5}, {ID: 3}}}))
	if a.auction.pending || a.auction.selected != 0 || a.auction.row().ID != 5 {
		t.Fatal("refresh lost selection")
	}
	a.state.Self.Dead = true
	a.updateAuction(&platform.Input{}, 1024, 768)
	if a.auction.npc != 0 {
		t.Fatal("market stayed open after death")
	}
}

func TestAuctionAcceptsExplicitInventoryDrag(t *testing.T) {
	a := inputApp()
	a.state.Entities[42] = &game.Entity{ID: 42, Kind: game.KindNpc, NpcType: 1, TypeID: 240}
	a.npcDefs[sdata.NPCKey{Type: 1, ID: 240}] = sdata.NPC{MerchantType: proto.AuctionBoard}
	a.shop.npcID = 42
	a.beginAuction(42)
	a.auction.view = proto.AuctionSelling
	a.auctionAction("setup")
	key := [2]uint8{1, 0}
	a.inventory.bag = 1
	a.state.Inventory[key] = proto.InventoryItem{Bag: 1, Type: 25, TypeID: 1, Count: 9}
	ix, iy := a.inventoryUI.Origin(1024, 768)
	sx, sy, _, _ := ui.InventoryCell(1, 0)
	a.worldControls(&platform.Input{MouseX: ix + sx + 2, MouseY: iy + sy + 2, Pressed: true, MouseDown: true}, 1024, 768)
	if a.auction.source.Type != 0 {
		t.Fatal("inventory click picked a listing source")
	}
	slot := auctionDialogBox("setup", 1024, 768).source()
	a.worldControls(&platform.Input{MouseX: slot.x + 4, MouseY: slot.y + 4, Clicked: true}, 1024, 768)
	if a.auction.source != a.state.Inventory[key] || a.auction.fields[0] != "9" || a.auction.token == "" {
		t.Fatal("auction did not accept inventory drop", a.auction.source)
	}
}

// Every control of every tab sits where Game.exe puts it and fits the window.
func TestAuctionNativeControlsMatchOriginalGeometry(t *testing.T) {
	a := inputApp()
	a.auction = auctionView{npc: 42, selected: -1, focus: -1}
	for view := proto.AuctionBrowse; view <= proto.AuctionWatch; view++ {
		a.auction.view = view
		for _, c := range a.auctionControls(100, 50) {
			x, y := c.b.X()-100, c.b.Y()-50
			if x < 0 || y < 0 || x+c.b.HitW > auctionWidth || y+c.b.HitH > auctionPixels {
				t.Fatalf("view %d control at %d,%d outside the window", view, x, y)
			}
		}
	}
	a.auction.view = proto.AuctionBrowse
	want := map[string][2]int{"tab0": {0x43, 0x14}, "tab3": {0x229, 0x14}, "close": {0x2c5, 6}, "find": {0x293, 0x43},
		"bid": {0x146, 0x233}, "exact": {0x272, 0x233}, "prev": {0x17f, 0x1f7}, "next": {0x1db, 0x1f7}, "cat4": {0x12, 0x84 + 4*0x19}}
	for id, at := range want {
		a.auctionControls(0, 0)
		b := a.auctionUI.btn[id]
		if b == nil || b.X() != at[0] || b.Y() != at[1] {
			t.Fatalf("%s is not at %v", id, at)
		}
	}
	x, y, width := a.auctionPanel(1024, 768)
	if x != 1024/2-0x172 || y != 768/2-280 || width != 0x2e4 {
		t.Fatal("window origin", x, y, width)
	}
}

func TestAuctionExpandedCategoryMovesFollowingGroupsToTheBottom(t *testing.T) {
	for i, want := range []int{0x84, 0x9d, 0xb6, 0xcf, 0xe8} {
		if auctionGroupY(-1, i) != want {
			t.Fatal("collapsed group", i)
		}
	}
	if auctionGroupY(0, 0) != 0x84 || auctionGroupY(0, 1) != 0x179 || auctionGroupY(1, 2) != 0x192 || auctionGroupY(2, 4) != 0x1c4 {
		t.Fatal("expanded groups do not follow 0x455e40")
	}
}

func TestAuctionSearchBarSelectsLevelsGradeAndCategory(t *testing.T) {
	a := inputApp()
	a.auction = auctionView{npc: 42, selected: -1, focus: -1, grade: 7}
	press := func(x, y int) { a.updateAuctionSearchBar(&platform.Input{Pressed: true}, x, y) }
	press(0xc1, 0x42+4)
	if a.auction.dropdown != 1 {
		t.Fatal("min level dropdown did not open")
	}
	press(0xc1, 0x42+0x19+3*0xf+2)
	if a.auction.minIdx != 3 || a.auction.dropdown != 0 {
		t.Fatal("min level not chosen", a.auction.minIdx)
	}
	press(0x1b8-90, 0x42+4)
	press(0x1b8-90, 0x42+0x19+2*0xf+2)
	if a.auction.grade != 2 {
		t.Fatal("grade not chosen")
	}
	press(0x24, 0x44)
	if !a.auction.unchecked[0] {
		t.Fatal("Light filter did not toggle")
	}
	press(0x1e0, 0x48)
	if a.auction.focus != 0 {
		t.Fatal("search field did not take focus")
	}
	a.auction.group, a.auction.sub = 2, 4
	a.world = nil
	a.auctionSearch()
	if a.auction.minLevel != "30" || a.auction.maxLevel != "" || a.auction.category != 2 || a.auction.subcategory != 5 {
		t.Fatalf("search state %+v", a.auction)
	}
	if a.auction.searchFaction != proto.AuctionFury || a.auction.searchGrade == nil || *a.auction.searchGrade != 2 {
		t.Fatalf("submitted faction and grade %+v", a.auction)
	}
	a.auction.unchecked, a.auction.grade = [2]bool{false, true}, 6
	a.auction.pending = true
	a.auctionSearch()
	if a.auction.searchFaction != proto.AuctionFury || a.auction.searchGrade == nil || *a.auction.searchGrade != 2 {
		t.Fatal("in-flight search filters changed")
	}
	a.auction.pending = false
	a.auctionSearch()
	if a.auction.searchFaction != proto.AuctionLight || a.auction.searchGrade == nil || *a.auction.searchGrade != 7 {
		t.Fatal("native Light or Ancient selector lost")
	}
	a.auction.unchecked, a.auction.grade = [2]bool{true, true}, 7
	a.auctionSearch()
	if a.auction.searchFaction != proto.AuctionAnyFaction || a.auction.searchGrade != nil {
		t.Fatal("equal checkbox states and All grades should be unrestricted")
	}
}

func TestAuctionPairedListsKeepIndependentSelectionAndPage(t *testing.T) {
	a := inputApp()
	a.auction = auctionView{
		npc: 42, view: proto.AuctionItems, selected: 0, page: 1,
		rows:        []proto.AuctionRow{{ID: 101}, {ID: 102}},
		other:       auctionList{selected: -1},
		pending:     true,
		sequence:    4,
		requestView: proto.AuctionGold,
	}
	a.applyAuction(proto.AuctionPacket(proto.AuctionReply{
		Sequence: 4, NPC: 42, View: proto.AuctionGold, More: true,
		Rows: []proto.AuctionRow{{ID: 201}, {ID: 202}},
	}))
	v := &a.auction
	if v.view != proto.AuctionItems || v.row().ID != 101 || v.page != 1 {
		t.Fatal("gold reply changed the selected item claim or its page")
	}
	a.pickAuctionRow(proto.AuctionGold, 1, 0x7f)
	if v.view != proto.AuctionGold || v.row().ID != 202 || !v.more {
		t.Fatal("click did not select the gold claim")
	}
	a.pickAuctionRow(proto.AuctionItems, 0, 0x7f)
	if v.view != proto.AuctionItems || v.page != 1 || v.row().ID != 101 {
		t.Fatal("item pane lost its page or row")
	}
	if count, offset := pageScroll(v.paneList(proto.AuctionGold)); count != 4*auctionPage/2 || offset != 0 {
		t.Fatal("gold pane scrollbar", count, offset)
	}
}

func TestAuctionDoubleClickOpensDetailsAndDialogClosesFromItsButtons(t *testing.T) {
	a := inputApp()
	a.auction = auctionView{npc: 42, selected: -1, focus: -1, rows: []proto.AuctionRow{{ID: 1}, {ID: 2}}}
	a.pickAuctionRow(proto.AuctionBrowse, 1, 0x7c)
	a.clock += .1
	a.pickAuctionRow(proto.AuctionBrowse, 1, 0x7c)
	if a.auction.modal != "details" || a.auction.detail != proto.AuctionBrowse {
		t.Fatal("double click did not open details")
	}
	b := auctionDialogBox("details", 1024, 768)
	var closed bool
	for _, c := range a.auctionDialogControls(b.x, b.y) {
		if c.b.X() == b.x+0x8d && c.b.Y() == b.y+0x1d9 {
			c.do()
			closed = a.auction.modal == ""
		}
	}
	if !closed {
		t.Fatal("details Close button did not close")
	}
	a.auction.modal, a.auction.focus = "setup", -1
	b = auctionDialogBox("setup", 1024, 768)
	click := func(x, y int) {
		a.updateAuctionDialog(&platform.Input{MouseX: b.x + x, MouseY: b.y + y, Pressed: true, MouseDown: true}, 1024, 768)
		a.updateAuctionDialog(&platform.Input{MouseX: b.x + x, MouseY: b.y + y, Clicked: true}, 1024, 768)
	}
	click(0xb5+2, 0x8b+2)
	if a.auction.dropdown != 4 {
		t.Fatal("duration dropdown did not open")
	}
	click(0x70, 0xa4+3*0xf+2)
	if a.auction.duration != 3 {
		t.Fatal("duration entry was not selected")
	}
	click(0xcd+2, 0x71+2)
	if a.auction.focus != 2 {
		t.Fatal("buyout Input button did not focus the field")
	}
	click(0x8c+2, 0xd8+2)
	if a.auction.modal != "" {
		t.Fatal("setup Close button did not close")
	}
}

func TestAuctionTimeLeftFollowsNativeBands(t *testing.T) {
	now := time.Unix(1_000_000, 0)
	for _, c := range []struct {
		left int64
		text string
		want [4]float32
	}{{-1, "0", auctionRed}, {600, "10m", auctionRed}, {3 * 3600, "3H", auctionOrange}, {13 * 3600, "13H", auctionYellow}, {49 * 3600, "2D", auctionYellow}, {3600, "1H", auctionRed}} {
		text, color := auctionLeft(now.Unix()+c.left, now)
		if text != c.text || color != c.want {
			t.Fatalf("%d left: %q %v", c.left, text, color)
		}
	}
}

func TestAuctionTabAndSelectionClicksUseNativeHitBoxes(t *testing.T) {
	a := inputApp()
	const npc = 42
	a.state.Entities[npc] = &game.Entity{ID: npc}
	a.shop.npcID = npc
	a.auction = auctionView{npc: npc, selected: -1, focus: -1, grade: 7, rows: []proto.AuctionRow{{ID: 1}, {ID: 2}}}
	x, y, _ := a.auctionPanel(1024, 768)
	click := func(rx, ry int) {
		a.updateAuction(&platform.Input{MouseX: x + rx, MouseY: y + ry, Pressed: true, MouseDown: true}, 1024, 768)
		a.updateAuction(&platform.Input{MouseX: x + rx, MouseY: y + ry, Clicked: true}, 1024, 768)
	}
	click(0x9b+10, 0x7c+0x33+5)
	if a.auction.selected != 1 {
		t.Fatal("second result row was not selected")
	}
	click(0xe2+10, 0x14+10)
	if a.auction.view != proto.AuctionSelling {
		t.Fatal("Auction Progress tab did not open")
	}
	click(0x216+10, 0x231+10)
	click(0x2c5+5, 11)
	if a.auction.npc != 0 {
		t.Fatal("close button left the market open")
	}
}
