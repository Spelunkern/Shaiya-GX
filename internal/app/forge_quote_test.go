package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/data/text"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
	"strings"
	"testing"
)

func testForgeQuotePacket(q proto.ForgeQuote) []byte {
	w := proto.NewWriter(proto.GXForgeQuote)
	w.U32(q.Serial)
	w.U8(q.Kind)
	w.Bool(q.Valid)
	w.U32(q.Rate)
	w.U32(q.Gold)
	w.U8(q.Lapis)
	w.U32(q.Material)
	return w.Bytes()
}
func TestForgeNativeConfirmationUsesActualCostAndRate(t *testing.T) {
	a := inputApp()
	a.msg = text.SysMsg{10011: "Gold <b>, success <v>%%%%. Link?", 840: "Extract: <b> Gold, <v>%%"}
	for _, q := range []proto.ForgeQuote{{Kind: proto.ForgeQuoteLink, Rate: 250000, Gold: 0}, {Kind: proto.ForgeQuoteExtract, Rate: 125000, Gold: 12345}, {Kind: proto.ForgeQuoteEnchant, Rate: 799999, Gold: 7500}} {
		value := a.forgeConfirmation(q)
		if strings.Contains(value, "<") || strings.Contains(value, "%%") || !strings.Contains(value, forgePercent(q.Rate)+"%") {
			t.Fatal(value)
		}
	}
}
func TestForgeHoverIncludesInstancesSocketsHammerAndHints(t *testing.T) {
	a := inputApp()
	a.blacksmithUI, _ = ui.NewBlacksmith(inputTextures{})
	a.blacksmith.keys = [4][2]uint8{{1, 0}, {1, 1}, {1, 2}, {1, 3}}
	for i, key := range a.blacksmith.keys {
		it := proto.InventoryItem{Bag: 1, Slot: uint8(i), Type: uint8(i + 1), TypeID: 1, Count: 1}
		a.state.Inventory[key] = it
		a.blacksmith.items[i] = it
		a.blacksmith.selected[i] = true
	}
	item := a.state.Inventory[[2]uint8{1, 0}]
	item.Gems[0] = 7
	a.state.Inventory[[2]uint8{1, 0}] = item
	for _, slot := range []int{0, 1, 3} {
		sx, sy := a.blacksmithUI.Slot(slot)
		it, ok, ghost := a.blacksmithHoveredItem(sx+1, sy+1)
		if !ok || ghost || it != a.state.Inventory[a.blacksmith.keys[slot]] {
			t.Fatal(slot, it)
		}
	}
	// Existing socket hovers even when it is not the active incoming socket.
	a.blacksmithUI.GemSlot = 1
	it, ok, _ := a.blacksmithHoveredItem(271, 207)
	if !ok || it.Type != 30 || it.TypeID != 7 {
		t.Fatal(it)
	}
	a.blacksmithUI.Mode = ui.ForgeEnchant
	a.blacksmith.selected[1] = false
	a.blacksmith.selected[2] = false
	a.blacksmith.quote.known = true
	a.blacksmith.quote.value = proto.ForgeQuote{Lapis: 4, Material: 94001}
	sx, sy := a.blacksmithUI.Slot(1)
	if _, ok, _ := a.blacksmithHoveredItem(sx+1, sy+1); ok {
		t.Fatal("lapisia ghost must not exist")
	}
	sx, sy = a.blacksmithUI.Slot(2)
	it, ok, ghost := a.blacksmithHoveredItem(sx+1, sy+1)
	if !ok || !ghost || it.Type != 94 {
		t.Fatal("missing material ghost", it)
	}

}
func TestForgeRejectsStaleQuoteAndKeepsRecreatedItem(t *testing.T) {
	a := inputApp()
	a.shop.npcID = 42
	a.state.Entities[42] = &game.Entity{ID: 42, Kind: game.KindNpc, NpcType: 3}
	a.blacksmithUI, _ = ui.NewBlacksmith(inputTextures{})
	a.gemConfirm, _ = ui.NewMessage(inputTextures{})
	a.blacksmith.selected[0] = true
	a.blacksmith.keys[0] = [2]uint8{1, 0}
	it := proto.InventoryItem{Bag: 1, Type: 1, TypeID: 1, Count: 1, Craft: "01000000000000000000"}
	a.blacksmith.items[0] = it
	a.state.Inventory[[2]uint8{1, 0}] = it
	a.requestForgeQuote(proto.ForgeQuoteEnchant, 5, true)
	serial := a.blacksmith.quote.request.Serial
	it.Craft = "07000000000000000000"
	a.state.Inventory[[2]uint8{1, 0}] = it
	a.confirmedForgeQuote(testForgeQuotePacket(proto.ForgeQuote{Serial: serial, Kind: proto.ForgeQuoteEnchant, Valid: true, Rate: 500000}))
	if a.gemConfirm.Visible {
		t.Fatal("accepted quote for replaced stats")
	}
	a.syncBlacksmithItems()
	if !a.blacksmith.selected[0] || a.blacksmith.items[0].Craft != it.Craft {
		t.Fatal("recreated item was not refreshed")
	}
	a.requestForgeQuote(proto.ForgeQuoteEnchant, 5, true)
	a.confirmedForgeQuote(testForgeQuotePacket(proto.ForgeQuote{Serial: a.blacksmith.quote.request.Serial, Kind: proto.ForgeQuoteEnchant, Valid: true, Rate: 500000}))
	if !a.gemConfirm.Visible {
		t.Fatal("fresh authoritative quote did not open confirmation")
	}
	a.items = map[sdata.ItemKey]*sdata.Item{{Type: 1, TypeID: 1}: {Type: 1, Name: "Equipment"}}
	a.drawItemDetails(a.itemsItem(1, 1), &it, 1, 0, 0, 1024, 768, "")
	if len(a.tooltip) == 0 {
		t.Fatal("missing inventory tooltip")
	}
}

func TestNativeLinkConfirmationOmitsObsoleteVgIntro(t *testing.T) {
	a := inputApp()
	a.msg = text.SysMsg{10011: "cost depends on tendency\nneed <b> gold, <v>%%%% chance.\nLink?"}
	message := a.forgeConfirmation(proto.ForgeQuote{Kind: proto.ForgeQuoteLink, Rate: 610000, Gold: 4095})
	if strings.Contains(message, "tendency") || !strings.Contains(message, "4095") || !strings.Contains(message, "61%") || !strings.Contains(message, "Link?") {
		t.Fatal(message)
	}
}
