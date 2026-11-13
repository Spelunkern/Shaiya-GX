//go:build integration

package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

func verifyNativeWeaponEnchants(t *testing.T, a, b *App, pump func(func() bool)) {
	t.Helper()
	check := func(want int32) {
		t.Helper()
		b.prepareRemotePlayers()
		own := a.wv.self.WeaponEnchantRecords()
		remote := b.wv.actors[a.state.SelfID]
		if remote == nil {
			t.Fatal("missing enchant observer")
		}
		other := remote.character.WeaponEnchantRecords()
		if want == 0 {
			if len(own) != 0 || len(other) != 0 {
				t.Fatal("unequipped enchant remains", own, other)
			}
		} else if len(own) != 1 || len(other) != 1 || own[0] != want || other[0] != want {
			t.Fatal("owner/observer enchant recipes", own, other, "want", want)
		}
	}
	key := [2]uint8{0, 5}
	if own := a.wv.self.WeaponEnchantRecords(); len(own) != 1 || own[0] != 1 {
		t.Fatal("initial owner +5 enchant", own)
	}
	// The observer must see the initial 0x0303 enchant before any item moves.
	pump(func() bool { return b.state.Entities[a.state.SelfID].Shape.Equipment[5][2] == 5 })
	check(1)
	slot := uint8(0)
	for ; slot < 24; slot++ {
		if _, ok := a.state.Inventory[[2]uint8{1, slot}]; !ok {
			break
		}
	}
	if slot == 24 {
		t.Fatal("no free weapon fixture slot")
	}
	move := func(equip bool, level uint8) {
		t.Helper()
		p := proto.ItemMovePacket(0, 5, 1, slot)
		if equip {
			p = proto.ItemMovePacket(1, slot, 0, 5)
		}
		if err := a.world.Send(p); err != nil {
			t.Fatal(err)
		}
		pump(func() bool {
			if equip {
				return a.state.Inventory[key].Type != 0 && b.state.Entities[a.state.SelfID].Shape.Equipment[5][2] == level
			}
			return a.state.Inventory[key].Type == 0 && b.state.Entities[a.state.SelfID].Shape.Equipment[5][0] == 0
		})
	}
	move(false, 0)
	check(0)
	move(true, 5)
	check(1)
	for attempt := 0; ; attempt++ {
		if attempt == 40 {
			t.Fatal("no safe +5 to +6 enchant success")
		}
		before := a.state.Inventory[key]
		lapCount := a.state.Inventory[[2]uint8{1, 19}].Count
		matCount := a.state.Inventory[[2]uint8{1, 18}].Count
		if !a.sendEnchant([2]uint8{1, 19}, [2]uint8{1, 18}, key, 1) {
			t.Fatal("enchant request refused")
		}
		if a.state.Inventory[key] != before {
			t.Fatal("predicted enchant before server acknowledgement")
		}
		pump(func() bool { return !a.enchant.pending })
		if a.state.Inventory[[2]uint8{1, 19}].Count != lapCount-1 || a.state.Inventory[[2]uint8{1, 18}].Count != matCount-1 {
			t.Fatal("enchant consumption did not follow server acknowledgement")
		}
		if a.state.LastEnchant.Result == 1 {
			break
		}
		check(1)
	}
	pump(func() bool { return b.state.Entities[a.state.SelfID].Shape.Equipment[5][2] == 6 })
	check(2)
	move(false, 0)
	check(0)
	move(true, 6)
	check(2)
	a.inventory.open = true
	x, y := a.inventoryUI.Origin(1024, 734)
	cx, cy, _, _ := ui.InventoryCell(1, 21)
	for _, level := range []uint8{0, 6} {
		a.updateInventory(&platform.Input{MouseX: x + cx + 2, MouseY: y + cy + 2, RightPressed: true}, 1024, 734)
		pump(func() bool {
			return !a.inventory.pending && weaponEnchantFromCraft(a.state.Inventory[key].Craft) == level && b.state.Entities[a.state.SelfID].Shape.Equipment[5][2] == level
		})
		recipe := int32(0)
		if level == 6 {
			recipe = 2
		}
		check(recipe)
	}
	a.inventory.open = false
	t.Log("initial remote +5 glow, native enchant +5→+6 ACK, consumption and equip glow confirmed for owner and observer")
}
