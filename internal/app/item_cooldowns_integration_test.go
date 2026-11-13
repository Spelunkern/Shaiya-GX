//go:build integration

package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

func verifyPotionReuse(t *testing.T, a *App, pump func(func() bool)) {
	t.Helper()
	key := [2]uint8{1, 23}
	item := a.state.Inventory[key]
	d := a.itemsItem(item.Type, item.TypeID)
	if d == nil || item.Count < 2 || itemReuseSeconds(d.ReqIg) == 0 {
		t.Fatal("missing reusable disposable potion", item, d)
	}
	old := a.state.QuickSlots
	a.state.QuickSlots = []proto.QuickSlot{{Slot: 0, Bag: key[0], Number: uint16(key[1])}, {Slot: 1, Bag: key[0], Number: uint16(key[1])}}
	before := a.state.Pools
	chats := len(a.state.Chat)
	a.state.LastItemUse.Type = 0
	a.inventory.open = true
	x, y := a.inventoryUI.Origin(1024, 734)
	cx, cy, _, _ := ui.InventoryCell(key[0], key[1])
	a.updateInventory(&platform.Input{MouseX: x + cx + 2, MouseY: y + cy + 2, Pressed: true, MouseDown: true}, 1024, 734)
	a.updateInventory(&platform.Input{MouseX: x + cx + 2, MouseY: y + cy + 2, Clicked: true}, 1024, 734)
	a.inventory.open = false
	if a.actionPacket(1) != nil {
		t.Fatal("second shortcut bypassed pending item use")
	}
	pump(func() bool {
		return a.state.LastItemUse.Type == item.Type && a.state.Inventory[key].Count == item.Count-1 && !a.inventory.pending
	})
	deadline := time.Now().Add(time.Second)
	for a.state.Pools == before && time.Now().Before(deadline) {
		a.pumpWorld()
		time.Sleep(time.Millisecond)
	}
	if a.itemCooldown(item) <= 0 || a.actionPacket(0) != nil || a.actionPacket(1) != nil {
		t.Fatal("confirmed shared cooldown is not enforced in client")
	}
	restoration := 0
	for _, line := range a.state.Chat[chats:] {
		if line.MessageID == 100 && line.Style == 4 {
			restoration++
		}
	}
	if a.state.Pools.HP > before.HP || a.state.Pools.MP > before.MP || a.state.Pools.SP > before.SP {
		if restoration == 0 {
			t.Fatal("potion restored resources without original green SysMsg100")
		}
	}
	// Bypass the client guard deliberately: the real isolated server must
	// refuse a second use and leave both resources and stack unchanged.
	a.state.LastItemUse.Type = 255
	if err := a.world.Send(proto.UseItemPacket(key[0], key[1])); err != nil {
		t.Fatal(err)
	}
	pump(func() bool { return a.state.LastItemUse.Type == 0 })
	if a.state.Inventory[key].Count != item.Count-1 {
		t.Fatal("server accepted reuse during cooldown")
	}
	a.state.QuickSlots = old
	t.Logf("ReqIg %d reuse %.0fs: both shortcuts blocked; bypass refused by Rapture; original green restoration lines=%d", d.ReqIg, itemReuseSeconds(d.ReqIg), restoration)
}
