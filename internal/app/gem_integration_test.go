//go:build integration

package app

import (
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/testworld"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

// The fixtures are disposable: a safe gem stack and compatible socketed item.
func TestGemAddAgainstRapture(t *testing.T) {
	gemID, err1 := strconv.Atoi(os.Getenv("SHAIYA_TEST_GEM_ID"))
	itemID, err2 := strconv.Atoi(os.Getenv("SHAIYA_TEST_GEM_TARGET_ID"))
	if err1 != nil || err2 != nil {
		t.Skip("requires gem fixtures")
	}
	st, world, send, wait := testworld.Enter(t)
	gemKey, itemKey := [2]uint8{1, 21}, [2]uint8{1, 22}
	hammerKey := [2]uint8{1, 19}
	gem, item := st.Inventory[gemKey], st.Inventory[itemKey]
	if gem.Type != 30 || int(gem.TypeID) != gemID || gem.Count != 20 || item.Type != 1 || int(item.TypeID) != itemID || item.Count != 1 {
		t.Fatal("invalid disposable gem fixtures")
	}
	hammer := st.Inventory[hammerKey]
	if hammer.Count != 1 || (hammer.Type != 25 && hammer.Type != 44 && hammer.Type != 100) {
		t.Fatal("missing disposable linking hammer")
	}
	// Rapture echoes the authoritative item even for a same-slot move.
	// Simulate stale client state to exercise both an occupied and empty slot.
	stale := gem
	stale.Count--
	st.Inventory[gemKey] = stale
	send(proto.ItemMovePacket(gemKey[0], gemKey[1], gemKey[0], gemKey[1]))
	wait(func() bool { return st.Inventory[gemKey] == gem })
	empty := [2]uint8{1, 20}
	if _, occupied := st.Inventory[empty]; occupied {
		t.Fatal("same-slot probe requires an empty slot")
	}
	st.Inventory[empty] = proto.InventoryItem{Bag: empty[0], Slot: empty[1], Type: 25, TypeID: 1, Count: 1}
	send(proto.ItemMovePacket(empty[0], empty[1], empty[0], empty[1]))
	wait(func() bool { _, occupied := st.Inventory[empty]; return !occupied })
	initialMoney := st.Details.Money
	app := inputApp()
	app.state, app.world = st, world
	app.blacksmithUI, _ = ui.NewBlacksmith(inputTextures{})
	app.gemConfirm, _ = ui.NewMessage(inputTextures{})
	app.npcQuestUI, _ = ui.NewNPCQuest(inputTextures{})
	app.wv.pos[0], app.wv.pos[1], app.wv.pos[2] = st.Self.X, st.Self.Y, st.Self.Z
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
	app.worldControls(&platform.Input{Chars: []rune{'n'}}, 1024, 768)
	if app.shop.npcID != smith {
		t.Fatal("nearby blacksmith did not open")
	}
	app.worldControls(&platform.Input{}, 1024, 768)
	if !app.inventory.open {
		t.Fatal("blacksmith did not open inventory")
	}
	ix, iy := app.inventoryUI.Origin(1024, 768)
	bx, by := app.blacksmithUI.Origin(1024, 768)
	drag := func(key [2]uint8, slot int) {
		sx, sy, _, _ := ui.InventoryCell(key[0], key[1])
		dx, dy := ui.BlacksmithSlot(slot)
		app.worldControls(&platform.Input{MouseX: ix + sx + 2, MouseY: iy + sy + 2, Pressed: true, MouseDown: true}, 1024, 768)
		app.worldControls(&platform.Input{MouseX: bx + dx + 2, MouseY: by + dy + 2, Clicked: true}, 1024, 768)
	}
	// A rejected target must not consume the optional hammer or the gem.
	if !app.sendGemLink(gemKey, [2]uint8{1, 18}, hammerKey) {
		t.Fatal("invalid-target request did not start")
	}
	deadline := time.Now().Add(10 * time.Second)
	for app.shop.pending {
		app.pumpWorld()
		if app.state == nil || time.Now().After(deadline) {
			t.Fatal("rejected link refresh timed out")
		}
		time.Sleep(time.Millisecond)
	}
	if st.LastGemAdd.Result != 0 || st.Inventory[gemKey] != gem || st.Inventory[hammerKey] != hammer {
		t.Fatal("rejected link consumed its gem or hammer")
	}
	for attempt := 0; attempt < 20; attempt++ {
		before, serial := st.Inventory[gemKey], st.GemAddSerial
		drag(itemKey, 0)
		drag(gemKey, 1)
		if attempt == 0 {
			drag(hammerKey, 3)
		}
		app.worldControls(&platform.Input{MouseX: bx + 300, MouseY: by + 380, Pressed: true, MouseDown: true}, 1024, 768)
		app.worldControls(&platform.Input{MouseX: bx + 300, MouseY: by + 380, Clicked: true}, 1024, 768)
		awaitForgeConfirmation(t, app)
		if !app.gemConfirm.Visible {
			t.Fatal("missing link risk confirmation")
		}
		app.worldControls(&platform.Input{Keys: []platform.Key{platform.KeyEnter}}, 1024, 768)
		if !app.shop.pending {
			t.Fatal("gem request was not started")
		}
		if st.Inventory[gemKey] != before || st.Inventory[itemKey].Gems[0] != 0 || attempt == 0 && st.Inventory[hammerKey] != hammer {
			t.Fatal("gem request changed inventory before response")
		}
		deadline := time.Now().Add(10 * time.Second)
		for app.shop.pending {
			app.pumpWorld()
			if app.state == nil || time.Now().After(deadline) {
				t.Fatal("gem response or protection refresh timed out")
			}
			time.Sleep(time.Millisecond)
		}
		if st.GemAddSerial <= serial || len(app.gemLink.refresh.pending) != 0 {
			t.Fatal("gem operation completed without all confirmations")
		}
		if _, exists := st.Inventory[hammerKey]; exists {
			t.Fatal("consumed linking hammer still appears in inventory")
		}
		if st.LastGemAdd.GemCount != before.Count-1 || st.Details.Money > initialMoney {
			t.Fatal("server did not consume the gem or returned invalid money")
		}
		if st.LastGemAdd.Result == 1 {
			if st.Inventory[itemKey].Gems[st.LastGemAdd.Position] != uint8(gemID) {
				t.Fatal("confirmed gem not linked")
			}
			if err := world.Shutdown(10 * time.Second); err != nil {
				t.Fatal(err)
			}
			next, _, _, waitNext := testworld.Enter(t)
			waitNext(func() bool { return next.Inventory[itemKey].Gems[st.LastGemAdd.Position] == uint8(gemID) })
			if _, exists := next.Inventory[hammerKey]; exists {
				t.Fatal("consumed hammer returned after reconnect")
			}
			t.Logf("server linked and persisted gem after %d attempt(s)", attempt+1)
			return
		}
	}
	t.Fatal("no successful link after 20 attempts")
}

// The risky gem and a single Lucky Charm are provisioned only on an N2 account.
func TestGemProtectionAgainstRapture(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_GEM_PROTECTION") != "1" {
		t.Skip("requires disposable protection fixture")
	}
	st, world, _, _ := testworld.Enter(t)
	app := inputApp()
	app.state, app.world = st, world
	gemKey, charmKey := [2]uint8{1, 21}, [2]uint8{1, 22}
	if st.Inventory[gemKey].Type != 30 || st.Inventory[charmKey].Type != 100 || st.Inventory[charmKey].Count != 1 {
		t.Fatal("missing risky gem or protection fixture")
	}
	for slot := uint8(0); slot < 6; slot++ {
		itemKey := [2]uint8{2, slot}
		if st.Inventory[itemKey].Type != 1 || !app.sendGemLink(gemKey, itemKey) {
			t.Fatal("unable to start risky link")
		}
		deadline := time.Now().Add(10 * time.Second)
		for app.shop.pending {
			app.pumpWorld()
			if app.state == nil || time.Now().After(deadline) {
				t.Fatal("protection refresh timed out")
			}
			time.Sleep(time.Millisecond)
		}
		if st.LastGemAdd.Result == 0 {
			if _, exists := st.Inventory[charmKey]; exists {
				t.Fatal("consumed last Lucky Charm still appears in inventory")
			}
			if _, exists := st.Inventory[itemKey]; !exists {
				t.Fatal("protected item was destroyed")
			}
			if err := world.Shutdown(10 * time.Second); err != nil {
				t.Fatal(err)
			}
			next, _, _, _ := testworld.Enter(t)
			if _, exists := next.Inventory[charmKey]; exists {
				t.Fatal("consumed protection returned after reconnect")
			}
			if _, exists := next.Inventory[itemKey]; !exists {
				t.Fatal("protected item did not persist")
			}
			t.Logf("protected failure and refresh confirmed after %d attempt(s)", slot+1)
			return
		}
	}
	t.Fatal("all six risky links succeeded; protected failure not exercised")
}

