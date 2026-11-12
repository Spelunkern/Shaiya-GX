//go:build integration

package app

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
	"github.com/go-gl/gl/v4.5-core/gl"
)

func TestHUDPresentationGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL and original assets")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Title: "HUD composition probe", Width: 1024, Height: 768, Hidden: true, Debug: true})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
	if err := render.Init(true); err != nil {
		t.Fatal(err)
	}
	arc, err := archive.Open(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	a, err := New(Config{}, arc)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	a.state, a.wv = game.New(), &worldView{}
	a.state.SelfID = 1
	a.state.Slots[0] = proto.CharacterSlot{ID: 1, Name: "Priest", Level: 60, Race: 0, Class: 5, Gender: 0, Height: 2, Face: 1, Hair: 1}
	dir := os.Getenv("SHAIYA_TEST_ARTIFACTS")
	if dir == "" {
		dir = t.TempDir()
	}
	draw := func(file string) {
		t.Helper()
		gl.Viewport(0, 0, 1024, 768)
		gl.ClearColor(.2, .3, .4, 1)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		a.sprites.Begin(1024, 768)
		portrait := a.hudPortrait(1024, 768)
		if portrait == nil {
			t.Fatal("missing portrait")
		}
		a.hud.Draw(a.sprites, a.font, ui.HUDState{Name: "Priest", Level: 60, HP: ui.Resource{Current: 1234, Max: 4321}, MP: ui.Resource{Current: 225, Max: 600}, SP: ui.Resource{Current: 120, Max: 500}, Portrait: portrait})
		// Show the same snapshot large as well as in the final HUD ellipse.
		a.sprites.DrawPortrait(portrait, 260, 15, 220, 212)
		a.sprites.End()
		if err := render.SavePNG(file, 1024, 768); err != nil {
			t.Fatal(err)
		}
	}
	first := filepath.Join(dir, "hud-portrait-composition.png")
	draw(first)
	before, err := os.ReadFile(first)
	if err != nil {
		t.Fatal(err)
	}
	// Re-capture after hostile world state. Inventory and portrait must both
	// retain normal materials, rather than a clipped shadow silhouette.
	a.wv.portrait.Delete()
	a.wv.portrait = nil
	a.wv.portraitLoaded = false
	a.scene3d.Shadow = true
	a.scene3d.ClipPlane = [4]float32{0, -1, 0, 0}
	second := filepath.Join(dir, "hud-portrait-isolated.png")
	draw(second)
	after, err := os.ReadFile(second)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("world state changed portrait materials/composition", err)
	}
	a.scene3d.Shadow = false
	a.scene3d.ClipPlane = [4]float32{}
	// Exercise the original priest's model-20 armor, using a local appearance
	// fixture to select its MLT records independently of server equipment IDs.
	a.items = map[sdata.ItemKey]*sdata.Item{}
	own := a.state.Slots[0]
	for slot := 1; slot <= 4; slot++ {
		typ := uint8(200 + slot)
		a.items[sdata.ItemKey{Type: typ, TypeID: 1}] = &sdata.Item{Type: typ, TypeID: 1, Model: 20}
		own.EquipType[slot], own.EquipTypeID[slot] = typ, 1
	}
	a.state.Slots[0] = own
	draw(filepath.Join(dir, "hud-portrait-equipped.png"))
	a.inventory.open = true
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	a.sprites.Begin(1024, 768)
	a.drawInventoryPreview(1024, 768)
	a.sprites.End()
	if a.inventory.previewAppearance != a.wv.portraitAppearance {
		t.Fatal("inventory and portrait resolved different equipment")
	}
	if err := render.SavePNG(filepath.Join(dir, "hud-inventory-equipped.png"), 1024, 768); err != nil {
		t.Fatal(err)
	}
	a.inventory.open = false

	a.loadSkills()
	a.skills.open = true
	a.state.Details.SkillPoint = 100
	var k sdata.SkillKey
	for _, key := range a.availableSkills() {
		if a.skillDefs[key].Effect != 0 {
			k = key
			break
		}
	}
	if k.ID == 0 {
		t.Fatal("no active skill definition")
	}
	a.skills.category = a.skillDefs[k].Show - 1
	a.state.Skills[1] = proto.LearnedSkill{Number: 1, ID: k.ID, Level: k.Level}
	a.state.SkillBarReady = true
	a.state.QuickSlots = []proto.QuickSlot{{Bar: 0, Slot: 0, Bag: 100, Number: 1}}
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	a.sprites.Begin(1024, 768)
	a.drawSkills(1024, 768)
	x, y := a.actionBar.SlotOrigin(0, 1024, 768)
	a.lastMouseX, a.lastMouseY = x+5, y+5
	a.queueActionTooltip(1024, 768)
	a.drawSharedTooltip(1024, 768)
	a.sprites.End()
	if err := render.SavePNG(filepath.Join(dir, "hud-skills-tooltip.png"), 1024, 768); err != nil {
		t.Fatal(err)
	}
	a.tooltip = nil
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	a.sprites.Begin(1024, 768)
	item := &sdata.Item{Name: "Equipment", Description: "{c3}A coloured description{/c}\nA complete second line which wraps inside the shared Phoenix tooltip.", Attack: 50, AttackAdd: 12, MinLevel: 20}
	instance := &proto.InventoryItem{Count: 2, Quality: 35, Craft: "01020304050607080953", Gems: [6]uint8{1}}
	a.lastMouseX, a.lastMouseY = 10, 740
	a.drawItemDetails(item, instance, 2, 0, 0, 1024, 768, "Sell: 100 gold")
	a.drawSharedTooltip(1024, 768)
	a.sprites.End()
	if err := render.SavePNG(filepath.Join(dir, "hud-item-tooltip.png"), 1024, 768); err != nil {
		t.Fatal(err)
	}
	// Same explicit documents rendered by Phoenix's actual tooltip scripts in
	// the isolated reference harness. Compare measured blocks and final cards.
	a.lastMouseX, a.lastMouseY = 400, 100
	a.tooltip = []itemDetailLine{{"Magic Arrow Lv1", [4]float32{1, 1, 0, 1}}, {"SP: 15", white}, {"MP: 20", white}, {"Cast time: 1.5 seconds", white}, {"Recharge time: 12 seconds", white}, {"Range: 27 m", white}}
	a.tooltipItem = false
	a.tooltipSockets = nil
	a.tooltipDescription = "Launches a non-elemental magic arrow that causes 39 extra damage plus your Magic Power."
	skillLayout := layoutTooltip(a.tooltipFont, a.tooltip, false, a.tooltipDescription, 0)
	t.Logf("reference font height=%d ascent=%d skill height=%d", a.tooltipFont.Height, a.tooltipFont.Ascent, skillLayout.height)
	if skillLayout.height != 186 {
		t.Fatal("skill tooltip height differs from Phoenix", skillLayout.height)
	}
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	a.sprites.Begin(1024, 768)
	a.drawSharedTooltip(1024, 768)
	a.sprites.End()
	if err := render.SavePNG(filepath.Join(dir, "go-skill-tooltip-reference.png"), 1024, 768); err != nil {
		t.Fatal(err)
	}
	a.tooltip = []itemDetailLine{{"Templar Armor +3", white}, {"Defense: 50", white}, {"HP +100  {r}(+200){/r}", [4]float32{1, 1, 0, 1}}, {"STR +3", [4]float32{1, 135.0 / 255, 0, 1}}, {"Required level: 20", [4]float32{1, 0, 0, 1}}, {"Durability: 35 / 100", white}, {"Sell price: 100 Gold", white}}
	a.tooltipItem = true
	a.tooltipSockets = []tooltipSocket{{}, {}}
	a.tooltipDescription = "{c3}A coloured description{/c}\nA complete second line which wraps inside the tooltip."
	itemLayout := layoutTooltip(a.tooltipFont, a.tooltip, true, a.tooltipDescription, 2)
	t.Logf("item height=%d socket_y=%d", itemLayout.height, itemLayout.socketY)
	if itemLayout.height != 216 || itemLayout.socketY != 185 {
		t.Fatal("item tooltip blocks differ from Phoenix", itemLayout.height, itemLayout.socketY)
	}
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	a.sprites.Begin(1024, 768)
	a.drawSharedTooltip(1024, 768)
	a.sprites.End()
	if err := render.SavePNG(filepath.Join(dir, "go-item-tooltip-reference.png"), 1024, 768); err != nil {
		t.Fatal(err)
	}

	if err := gl.GetError(); err != gl.NO_ERROR {
		t.Fatalf("HUD GL error %#x", err)
	}
}
