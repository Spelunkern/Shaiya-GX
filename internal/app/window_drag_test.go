package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

func TestWindowDragOwnsMotionReleaseAndRespectsClose(t *testing.T) {
	for _, panel := range []string{"inventory", "skills", "status", "subactions", "shop"} {
		t.Run(panel, func(t *testing.T) {
			a := inputApp()
			var position *ui.WindowPosition
			var x, y int
			switch panel {
			case "inventory":
				a.inventory.open = true
				position = &a.inventoryUI.Position
				x, y = a.inventoryUI.Origin(1024, 768)
			case "skills":
				a.skills.open = true
				position = &a.skillBook.Position
				x, y = a.skillBook.Origin(1024, 768)
			case "status":
				a.status.open = true
				position = &a.statusUI.Position
				x, y = a.statusUI.Origin(1024, 768)
			case "subactions":
				a.subActionsOpen = true
				position = &a.subActionsUI.Position
				x, y = a.subActionsUI.Origin(1024, 768)
			case "shop":
				a.shop.npcID = 42
				a.state.Entities[42] = &game.Entity{Kind: game.KindNpc, NpcType: 1}
				position = &a.marketUI.Position
				x, y = a.marketUI.Origin(1024, 768)
			}
			controls := a.worldControls(&platform.Input{MouseX: x + 50, MouseY: y + 15, Pressed: true, MouseDown: true, Down: "W"}, 1024, 768)
			if a.windowDrag.position != position || controls.Pressed || controls.MouseDown || controls.Down != "W" {
				t.Fatal("title failed to own press")
			}
			controls = a.worldControls(&platform.Input{MouseX: 150, MouseY: 135, MouseDown: true, Down: "W"}, 1024, 768)
			px, py := position.Origin(1024, 768, 256, 486, 0, 0)
			if !position.Moved || controls.MouseDown || px < 0 || py < 0 {
				t.Fatal("drag failed", position, px, py)
			}
			controls = a.worldControls(&platform.Input{MouseX: -100, MouseY: 2000, Clicked: true, Down: "W"}, 1024, 768)
			if a.windowDrag.position != nil || controls.Clicked || controls.Down != "W" {
				t.Fatal("release leaked to world")
			}
		})
	}
	a := inputApp()
	a.inventory.open = true
	x, y := a.inventoryUI.Origin(1024, 768)
	if a.updateWindowDrag(&platform.Input{MouseX: x + 235, MouseY: y + 12, Pressed: true, MouseDown: true}, 1024, 768) {
		t.Fatal("close control started title drag")
	}
	if a.updateWindowDrag(&platform.Input{MouseX: x + 30, MouseY: y + 70, Pressed: true, MouseDown: true}, 1024, 768) {
		t.Fatal("item cell started title drag")
	}
}

func TestUILayoutPersistsAcrossAppAndPreservesCaptureDefaults(t *testing.T) {
	testConfigDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", testConfigDir)
	t.Setenv("APPDATA", testConfigDir)
	a := inputApp()
	a.cfg.RememberLayout = true
	a.inventoryUI.Position.Move(200, 100, 1024, 768, ui.InventoryWidth, ui.InventoryHeight)
	a.subActionsUI.Position.Move(500, 80, 1024, 768, ui.SubActionWidth, ui.SubActionHeight)
	a.actionBar.ShowThird = true
	a.actionBar.Position[2].Vertical = true
	a.actionBar.Position[2].Move(100, 120, 1024, 768, 56, 449)
	a.saveUILayout()
	b := inputApp()
	b.cfg.RememberLayout = true
	b.loadUILayout()
	if !b.actionBar.ShowThird || b.actionBar.Position[2] != a.actionBar.Position[2] {
		t.Fatal("third bar layout did not survive restart")
	}
	if b.inventoryUI.Position != a.inventoryUI.Position {
		t.Fatal("layout did not survive restart")
	}
	if b.subActionsUI.Position != a.subActionsUI.Position {
		t.Fatal("sub-action layout did not survive restart")
	}
	capture := inputApp()
	capture.loadUILayout()
	if capture.inventoryUI.Position.Moved {
		t.Fatal("offscreen capture loaded user layout")
	}
	path, err := uiLayoutPath()
	if err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	capture.inventoryUI.Position.Move(1, 1, 1024, 768, 256, 486)
	capture.saveUILayout()
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("capture changed user's layout")
	}
	files, err := filepath.Glob(filepath.Join(filepath.Dir(path), ".ui-layout-*"))
	if err != nil || len(files) != 0 {
		t.Fatal("temporary persistence files leaked", files, err)
	}
	if err = os.WriteFile(path, []byte(`{"inventory":{"CenterX":100,"CenterY":-1,"Moved":true}}`), 0600); err != nil {
		t.Fatal(err)
	}
	b = inputApp()
	b.cfg.RememberLayout = true
	b.loadUILayout()
	if b.inventoryUI.Position.Moved || b.actionBar.ShowThird {
		t.Fatal("invalid layout or missing third-bar setting accepted")
	}
}

