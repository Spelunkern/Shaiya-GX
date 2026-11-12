package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
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
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	a := inputApp()
	a.cfg.RememberLayout = true
	a.inventoryUI.Position.Move(200, 100, 1024, 768, ui.InventoryWidth, ui.InventoryHeight)
	a.subActionsUI.Position.Move(500, 80, 1024, 768, ui.SubActionWidth, ui.SubActionHeight)
	a.saveUILayout()
	b := inputApp()
	b.cfg.RememberLayout = true
	b.loadUILayout()
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
	if b.inventoryUI.Position.Moved {
		t.Fatal("invalid saved layout accepted")
	}
}
