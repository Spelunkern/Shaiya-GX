package app

import (
	"fmt"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/data/model"
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	gametext "github.com/Spelunkern/Shaiya-GX/internal/data/text"
	"github.com/Spelunkern/Shaiya-GX/internal/data/world"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

func TestNPCPatrolUsesAuthoredStepTime(t *testing.T) {
	var r remotePosition
	e := &game.Entity{X: 10, Y: 40, Z: 10}
	def := sdata.NPC{MoveDistance: 10, MoveSpeed: 5000}
	r.updateNPC(e, 0, def)
	e.X = 20
	e.MoveSerial++
	r.updateNPC(e, 1, def)
	p := r.updateNPC(e, 3.5, def)
	if p[0] != 15 {
		t.Fatal("patrol compressed into player sample interval", p)
	}
	p = r.updateNPC(e, 6, def)
	if p[0] != 20 {
		t.Fatal("patrol never arrived", p)
	}
	e.X = 200
	e.PositionResetSerial++
	if p = r.updateNPC(e, 7, def); p[0] != 200 {
		t.Fatal("reset did not snap", p)
	}
}
func TestQuickPotionsBindPersistAndFollowInventory(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	a := inputApp()
	a.state.SelfID = 78
	a.ensureQuickPotions()
	a.quickPotions.Position.Move(390, 220, 1024, 768, 94, 54)
	a.inventory.open = true
	a.inventory.source = [2]uint8{1, 5}
	a.inventory.dragging, a.inventory.selected = true, true
	a.state.Inventory[a.inventory.source] = proto.InventoryItem{Type: 25, TypeID: 4, Count: 20}
	x, y := ui.QuickPotionOrigin(a.quickPotions.Position, 1024, 768)
	if !a.updateQuickPotionDrop(&platform.Input{MouseX: x + 15, MouseY: y + 15, Clicked: true}, 1024, 768) {
		t.Fatal("drop was not captured")
	}
	b := inputApp()
	b.state.SelfID = 78
	b.ensureQuickPotions()
	b.state.Inventory[[2]uint8{2, 6}] = a.state.Inventory[a.inventory.source]
	key, it, ok := b.quickPotionItem(0)
	if !ok || key != [2]uint8{2, 6} || it.Count != 20 {
		t.Fatal("binding did not survive relocation/login", key, it, ok)
	}
	b.state.SelfID = 79
	b.ensureQuickPotions()
	if b.quickPotions.bindings != [2]sdata.ItemKey{} {
		t.Fatal("bindings leaked between characters")
	}
}
func TestNPCQuestAndPortraitDragOwnRelease(t *testing.T) {
	a := inputApp()
	a.npcQuestUI, _ = ui.NewNPCQuest(inputTextures{})
	a.hud, _ = ui.NewHUD(inputTextures{})
	a.shop.npcID = 42
	a.state.Entities[42] = &game.Entity{Kind: game.KindNpc, NpcType: 4}
	x, y := a.npcQuestUI.Origin(1024, 768)
	a.worldControls(&platform.Input{MouseX: x + 100, MouseY: y + 12, Pressed: true, MouseDown: true}, 1024, 768)
	a.worldControls(&platform.Input{MouseX: x + 200, MouseY: y + 100, Clicked: true}, 1024, 768)
	if !a.npcQuestUI.Position.Moved {
		t.Fatal("dialogue did not move")
	}
	x, y = a.hud.Origin(1024, 768)
	a.state.SelfID = 7
	a.worldControls(&platform.Input{MouseX: x + 40, MouseY: y + 50, Pressed: true, MouseDown: true}, 1024, 768)
	a.worldControls(&platform.Input{MouseX: x + 40, MouseY: y + 50, Clicked: true}, 1024, 768)
	if a.state.Target != 7 || a.hud.Position.Moved {
		t.Fatal("portrait click became a drag")
	}
	a.worldControls(&platform.Input{MouseX: x + 40, MouseY: y + 50, Pressed: true, MouseDown: true}, 1024, 768)
	a.worldControls(&platform.Input{MouseX: x + 140, MouseY: y + 150, Clicked: true}, 1024, 768)
	if !a.hud.Position.Moved {
		t.Fatal("portrait did not drag")
	}
}
func TestInteractionApproachesBesideAllNPCsAndLoot(t *testing.T) {
	a := inputApp()
	for kind := uint8(1); kind <= 13; kind++ {
		a.state.Entities[42] = &game.Entity{Kind: game.KindNpc, NpcType: kind, X: 8}
		a.wv.approachTarget, a.wv.approachNPC = 42, true
		a.updateActionApproach(&platform.Input{})
		if !a.wv.travel || a.wv.destination[0] < 6 || a.wv.destination[0] > 7 {
			t.Fatal("NPC did not approach beside actor", kind, a.wv.destination)
		}
	}
	a.state.GroundItems[3] = proto.GroundItem{ID: 3, X: 8}
	a.approachLoot(3)
	if !a.wv.travel || a.wv.destination[0] < 6 {
		t.Fatal("loot collected at old ten-unit reach")
	}
	a.updateActionApproach(&platform.Input{Clicked: true})
	if a.wv.approachTarget != 0 {
		t.Fatal("new world click did not cancel approach")
	}
}
func TestRealPrayerLabelsAndGestureSemantics(t *testing.T) {
	dir := os.Getenv("SHAIYA_DATA")
	if dir == "" {
		t.Skip("requires assets")
	}
	arc, err := archive.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	read := func(name string) []byte {
		t.Helper()
		b, err := arc.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	defs, err := sdata.ParseSkills(read("Character/Skill.SData"))
	if err != nil {
		t.Fatal(err)
	}
	d := defs[sdata.SkillKey{ID: 123, Level: 1}]
	if !strings.HasPrefix(d.Name, "Prayer") || d.Target != 4 {
		t.Fatalf("unexpected Prayer fixture %+v", d)
	}
	a := inputApp()
	a.state.SelfID = 1
	a.skillDefs = defs
	a.clock = 1
	a.playSkillSound(1, 123, 1, false)
	key := skillSoundKey{1, 123, 1, false}
	if a.skillSoundEvents[key] != 1 {
		t.Fatal("Prayer group release was suppressed")
	}
	a.clock = 1.1
	a.playSkillSound(1, 123, 1, false)
	if a.skillSoundEvents[key] != 1 {
		t.Fatal("multi-target release was repeated")
	}
	t.Logf("Prayer: target=%d effect=%d animation=%d", d.Target, d.Effect, d.Animation)
	w, err := world.ParseWLD(read("world/1.wld"))
	if err != nil {
		t.Fatal(err)
	}
	texts := gametext.ParseSysMsg(read("world/1.txt"))
	hover, banners := 0, 0
	for _, area := range w.NamedAreas {
		if area.Mode == 0 {
			hover++
			if _, ok := texts[area.TextID]; !ok {
				t.Fatal("missing hover text ID", area.TextID)
			}
		}
		if area.Mode == 2 {
			banners++
			name := path.Base(strings.ReplaceAll(area.Text, `\`, "/"))
			name = strings.TrimSuffix(name, path.Ext(name)) + ".png"
			read("interface/worldname/" + name)
		}
	}
	if hover == 0 || banners == 0 {
		t.Fatal("named area modes were lost", hover, banners)
	}
	t.Logf("map1: %d hover labels, %d original bitmap banners", hover, banners)
	actions, err := model.ParseALT(read("Character/Vile/viwm_action.alt"))
	if err != nil {
		t.Fatal(err)
	}
	for i, clip := range []int{2, 1, 4, 3, 6, 5, 8, 7, 9, 10} {
		slot, _ := proto.GestureAction(uint8(116 + i))
		if !strings.Contains(strings.ToLower(actions[slot].Name), fmt.Sprintf("emoti%03d", clip)) {
			t.Fatal("subaction clip differs from Phoenix", i, slot, actions[slot].Name)
		}
	}
}

func TestPotionBindingRequiresExplicitDragOutToRemove(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	a := inputApp()
	a.state.SelfID = 78
	a.ensureQuickPotions()
	a.quickPotions.Position.Move(390, 220, 1024, 768, 94, 54)
	a.quickPotions.bindings[0] = sdata.ItemKey{Type: 25, TypeID: 4}
	x, y := ui.QuickPotionOrigin(a.quickPotions.Position, 1024, 768)
	binding := a.quickPotions.bindings[0]
	a.updateQuickPotions(&platform.Input{MouseX: x + 15, MouseY: y + 15, RightPressed: true}, 1024, 768)
	a.updateQuickPotions(&platform.Input{MouseX: x + 15, MouseY: y + 15, Pressed: true, MouseDown: true}, 1024, 768)
	a.updateQuickPotionDrop(&platform.Input{MouseX: x + 15, MouseY: y + 15, Clicked: true}, 1024, 768)
	a.updateQuickPotions(&platform.Input{MouseX: x + 15, MouseY: y + 15, Clicked: true}, 1024, 768)
	if a.quickPotions.bindings[0] != binding {
		t.Fatal("click removed binding")
	}
	a.updateQuickPotions(&platform.Input{MouseX: x + 15, MouseY: y + 15, Pressed: true, MouseDown: true}, 1024, 768)
	a.updateQuickPotionDrop(&platform.Input{MouseX: x + 150, MouseY: y + 150, MouseDown: true}, 1024, 768)
	a.updateQuickPotionDrop(&platform.Input{MouseX: x + 150, MouseY: y + 150, Clicked: true}, 1024, 768)
	if a.quickPotions.bindings[0] != (sdata.ItemKey{}) {
		t.Fatal("explicit drag out did not remove binding")
	}
}