func TestRecordedPointWindowIsMovableAndNonModal(t *testing.T) {
	a := inputApp()
	if !a.loadRecordedPoints() {
		t.Fatal("failed to create recorded point controls")
	}
	a.teleportStone.open = true
	const w, h = 1024, 768
	x, y := a.recordedPoints.Origin(w, h)
	a.worldControls(&platform.Input{Pressed: true, MouseDown: true, MouseX: x + 80, MouseY: y + 14}, w, h)
	if a.windowDrag.position != &a.recordedPoints.Position {
		t.Fatal("recorded point title did not start a drag")
	}
	a.worldControls(&platform.Input{MouseDown: true, MouseX: x + 210, MouseY: y + 54}, w, h)
	release := a.worldControls(&platform.Input{Clicked: true, MouseX: x + 210, MouseY: y + 54}, w, h)
	nx, ny := a.recordedPoints.Origin(w, h)
	if nx != x+130 || ny != y+40 || release.Clicked || a.windowDrag.position != nil {
		t.Fatalf("drag did not move the window or leaked its release: %d %d", nx, ny)
	}

	a.worldControls(&platform.Input{Pressed: true, MouseDown: true, MouseX: nx + 30, MouseY: ny + 80}, w, h)
	release = a.worldControls(&platform.Input{Clicked: true, MouseX: nx - 20, MouseY: ny + 80}, w, h)
	if release.Clicked {
		t.Fatal("click outside leaked from a press inside the panel")
	}

	outside := a.worldControls(&platform.Input{Pressed: true, MouseDown: true, MouseX: 470, MouseY: 110, Down: "W"}, w, h)
	if !outside.Pressed || !outside.Held('W') || outside.SuppressAutoRun || !a.teleportStone.open {
		t.Fatal("open recorded point window blocked gameplay outside it")
	}
}

func TestPartyListIsMovableSelectsOnPlainClickAndPersists(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	a := inputApp()
	a.cfg.RememberLayout = true
	a.partyUI, _ = ui.NewPartyPanel(inputTextures{})
	a.state.Party.Members = []proto.PartyMember{{ID: 7}, {ID: 8}, {ID: 9}}
	const w, h = 1024, 768
	x, y := a.partyUI.Origin(w, h, 3)
	// A press that never moves still selects the member under it.
	a.worldControls(&platform.Input{Pressed: true, MouseDown: true, MouseX: x + 20, MouseY: y + ui.PartyRowHeight + 5}, w, h)
	a.worldControls(&platform.Input{Clicked: true, MouseX: x + 20, MouseY: y + ui.PartyRowHeight + 5}, w, h)
	if a.state.Target != 8 || a.partyUI.Position.Moved {
		t.Fatalf("plain click: target %d moved %v", a.state.Target, a.partyUI.Position.Moved)
	}
	// Moving past the threshold drags the list and does not change the target.
	a.worldControls(&platform.Input{Pressed: true, MouseDown: true, MouseX: x + 20, MouseY: y + 5}, w, h)
	a.worldControls(&platform.Input{MouseDown: true, MouseX: x + 220, MouseY: y + 105}, w, h)
	a.worldControls(&platform.Input{Clicked: true, MouseX: x + 220, MouseY: y + 105}, w, h)
	nx, ny := a.partyUI.Origin(w, h, 3)
	if nx != x+200 || ny != y+100 || a.state.Target != 8 {
		t.Fatalf("drag: origin %d,%d target %d", nx, ny, a.state.Target)
	}
	b := inputApp()
	b.cfg.RememberLayout = true
	b.partyUI, _ = ui.NewPartyPanel(inputTextures{})
	b.loadUILayout()
	if b.partyUI.Position != a.partyUI.Position {
		t.Fatal("party position did not persist")
	}
}

