package app

import (
	"os"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

func TestLocalActionsPersistPerCharacterAndServer(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	a := inputApp()
	a.cfg.LoginAddr = "localhost"
	a.cfg.WorldPort = 30810
	a.state.SelfID = 42
	a.wv.localActions[4] = 11
	a.saveLocalActions()
	a.wv.localActions = [ui.TotalActionSlots]uint8{}
	a.loadLocalActions()
	if got, ok := a.localAction(4); !ok || got != 10 {
		t.Fatalf("loaded action=%v %v", got, ok)
	}
	a.state.QuickSlots = []proto.QuickSlot{{Slot: 4, Bag: 100, Number: 1}}
	if _, ok := a.localAction(4); ok {
		t.Fatal("local shortcut overwrote server slot")
	}
	a.state.SkillBarReady = true
	a.reconcileLocalActions()
	a.state.QuickSlots = nil
	if _, ok := a.localAction(4); ok {
		t.Fatal("conflicting shortcut returned after server slot cleared")
	}
	a.wv.localActions[4] = 11
	a.saveLocalActions()
	a.state.SelfID = 43
	a.wv.localActions = [ui.TotalActionSlots]uint8{}
	a.loadLocalActions()
	if _, ok := a.localAction(4); ok {
		t.Fatal("another character inherited shortcut")
	}
	a.state.SelfID = 42
	a.cfg.LoginAddr = "other"
	a.loadLocalActions()
	if _, ok := a.localAction(4); ok {
		t.Fatal("another server inherited shortcut")
	}
	path, err := localActionsPath()
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatalf("shortcut file: %v %v", info, err)
	}
}
func TestActionDragOwnsReleaseOutsidePanel(t *testing.T) {
	a := inputApp()
	a.subActionsOpen = true
	x, y := a.subActionsUI.Origin(1024, 734)
	cx, cy := ui.SubActionCell(10)
	x += cx + 2
	y += cy + 2
	a.worldControls(&platform.Input{MouseX: x, MouseY: y, Pressed: true, MouseDown: true}, 1024, 734)
	if !a.wv.localActionDrag.tracking {
		t.Fatal("press did not begin tracking")
	}
	controls := a.worldControls(&platform.Input{MouseX: x - 100, MouseY: y + 200, MouseDown: true}, 1024, 734)
	if !a.wv.localActionDrag.active || controls.MouseDown {
		t.Fatal("drag did not own pointer")
	}
	controls = a.worldControls(&platform.Input{MouseX: 20, MouseY: 500, Clicked: true}, 1024, 734)
	if controls.Clicked || a.wv.localActionDrag.tracking {
		t.Fatal("release leaked or drag retained")
	}
}
