//go:build integration

package app

import (
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/testworld"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

func TestItemRepairAgainstRapture(t *testing.T) {
	slot, err := strconv.Atoi(os.Getenv("SHAIYA_TEST_WORN_SLOT"))
	if err != nil {
		t.Skip("requires disposable worn equipment fixtures")
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
	key, equipped := [2]uint8{1, 22}, [2]uint8{0, uint8(slot)}
	if st.Inventory[key].Quality != 1 || st.Inventory[equipped].Quality != 1 {
		t.Fatal("repair fixtures are not worn")
	}
	ix, iy := a.inventoryUI.Origin(1024, 768)
	bx, by := a.blacksmithUI.Origin(1024, 768)
	click := func(x, y int) {
		a.worldControls(&platform.Input{MouseX: x, MouseY: y, Pressed: true, MouseDown: true}, 1024, 768)
		a.worldControls(&platform.Input{MouseX: x, MouseY: y, Clicked: true}, 1024, 768)
	}
	selectItem := func() {
		sx, sy, _, _ := ui.InventoryCell(key[0], key[1])
		dx, dy := ui.BlacksmithSlot(0)
		a.worldControls(&platform.Input{MouseX: ix + sx + 2, MouseY: iy + sy + 2, Pressed: true, MouseDown: true}, 1024, 768)
		a.worldControls(&platform.Input{MouseX: bx + dx + 2, MouseY: by + dy + 2, Clicked: true}, 1024, 768)
	}
	confirm := func(x, y int) {
		click(x, y)
		if !a.gemConfirm.Visible {
			t.Fatal("repair confirmation missing")
		}
		a.worldControls(&platform.Input{Keys: []platform.Key{platform.KeyEnter}}, 1024, 768)
		if !a.shop.pending {
			t.Fatal("repair did not begin")
		}
	}
	poll := func() {
		deadline := time.Now().Add(10 * time.Second)
		for a.shop.pending {
			a.pumpWorld()
			if a.state == nil || time.Now().After(deadline) {
				t.Fatal("repair barrier timed out")
			}
			time.Sleep(time.Millisecond)
		}
	}
	money := st.Details.Money
	selectItem()
	before := st.Inventory[key]
	confirm(bx+40, by+306)
	if st.Inventory[key] != before || st.Details.Money != money {
		t.Fatal("repair applied optimistically")
	}
	poll()
	if !a.repair.accepted || st.Inventory[key].Quality <= 1 || st.Details.Money >= money {
		t.Fatal("individual repair not confirmed")
	}
	repaired, afterMoney := st.Inventory[key], st.Details.Money
	selectItem()
	confirm(bx+40, by+306)
	poll()
	if a.repair.accepted || st.Inventory[key] != repaired || st.Details.Money != afterMoney {
		t.Fatal("silent rejection was not resolved correctly")
	}
	confirm(bx+40, by+341)
	poll()
	if !a.repair.accepted || st.Inventory[equipped].Quality <= 1 || st.Details.Money >= afterMoney {
		t.Fatal("repair all did not refresh equipped item")
	}
	repairedEquipped := st.Inventory[equipped]
	if err := world.Shutdown(10 * time.Second); err != nil {
		t.Fatal(err)
	}
	next, _, _, _ := testworld.Enter(t)
	if next.Inventory[key].Quality != repaired.Quality || next.Inventory[equipped].Quality != repairedEquipped.Quality {
		t.Fatal("repairs did not persist")
	}
	t.Log("single repair, silent refusal and repair-all confirmed and persisted")
}