func TestRaidSideListIsMovable(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	a := inputApp()
	a.cfg.RememberLayout = true
	a.partyUI, _ = ui.NewPartyPanel(inputTextures{})
	a.state.SelfID = 1
	a.state.Party.Raid, a.state.Party.Leader = true, 1
	a.state.Party.Members = []proto.PartyMember{{ID: 1}, {ID: 2}}
	a.state.Party.RaidSlots = map[uint16]uint32{0: 1, 1: 2}
	a.party.raidHidden = true
	const w, h = 1024, 768
	x, y := a.partyUI.RaidCompactOrigin(w, h)
	py := y + 24 + ui.PartyRowHeight + 5
	a.worldControls(&platform.Input{Pressed: true, MouseDown: true, MouseX: x + 20, MouseY: py}, w, h)
	a.worldControls(&platform.Input{MouseDown: true, MouseX: x + 220, MouseY: py + 100}, w, h)
	a.worldControls(&platform.Input{Clicked: true, MouseX: x + 220, MouseY: py + 100}, w, h)
	nx, ny := a.partyUI.RaidCompactOrigin(w, h)
	if nx != x+200 || ny != y+100 {
		t.Fatalf("drag: origin %d,%d from %d,%d", nx, ny, x, y)
	}
	b := inputApp()
	b.cfg.RememberLayout = true
	b.partyUI, _ = ui.NewPartyPanel(inputTextures{})
	b.loadUILayout()
	if b.partyUI.RaidCompactPosition != a.partyUI.RaidCompactPosition {
		t.Fatal("raid list position did not persist")
	}
}

func TestRaidViewPersistsAndMoveGroupUsesPlainClicks(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	a := inputApp()
	a.cfg.RememberLayout = true
	a.partyUI, _ = ui.NewPartyPanel(inputTextures{})
	a.state.SelfID = 1
	a.state.Party.Raid, a.state.Party.Leader, a.state.Party.RaidSubLeader = true, 1, 1
	a.state.Party.Members = []proto.PartyMember{{ID: 1}, {ID: 2}}
	a.state.Party.RaidSlots = map[uint16]uint32{0: 1, 1: 2}
	a.party.raidHidden = true
	const w, h = 1024, 768
	// Transparency and options survive.
	a.partyUI.SetRaidViewSettings(.4, true, false)
	a.saveUILayout()
	b := inputApp()
	b.cfg.RememberLayout = true
	b.partyUI, _ = ui.NewPartyPanel(inputTextures{})
	b.loadUILayout()
	if o, c, g := b.partyUI.RaidViewSettings(); o != .4 || !c || g {
		t.Fatalf("raid view not restored: %v %v %v", o, c, g)
	}
	// Move Group completes with a plain click on a cell, empty ones included.
	a.performContextAction(1, contextAction{label: "Move group", slot: 1})
	gx, gy := a.partyUI.RaidGridOrigin(w, h)
	px, py := gx+98*2+5, gy+5 // slot 12: an empty cell of another group
	a.worldControls(&platform.Input{Pressed: true, MouseDown: true, MouseX: px, MouseY: py}, w, h)
	a.worldControls(&platform.Input{Clicked: true, MouseX: px, MouseY: py}, w, h)
	if a.party.moving {
		t.Fatal("the click did not finish Move Group")
	}
}

func TestEmptyRaidGroupTabsStayGrey(t *testing.T) {
	slots := map[uint16]uint32{0: 1, 6: 2}
	if !ui.RaidGroupUsed(slots, 0, 0) || !ui.RaidGroupUsed(slots, 0, 1) || ui.RaidGroupUsed(slots, 0, 2) {
		t.Fatal("group usage")
	}
}

