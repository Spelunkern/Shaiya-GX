package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/data/text"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"testing"
)

func TestConfirmedPotionReuseGroupsAndOriginalRestoration(t *testing.T) {
	a := &App{clock: 100, state: game.New(), items: map[sdata.ItemKey]*sdata.Item{
		{Type: 25, TypeID: 1}: {Name: "Potion", ReqIg: 1, RecoveryHP: 200},
		{Type: 25, TypeID: 2}: {Name: "Other potion", ReqIg: 1},
		{Type: 25, TypeID: 3}: {Name: "Independent potion", ReqIg: 11},
	}, msg: text.SysMsg{100: "<v><hp> is restored with <i>"}}
	a.state.SelfID = 42
	a.state.Pools = proto.Pools{HP: 500, MP: 50, SP: 70}
	a.state.LastItemUse.ID = 42
	a.confirmedItems(proto.UseItem)
	if len(a.inventory.cooldowns) != 0 {
		t.Fatal("refusal started reuse")
	}
	a.state.LastItemUse.Type, a.state.LastItemUse.TypeID = 25, 1
	a.confirmedItems(proto.UseItem)
	if a.itemCooldown(proto.InventoryItem{Type: 25, TypeID: 2}) != 15 || a.itemCooldown(proto.InventoryItem{Type: 25, TypeID: 3}) != 0 {
		t.Fatal("reuse not shared by ReqIg")
	}
	if len(a.state.Chat) != 0 {
		t.Fatal("restoration predicted before resource snapshot")
	}
	a.state.Pools = proto.Pools{HP: 620, MP: 55, SP: 70}
	a.confirmedItems(proto.CharacterCurrentHitpoints)
	if len(a.state.Chat) != 1 || a.state.Chat[0].MessageID != 100 || a.state.Chat[0].Style != 4 || a.state.Chat[0].Text != "120HP is restored with Potion" {
		t.Fatal("wrong restoration", a.state.Chat)
	}
	a.state.Pools.HP = 700
	a.confirmedItems(proto.CharacterCurrentHitpoints)
	if len(a.state.Chat) != 1 {
		t.Fatal("unrelated regeneration claimed as potion")
	}
	a.clock = 115
	if a.itemCooldown(proto.InventoryItem{Type: 25, TypeID: 1}) != 0 || cooldownFlash(115, 115) <= 0 || cooldownFlash(115, 116) != 0 {
		t.Fatal("cooldown boundary/ready flash")
	}
}

func TestItemTooltipDoesNotInventConsumableStatsOrStackMetadata(t *testing.T) {
	d := &sdata.Item{Name: "Potion", Type: 25, RecoveryHP: 200, Count: 99, Classes: [6]uint8{1}, RequiredStats: [6]uint16{12}, Description: "Restores 200 HP."}
	rows := itemDetailLines(d, nil, 12, "", nil)
	if len(rows) != 2 || rows[0].text != "Potion" || rows[1].text != d.Description {
		t.Fatal("consumable has invented equipment/stack metadata", rows)
	}
	if rows[0].color != ([4]float32{1, 1, 0, 1}) {
		t.Fatal("native title colour")
	}
}
