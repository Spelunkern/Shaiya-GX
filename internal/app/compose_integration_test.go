//go:build integration

package app

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/testworld"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

func TestComposeAgainstRapture(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_COMPOSE") != "1" {
		t.Skip("requires disposable recreation fixtures")
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
	a.blacksmithUI.Mode = ui.ForgeRemake
	ix, iy := a.inventoryUI.Origin(1024, 768)
	bx, by := a.blacksmithUI.Origin(1024, 768)
	click := func(x, y int) {
		a.worldControls(&platform.Input{MouseX: x, MouseY: y, Pressed: true, MouseDown: true}, 1024, 768)
		a.worldControls(&platform.Input{MouseX: x, MouseY: y, Clicked: true}, 1024, 768)
	}
	drag := func(key [2]uint8, slot int) {
		sx, sy, _, _ := ui.InventoryCell(key[0], key[1])
		dx, dy := a.blacksmithUI.Slot(slot)
		a.worldControls(&platform.Input{MouseX: ix + sx + 2, MouseY: iy + sy + 2, Pressed: true, MouseDown: true}, 1024, 768)
		a.worldControls(&platform.Input{MouseX: bx + dx + 2, MouseY: by + dy + 2, Clicked: true}, 1024, 768)
	}
	itemKey, runeKey, wrongKey := [2]uint8{1, 21}, [2]uint8{1, 20}, [2]uint8{1, 19}
	item, rune, wrong := st.Inventory[itemKey], st.Inventory[runeKey], st.Inventory[wrongKey]
	if item.Craft != "00000000000000000009" || rune.Count != 2 || wrong.Count != 2 {
		t.Fatal("bad recreation fixture")
	}
	perform := func(source [2]uint8) {
		t.Helper()
		beforeItem, beforeRune := st.Inventory[itemKey], st.Inventory[source]
		drag(itemKey, 0)
		drag(source, 2)
		click(bx+350, by+380)
		if !a.gemConfirm.Visible {
			t.Fatal("missing recreation confirmation")
		}
		a.worldControls(&platform.Input{Keys: []platform.Key{platform.KeyEnter}}, 1024, 768)
		if !a.shop.pending || st.Inventory[itemKey] != beforeItem || st.Inventory[source] != beforeRune {
			t.Fatal("recreation changed items optimistically")
		}
		deadline := time.Now().Add(10 * time.Second)
		for a.shop.pending {
			a.pumpWorld()
			if a.state == nil || time.Now().After(deadline) {
				t.Fatal("recreation refresh timed out")
			}
			time.Sleep(time.Millisecond)
		}
	}
	perform(wrongKey)
	if a.compose.success || st.Inventory[itemKey] != item || st.Inventory[wrongKey] != wrong {
		t.Fatal("rejected recreation modified inventory")
	}
	perform(runeKey)
	craft := st.Inventory[itemKey].Craft
	if !a.compose.success || st.Inventory[runeKey].Count != 1 || len(craft) != 20 || !strings.HasSuffix(craft, "09") || craft == item.Craft {
		t.Fatal("confirmed recreation or rune consumption missing")
	}
	if len(st.Chat) < 2 || st.Chat[len(st.Chat)-2].Text != "Before: No bonus attributes" || st.Chat[len(st.Chat)-1].Text != "After: "+craftSummary(craft) {
		t.Fatal("confirmed recreation comparison missing from chat")
	}
	if err := world.Shutdown(10 * time.Second); err != nil {
		t.Fatal(err)
	}
	next, _, _, _ := testworld.Enter(t)
	if next.Inventory[itemKey].Craft != craft || next.Inventory[runeKey].Count != 1 || next.Inventory[wrongKey] != wrong {
		t.Fatal("recreation did not persist")
	}
	t.Log("UI recreation and rejection confirmed; enchant preserved and rune consumption persisted")
}