// Five raids of 30: the tabs are always active, empty raids can be viewed and
// Move Group can cross pages.
func TestRaid150PagesTabsAndCrossPageMove(t *testing.T) {
	a := inputApp()
	a.partyUI, _ = ui.NewPartyPanel(inputTextures{})
	a.state.SelfID = 1
	a.state.Party.Raid, a.state.Party.Leader, a.state.Party.RaidSubLeader = true, 1, 1
	a.state.Party.Members = []proto.PartyMember{{ID: 1, Class: 0}, {ID: 2, Class: 3}, {ID: 3, Class: 3}}
	a.state.Party.RaidSlots = map[uint16]uint32{0: 1, 1: 2, 31: 3}
	a.party.raidHidden = true
	const w, h = 1024, 768
	gx, gy := a.partyUI.RaidGridOrigin(w, h)
	// An empty raid (page 5) opens with its tab.
	tabX, tabY := gx+ui.RaidTabX+4*ui.RaidTabStep+8, gy+ui.RaidTabY+15
	a.updatePartyPointer(&platform.Input{Pressed: true, MouseDown: true, MouseX: tabX, MouseY: tabY}, w, h)
	a.updatePartyPointer(&platform.Input{Clicked: true, MouseX: tabX, MouseY: tabY}, w, h)
	if a.partyUI.RaidPage != 4 {
		t.Fatalf("page %d", a.partyUI.RaidPage)
	}
	// A slot on that page is absolute: 4*30 + column 1 row 0.
	if slot := a.partyUI.RaidGridSlotAt(w, h, gx+ui.RaidCellWidth+5, gy+5); slot != 4*30+6 {
		t.Fatalf("slot %d", slot)
	}
	// Move Group armed on page 1 survives the page switch.
	a.partyUI.RaidPage = 0
	a.performContextAction(1, contextAction{label: "Move group", slot: 1})
	a.partyUI.RaidPage = 4
	if !a.party.moving || a.party.moveSlot != 1 {
		t.Fatal("move lost on page change")
	}
}

// Native frames (0x526f30) drag from anywhere inside their rectangle once
// the controls and item cells decline the press.
func TestFramesDragFromEmptyBodyButControlsKeepTheirPress(t *testing.T) {
	a := inputApp()
	a.inventory.open, a.inventory.bag = true, 1
	x, y := a.inventoryUI.Origin(1024, 768)
	if !a.updateWindowDrag(&platform.Input{MouseX: x + 100, MouseY: y + 270 - 100, Pressed: true, MouseDown: true}, 1024, 768) {
		t.Fatal("empty equipment body did not start a drag")
	}
	a.windowDrag = windowDrag{}
	if a.updateWindowDrag(&platform.Input{MouseX: x + 15 + 2, MouseY: y + 283 + 2, Pressed: true, MouseDown: true}, 1024, 768) {
		t.Fatal("bag cell started a drag")
	}
	if a.updateWindowDrag(&platform.Input{MouseX: x + 20, MouseY: y + 255, Pressed: true, MouseDown: true}, 1024, 768) {
		t.Fatal("bag tab started a drag")
	}
}

func TestStatusDragsFromBodyOnlyOutsideControlsAndExpandedGraph(t *testing.T) {
	a := inputApp()
	a.status.open = true
	x, y := a.statusUI.Origin(1024, 768)
	if !a.updateWindowDrag(&platform.Input{MouseX: x + 150, MouseY: y + 420, Pressed: true, MouseDown: true}, 1024, 768) {
		t.Fatal("status body did not drag")
	}
	a.windowDrag = windowDrag{}
	if a.updateWindowDrag(&platform.Input{MouseX: x + 3, MouseY: y + 160, Pressed: true, MouseDown: true}, 1024, 768) {
		t.Fatal("expand tab started a drag")
	}
	a.statusUI.Expanded = true
	x, y = a.statusUI.Origin(1024, 768)
	if a.updateWindowDrag(&platform.Input{MouseX: x + 50, MouseY: y + 200, Pressed: true, MouseDown: true}, 1024, 768) {
		t.Fatal("expanded graph panel must swallow the press like 0x5037f0")
	}
	if !a.updateWindowDrag(&platform.Input{MouseX: x + 50, MouseY: y + 400, Pressed: true, MouseDown: true}, 1024, 768) {
		t.Fatal("expanded sheet body did not drag")
	}
}

