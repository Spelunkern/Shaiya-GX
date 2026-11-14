//go:build integration

package app

import (
	"fmt"
	"github.com/Spelunkern/Shaiya-GX/internal/audio"
	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
	"github.com/go-gl/gl/v4.5-core/gl"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestForgeDetailsOriginalAssetsGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL and original assets")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Title: "Forge details", Width: 1024, Height: 768, Hidden: true, Debug: true})
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
	for _, name := range []string{"ui_rapis001.ogg", "ui_rapis002.ogg"} {
		raw, e := arc.ReadFile("Sound/" + name)
		if e != nil {
			t.Fatal(e)
		}
		if pcm, e := audio.DecodeOGG(raw); e != nil || len(pcm) == 0 {
			t.Fatal("native forge sound unavailable", name, e)
		}
	}
	a.state = game.New()
	a.shop.npcID = 42
	a.state.Entities[42] = &game.Entity{ID: 42, Kind: game.KindNpc, NpcType: 3}
	a.blacksmith.npcID = 42
	a.blacksmith.keys = [4][2]uint8{{1, 0}, {1, 1}, {1, 2}, {1, 3}}
	items := []proto.InventoryItem{{Bag: 1, Slot: 0, Type: 16, TypeID: 216, Count: 1, Craft: "01020300000001000003", Gems: [6]uint8{1, 2}}, {Bag: 1, Slot: 1, Type: 30, TypeID: 3, Count: 1}, {Bag: 1, Slot: 2, Type: 25, TypeID: 153, Count: 255}, {Bag: 1, Slot: 3, Type: 100, TypeID: 49, Count: 255}}
	for i, it := range items {
		a.state.Inventory[a.blacksmith.keys[i]] = it
		a.blacksmith.items[i] = it
	}
	out := os.Getenv("SHAIYA_TEST_ARTIFACTS")
	if out == "" {
		out = t.TempDir()
	}
	os.MkdirAll(out, 0755)
	capture := func(name string, hoverSlot int) {
		t.Helper()
		a.tooltip = nil
		x, y := a.blacksmithUI.Origin(1024, 768)
		sx, sy := a.blacksmithUI.Slot(hoverSlot)
		a.lastMouseX, a.lastMouseY = x+sx+2, y+sy+2
		gl.Viewport(0, 0, 1024, 768)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		a.sprites.Begin(1024, 768)
		a.drawBlacksmith(1024, 768, "NPC NAME MUST NOT RENDER")
		a.drawRecreationDetails(1024, 768)
		a.drawSharedTooltip(1024, 768)
		a.sprites.End()
		if len(a.tooltip) == 0 && a.blacksmithUI.Mode != ui.ForgeRemake {
			t.Fatal("missing hover tooltip", name)
		}
		if e := gl.GetError(); e != gl.NO_ERROR {
			t.Fatalf("GL error %x", e)
		}
		if err := render.SavePNG(filepath.Join(out, name+".png"), 1024, 768); err != nil {
			t.Fatal(err)
		}
	}
	a.blacksmith.selected = [4]bool{true, true, false, true}
	a.blacksmith.quote = forgeQuoteView{known: true, value: proto.ForgeQuote{Valid: true, Rate: 450000, Gold: 22965}}
	capture("lapis-equipment-hover", 0)
	capture("lapis-new-gem-hover", 1)
	if !strings.Contains(tooltipRowsText(a.tooltip), a.msg.Get(5181)) || a.msg.Get(5181) == "" {
		t.Fatal("lapis native target heading missing")
	}
	capture("lapis-hammer-hover", 3)
	// Native SysMsg confirmation must display all original lines, rate and gold.
	a.gemConfirm.OpenFull(a.forgeConfirmation(proto.ForgeQuote{Kind: proto.ForgeQuoteLink, Valid: true, Rate: 450000, Gold: 22965}), true, a.font)
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	a.sprites.Begin(1024, 768)
	a.drawBlacksmith(1024, 768, "")
	a.gemConfirm.Draw(a.sprites, a.font, 1024, 768)
	a.sprites.End()
	render.SavePNG(filepath.Join(out, "lapis-native-confirm.png"), 1024, 768)
	a.gemConfirm.Visible = false
	a.blacksmithUI.Mode = ui.ForgeEnchant
	a.blacksmith.selected = [4]bool{true}
	a.blacksmith.quote = forgeQuoteView{known: true, value: proto.ForgeQuote{Valid: true, Kind: proto.ForgeQuoteEnchant, Rate: 800000, Lapis: 6, Material: 25153}}
	if _, ok := a.blacksmithGhosts()[1]; ok {
		t.Fatal("unexpected lapisia ghost")
	}
	capture("enchant-material-ghost-hover", 2)
	capture("enchant-equipment-hover", 0)
	a.blacksmith.items[1] = proto.InventoryItem{Bag: 1, Slot: 1, Type: 95, TypeID: 6, Count: 255}
	a.state.Inventory[a.blacksmith.keys[1]] = a.blacksmith.items[1]
	a.blacksmith.selected[1] = true
	capture("lapisia-compatible-targets", 1)
	if !strings.Contains(tooltipRowsText(a.tooltip), a.msg.Get(5181)) {
		t.Fatal("lapisia native target heading missing")
	}
	a.blacksmithUI.Mode = ui.ForgeRemake
	a.state.Inventory[a.blacksmith.keys[2]] = proto.InventoryItem{Bag: 1, Slot: 2, Type: 100, TypeID: 171, Count: 255}
	a.blacksmith.items[2] = a.state.Inventory[a.blacksmith.keys[2]]
	a.blacksmith.selected = [4]bool{true, false, true}
	capture("recreate-before", 0)
	a.forgeResult(true)
	a.clock += .25
	capture("recreate-success-glow", 0)
	a.forgeResult(false)
	if a.blacksmith.shineUntil != 0 {
		t.Fatal("failure retained success glow")
	}
	capture("recreate-failure", 0)
	previous := strings.Builder{}
	it := a.state.Inventory[a.blacksmith.keys[0]]
	d := a.itemsItem(it.Type, it.TypeID)
	a.drawItemDetails(d, &it, 1, 0, 0, 1024, 768, "")
	if a.tooltipExtra.maxOJ != 13 {
		t.Fatal("native Max OJ not exposed", a.tooltipExtra)
	}
	layout := layoutTooltip(a.tooltipFont, a.tooltip, true, a.tooltipDescription, len(a.tooltipSockets), a.tooltipExtra)
	foundOJ := false
	for _, run := range layout.runs {
		if strings.Contains(run.text, "Max OJ 13/1300") {
			foundOJ = true
			if run.color != ([4]float32{.6, 1, .6, 1}) {
				t.Fatal("OJ color incorrect")
			}
		}
	}
	if !foundOJ || layout.materialY == 0 {
		t.Fatal("OJ/material footer missing")
	}

	for _, r := range a.tooltip {
		previous.WriteString(r.text)
	}
	it.Craft = "09080706050403020103"
	a.state.Inventory[a.blacksmith.keys[0]] = it
	a.syncBlacksmithItems()
	capture("recreate-after", 0)
	a.drawItemDetails(d, &it, 1, 0, 0, 1024, 768, "")
	var next strings.Builder
	for _, r := range a.tooltip {
		next.WriteString(r.text)
	}
	if previous.String() == next.String() {
		t.Fatal("authoritative recreation stats did not refresh tooltip")
	}
	// Real native gem definitions must change the equipment tooltip itself.
	var fire, water uint8
	for key, def := range a.items {
		if key.Type == 30 && def.Attribute == 5 {
			fire = key.TypeID
		}
		if key.Type == 30 && def.Attribute == 2 {
			water = key.TypeID
		}
	}
	if fire == 0 || water == 0 {
		t.Fatal("native elemental lapis missing")
	}
	it = proto.InventoryItem{Bag: 1, Slot: 0, Type: 1, TypeID: 2, Count: 1, Gems: [6]uint8{fire}}
	a.state.Inventory[a.blacksmith.keys[0]] = it
	a.blacksmith.items[0] = it
	capture("elemental-linked-weapon", 0)
	a.drawItemDetails(a.itemsItem(1, 2), &it, 1, 0, 0, 1024, 768, "")
	if !strings.Contains(tooltipRowsText(a.tooltip), elementName(5, a.msg)) {
		t.Fatal("effective linked element not visible")
	}
	it.Gems[1] = water
	a.state.Inventory[a.blacksmith.keys[0]] = it
	a.blacksmith.items[0] = it
	capture("elemental-first-lapis-priority", 0)
	a.drawItemDetails(a.itemsItem(1, 2), &it, 1, 0, 0, 1024, 768, "")
	if !strings.Contains(tooltipRowsText(a.tooltip), elementName(5, a.msg)) || !a.tooltipSockets[1].inactive {
		t.Fatal("first lapis priority / inactive marker missing")
	}
	original := a.itemsItem(1, 2)
	intrinsic := *original
	intrinsic.Attribute = 3
	a.items[sdata.ItemKey{Type: 1, TypeID: 2}] = &intrinsic
	capture("elemental-intrinsic-priority", 0)
	a.drawItemDetails(&intrinsic, &it, 1, 0, 0, 1024, 768, "")
	if !a.tooltipSockets[0].inactive || !a.tooltipSockets[1].inactive || !strings.Contains(tooltipRowsText(a.tooltip), elementName(3, a.msg)) {
		t.Fatal("intrinsic element must gray every linked elemental gem")
	}
	intrinsic.Slot = 6
	a.blacksmithUI.Mode = ui.ForgeLapis
	a.blacksmith.selected = [4]bool{true, false, false, false}
	capture("pre-element-ignored-lapis-forge", 0)
	intrinsic.Attribute = 0
	capture("first-element-active-lapis-forge", 0)
	a.items[sdata.ItemKey{Type: 1, TypeID: 2}] = original
	a.blacksmith.items[3] = proto.InventoryItem{Bag: 1, Slot: 3, Type: 100, TypeID: 141, Count: 255}
	a.state.Inventory[a.blacksmith.keys[3]] = a.blacksmith.items[3]
	a.blacksmith.selected[3] = true
	capture("extraction-hammer-client-rate", 3)
	a.loadSkills()
	for attr := uint8(1); attr <= 4; attr++ {
		var key sdata.SkillKey
		for k, d := range a.skillDefs {
			if k.Level == 1 && d.Attribute == attr && d.Effect == 1 && (key.ID == 0 || k.ID < key.ID) {
				key = k
			}
		}
		if key.ID == 0 {
			t.Fatal("native elemental spell missing", attr)
		}
		a.tooltip = nil
		a.queueSkillTooltip(key)
		a.lastMouseX, a.lastMouseY = 450, 200
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		a.sprites.Begin(1024, 768)
		a.drawSharedTooltip(1024, 768)
		a.sprites.End()
		if err := render.SavePNG(filepath.Join(out, fmt.Sprintf("spell-element-color-%d.png", attr)), 1024, 768); err != nil {
			t.Fatal(err)
		}
	}

}

func tooltipRowsText(rows []itemDetailLine) string {
	var out strings.Builder
	for _, row := range rows {
		out.WriteString(row.text)
		out.WriteByte('\n')
	}
	return out.String()
}