func TestGemExtractionAgainstRapture(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_GEM_EXTRACT") != "1" {
		t.Skip("requires disposable extraction fixtures")
	}
	gemID, err := strconv.Atoi(os.Getenv("SHAIYA_TEST_GEM_ID"))
	if err != nil {
		t.Fatal(err)
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
	a.inventory.bag = 2
	ix, iy := a.inventoryUI.Origin(1024, 768)
	bx, by := a.blacksmithUI.Origin(1024, 768)
	successes, failures := 0, 0
	var checked [][2]uint8
	for slot := uint8(0); slot < 12 && (successes == 0 || failures == 0); slot++ {
		key := [2]uint8{2, slot}
		item := st.Inventory[key]
		if item.Type != 1 || int(item.Gems[0]) != gemID {
			t.Fatal("invalid extraction fixture")
		}
		sx, sy, _, _ := ui.InventoryCell(key[0], key[1])
		dx, dy := ui.BlacksmithSlot(0)
		a.worldControls(&platform.Input{MouseX: ix + sx + 2, MouseY: iy + sy + 2, Pressed: true, MouseDown: true}, 1024, 768)
		a.worldControls(&platform.Input{MouseX: bx + dx + 2, MouseY: by + dy + 2, Clicked: true}, 1024, 768)
		serial := st.GemRemoveSerial
		a.worldControls(&platform.Input{MouseX: bx + 435, MouseY: by + 380, Pressed: true, MouseDown: true}, 1024, 768)
		a.worldControls(&platform.Input{MouseX: bx + 435, MouseY: by + 380, Clicked: true}, 1024, 768)
		awaitForgeConfirmation(t, a)
		if !a.gemConfirm.Visible || st.Inventory[key] != item {
			t.Fatal("missing extraction confirmation or optimistic mutation")
		}
		a.worldControls(&platform.Input{Keys: []platform.Key{platform.KeyEnter}}, 1024, 768)
		if !a.shop.pending || st.Inventory[key] != item {
			t.Fatal("extraction changed equipment before response")
		}
		deadline := time.Now().Add(10 * time.Second)
		for a.shop.pending {
			a.pumpWorld()
			if a.state == nil || time.Now().After(deadline) {
				t.Fatal("extraction or refresh timed out")
			}
			time.Sleep(time.Millisecond)
		}
		if st.GemRemoveSerial <= serial || st.Inventory[key].Gems[0] != 0 {
			t.Fatal("equipment not refreshed after extraction")
		}
		result := st.LastGemRemove
		if result.Result == 1 {
			successes++
			gem := st.Inventory[[2]uint8{result.Bags[0], result.Slots[0]}]
			if result.Removed[0] != 1 || int(gem.TypeID) != gemID || gem.Type != 30 {
				t.Fatal("recovered lapis missing")
			}
		} else {
			failures++
		}
		checked = append(checked, key)
	}
	if successes == 0 || failures == 0 {
		t.Fatal("did not exercise both extraction outcomes")
	}
	if err := world.Shutdown(10 * time.Second); err != nil {
		t.Fatal(err)
	}
	next, _, _, _ := testworld.Enter(t)
	for _, key := range checked {
		if next.Inventory[key].Gems[0] != 0 {
			t.Fatal("extracted equipment did not persist")
		}
	}
	t.Logf("extraction succeeded %d times, failed %d times; server state persisted", successes, failures)
}
