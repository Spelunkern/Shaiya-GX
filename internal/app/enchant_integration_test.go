//go:build integration

package app

import (
	"os"
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/testworld"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

func TestEnchantAgainstRapture(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_ENCHANT") != "1" {
		t.Skip("requires disposable enchant fixtures")
	}
	st, world, _, wait := testworld.Enter(t)
	a := inputApp()
	a.state, a.world = st, world
	a.blacksmithUI, _ = ui.NewBlacksmith(inputTextures{})
	a.gemConfirm, _ = ui.NewMessage(inputTextures{})
	a.npcQuestUI, _ = ui.NewNPCQuest(inputTextures{})
	a.wv.pos[0], a.wv.pos[1], a.wv.pos[2] = st.Self.X, st.Self.Y, st.Self.Z
	var smith uint32
	wait(func() bool {
		for id, e := range st.Entities {
			if e.Kind == game.KindNpc && e.NpcType == 3 && e.TypeID == 2 {
				smith = id
				return true
			}
		}
		return false
	})
	a.worldControls(&platform.Input{Chars: []rune{'n'}}, 1024, 768)
	if a.shop.npcID != smith {
		t.Fatal("blacksmith did not open")
	}
	a.worldControls(&platform.Input{}, 1024, 768)
	a.blacksmithUI.Mode = ui.ForgeEnchant
	ix, iy := a.inventoryUI.Origin(1024, 768)
	bx, by := a.blacksmithUI.Origin(1024, 768)
	click := func(x, y int) {
		a.worldControls(&platform.Input{MouseX: x, MouseY: y, Pressed: true, MouseDown: true}, 1024, 768)
		a.worldControls(&platform.Input{MouseX: x, MouseY: y, Clicked: true}, 1024, 768)
	}
	drag := func(key [2]uint8, slot int) {
		a.inventory.bag = key[0]
		sx, sy, _, _ := ui.InventoryCell(key[0], key[1])
		dx, dy := a.blacksmithUI.Slot(slot)
		a.worldControls(&platform.Input{MouseX: ix + sx + 2, MouseY: iy + sy + 2, Pressed: true, MouseDown: true}, 1024, 768)
		a.worldControls(&platform.Input{MouseX: bx + dx + 2, MouseY: by + dy + 2, Clicked: true}, 1024, 768)
	}
	try := func(itemKey, lapKey [2]uint8) bool {
		t.Helper()
		matKey := [2]uint8{1, 18}
		item, lap, mat := st.Inventory[itemKey], st.Inventory[lapKey], st.Inventory[matKey]
		drag(itemKey, 0)
		drag(lapKey, 1)
		drag(matKey, 2)
		click(bx+350, by+380)
		if a.blacksmith.quantity || a.gemConfirm.Visible {
			t.Fatal("enchant opened a quantity or confirmation dialog")
		}
		if !a.shop.pending || st.Inventory[itemKey] != item || st.Inventory[lapKey] != lap || st.Inventory[matKey] != mat {
			t.Fatal("enchantment mutated inventory before acknowledgement")
		}
		deadline := time.Now().Add(10 * time.Second)
		for a.shop.pending {
			a.pumpWorld()
			if a.state == nil || time.Now().After(deadline) {
				t.Fatal("enchantment refresh timed out")
			}
			time.Sleep(time.Millisecond)
		}
		if st.LastEnchant.LapisiaCount != lap.Count-1 || st.LastEnchant.MaterialCount != mat.Count-1 {
			t.Fatal("materials did not follow server response")
		}
		return st.LastEnchant.Result == 1
	}
	itemKey, safeLap := [2]uint8{1, 21}, [2]uint8{1, 19}
	for attempts := 0; ; attempts++ {
		if attempts == 10 {
			t.Fatal("no enchant success")
		}
		if try(itemKey, safeLap) {
			break
		}
	}
	if st.Inventory[itemKey].Craft != "00000000000000000001" {
		t.Fatal("enchant level missing")
	}
	var broken [2]uint8
	var originalQuality uint16
	for slot := uint8(0); slot < 6; slot++ {
		key := [2]uint8{2, slot}
		originalQuality = st.Inventory[key].Quality
		if !try(key, [2]uint8{1, 20}) {
			broken = key
			break
		}
	}
	if broken == [2]uint8{} || st.Inventory[broken].Quality != originalQuality {
		t.Fatal("failed enchant changed equipment quality")
	}
	if err := world.Shutdown(10 * time.Second); err != nil {
		t.Fatal(err)
	}
	next, _, _, _ := testworld.Enter(t)
	if next.Inventory[itemKey].Craft != "00000000000000000001" || next.Inventory[broken].Quality != originalQuality {
		t.Fatal("enchantment state did not persist")
	}
	t.Log("UI enchant success, safe failure and single material consumption confirmed and persisted")
}