func TestServiceWindowsDragFromBodyButKeepListsAndButtons(t *testing.T) {
	a := inputApp()
	a.npcTalkUI, _ = ui.NewNPCTalk(inputTextures{})
	a.shop.npcID, a.shop.talk = 42, true
	a.state.Entities[42] = &game.Entity{Kind: game.KindNpc, NpcType: 3}
	x, y := a.npcTalkUI.Origin(1024, 768)
	a.worldControls(&platform.Input{MouseX: 1, MouseY: 1}, 1024, 768)
	if a.updateWindowDrag(&platform.Input{MouseX: x + 100, MouseY: y + 150, Pressed: true, MouseDown: true}, 1024, 768) {
		t.Fatal("text area started a drag")
	}
	if a.updateWindowDrag(&platform.Input{MouseX: x + 63, MouseY: y + 344, Pressed: true, MouseDown: true}, 1024, 768) {
		t.Fatal("service button started a drag")
	}
	if !a.updateWindowDrag(&platform.Input{MouseX: x + 120, MouseY: y + 60, Pressed: true, MouseDown: true}, 1024, 768) {
		t.Fatal("header body did not drag")
	}
}

func TestInventoryDragWithMerchantOpen(t *testing.T) {
	const w, h = 1024, 768
	a := inputApp()
	a.inventory.open = true
	a.inventoryUI.Position.Move(670, 100, w, h, ui.InventoryWidth, ui.InventoryHeight)
	saved := a.inventoryUI.Position
	a.shop.npcID = 42
	a.state.Entities[42] = &game.Entity{Kind: game.KindNpc, NpcType: 1}

	drive := func(x, y int, press, down, release bool) platform.Input {
		return a.worldControls(&platform.Input{MouseX: x, MouseY: y, Pressed: press, MouseDown: down, Clicked: release}, w, h)
	}
	drag := func(x, y, dx, dy int) {
		t.Helper()
		if in := drive(x, y, true, true, false); in.Pressed || in.MouseDown {
			t.Fatal("frame press leaked to the world")
		}
		drive(x+dx, y+dy, false, true, false)
		if in := drive(x+dx, y+dy, false, false, true); in.Clicked || a.windowDrag.position != nil {
			t.Fatal("frame release leaked to the world")
		}
	}
	ix, iy := a.inventoryUI.Origin(w, h)
	mx, my := a.marketUI.Origin(w, h)
	if ix+ui.InventoryWidth != mx || iy != my {
		t.Fatal("inventory did not start beside the merchant")
	}
	drive(ix+50, iy+15, true, true, false)
	drive(ix+50, iy+15, false, false, true)
	drag(mx+50, my+15, 100, 24)
	ix, iy = a.inventoryUI.Origin(w, h)
	mx, my = a.marketUI.Origin(w, h)
	if ix+ui.InventoryWidth != mx || iy != my {
		t.Fatal("clicking the inventory frame broke its default docking")
	}
	drag(ix+50, iy+15, 32, 100)
	if x, y := a.inventoryUI.Origin(w, h); x != ix+32 || y != iy+100 {
		t.Fatalf("inventory failed to move independently while trading: got %d,%d", x, y)
	}
	if x, y := a.marketUI.Origin(w, h); x != mx || y != my {
		t.Fatal("moving inventory also moved the merchant")
	}
	drag(mx+50, my+15, 90, 24)
	if x, y := a.marketUI.Origin(w, h); x != mx+90 || y != my+24 {
		t.Fatal("merchant frame stopped dragging")
	}
	if x, y := a.inventoryUI.Origin(w, h); x != ix+32 || y != iy+100 {
		t.Fatal("the moved inventory followed the merchant")
	}

	drive(ix+82, iy+115, true, true, false)
	a.shop.npcID = 0
	if in := drive(ix+82, iy+115, false, false, true); in.Clicked || a.windowDrag.position != nil {
		t.Fatal("closing the merchant during a drag leaked the release")
	}
	if x, y := a.inventoryUI.Origin(w, h); x != 670 || y != 100 || a.inventoryUI.Position != saved {
		t.Fatal("temporary merchant placement replaced the saved inventory position")
	}
}
