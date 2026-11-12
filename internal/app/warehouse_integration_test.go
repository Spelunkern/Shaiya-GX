//go:build integration

package app

import (
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/testworld"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

func TestWarehouseMovesAgainstRapture(t *testing.T) {
	st, w, _, wait := testworld.Enter(t)
	wait(func() bool { return st.WarehouseReady })
	a := inputApp()
	a.state = st
	a.world = w
	bank, err := ui.NewWarehouse(inputTextures{})
	if err != nil {
		t.Fatal(err)
	}
	a.bankUI = bank
	var npc uint32
	wait(func() bool {
		for id, e := range st.Entities {
			if e.Kind == game.KindNpc && e.NpcType == 6 {
				npc = id
				return true
			}
		}
		return false
	})
	a.wv.pos[0], a.wv.pos[1], a.wv.pos[2] = st.Self.X, st.Self.Y, st.Self.Z
	a.worldControls(&platform.Input{Chars: []rune{'n'}}, 1024, 768)
	if a.shop.npcID != npc {
		t.Fatal("interacting with the server's banker did not open it")
	}
	a.worldControls(&platform.Input{}, 1024, 768)
	if !a.inventory.open || !a.warehouseOpen() {
		t.Fatal("banker did not open warehouse and inventory")
	}
	var source [2]uint8
	found := false
	for key, item := range st.Inventory {
		if key[0] > 0 && item.Type != 0 && item.Count > 0 {
			source = key
			found = true
			break
		}
	}
	if !found {
		t.Fatal("disposable fixture has no inventory item")
	}
	var slot uint8
	for ; slot < 240; slot++ {
		if _, ok := st.Warehouse[slot]; !ok {
			break
		}
	}
	if slot >= 240 {
		t.Fatal("warehouse fixture full")
	}
	a.bank.npcID = npc
	a.bank.page = int(slot) / 40
	a.inventory.open = true
	a.inventory.bag = source[0]
	ix, iy := a.inventoryUI.Origin(1024, 768)
	sx, sy, _, _ := ui.InventoryCell(source[0], source[1])
	bx, by := a.bankUI.Origin(1024, 768)
	dx, dy, _ := ui.WarehouseCell(a.bank.page, slot)
	drag := func(x, y, xx, yy int) {
		a.worldControls(&platform.Input{MouseX: x, MouseY: y, Pressed: true, MouseDown: true}, 1024, 768)
		a.worldControls(&platform.Input{MouseX: xx, MouseY: yy, Clicked: true}, 1024, 768)
	}
	ack := func() {
		t.Helper()
		deadline := time.Now().Add(5 * time.Second)
		for a.inventory.pending {
			a.pumpWorld()
			if time.Now().After(deadline) {
				t.Fatal("warehouse acknowledgement timeout")
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
	before := st.Inventory[source]
	drag(ix+sx+1, iy+sy+1, bx+dx+1, by+dy+1)
	if !a.inventory.pending || st.Inventory[source] != before {
		t.Fatal("deposit changed state before server response")
	}
	ack()
	if _, ok := st.Inventory[source]; ok {
		t.Fatal("deposit did not clear source")
	}
	if st.Warehouse[slot].Type != before.Type {
		t.Fatal("deposit missing from warehouse")
	}
	drag(bx+dx+1, by+dy+1, ix+sx+1, iy+sy+1)
	if !a.inventory.pending || st.Warehouse[slot].Type != before.Type {
		t.Fatal("withdrawal changed state before response")
	}
	ack()
	if _, ok := st.Warehouse[slot]; ok {
		t.Fatal("withdrawal did not clear warehouse")
	}
	if st.Inventory[source].Type != before.Type {
		t.Fatal("withdrawal missing from bag")
	}
}
