package world

import (
	"encoding/binary"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
	"testing"
)

func TestNativeStacksSurviveServerClientCodecs(t *testing.T) {
	it := db.Item{Bag: 1, Slot: 2, Type: 25, TypeID: 1, Count: 255, Quality: 17, CraftName: "010203"}
	payload := func(frame []byte) []byte {
		if binary.LittleEndian.Uint16(frame[2:4])&0x4000 != 0 {
			t.Fatal("retired wide opcode emitted")
		}
		return frame[2:]
	}
	state := game.New()
	for _, frame := range buildInventoryPackets([]db.Item{it}) {
		if _, err := state.Apply(payload(frame)); err != nil {
			t.Fatal(err)
		}
	}
	if state.Inventory[[2]uint8{1, 2}].Count != 255 {
		t.Fatal("inventory count truncated")
	}
	moved, err := proto.DecodeItemMove(payload(buildBagToBag(1, 2, &it, 1, 3, nil)))
	if err != nil || moved[0].Count != 255 || moved[1].Count != 0 {
		t.Fatal("move/empty slot", moved, err)
	}
	list, err := proto.DecodeWarehouse(payload(buildWarehouseItemList([]db.Item{it})))
	if err != nil || list.Items[0].Count != 255 {
		t.Fatal("warehouse", list, err)
	}
	for _, item := range []*db.Item{&it, nil} {
		e, err := proto.DecodeGuildBankEvent(payload(buildGuildBankSlot(packet.GuildBankPlaced, 2, item, 123)))
		if err != nil || e.Actor != 123 || item != nil && e.Item.Count != 255 || item == nil && e.Item.Count != 0 {
			t.Fatal("guild bank", e, err)
		}
	}
	_, offer, err := proto.DecodeExchangePartnerItem(payload(buildTradeAddDest(0, &offeredItem{Item: it, Count: 255})))
	if err != nil || offer.Count != 255 || offer.Craft != it.CraftName {
		t.Fatal("exchange", offer, err)
	}
	purchase, err := proto.DecodeMallPurchase(payload(buildPointBuyResult(0, 0, "TEST", 0, []grantedItem{{1, 2, 25, 1, 255}})))
	if err != nil || purchase.Items[0].Count != 255 {
		t.Fatal("mall", purchase, err)
	}
	use, err := proto.DecodeItemUse(payload(buildUseItemResult(123, &it)))
	if err != nil || use.Count != 255 {
		t.Fatal("use", use, err)
	}
	ground, err := proto.DecodeGroundItem(payload(buildMapAddItem(&MapItem{ID: 123, Item: it, PosX: 3, PosY: 4, PosZ: 5})))
	if err != nil || ground.Count != 255 || ground.Z != 5 {
		t.Fatal("ground", ground, err)
	}
	for _, frame := range [][]byte{buildShopBuyResult(0, 1, 2, 25, 1, 255, 123), buildShopSellResult(0, 1, 2, 25, 1, 255, 123)} {
		v, err := proto.DecodeShopResult(payload(frame))
		if err != nil || v.Count != 255 || v.Money != 123 {
			t.Fatal("shop", v, err)
		}
	}
	gem, err := proto.DecodeGemAddResult(payload(buildGemAddResult(1, 1, 2, 255, 1, 3, 0, 4, 123)))
	if err != nil || gem.GemCount != 255 || gem.Money != 123 {
		t.Fatal("gem", gem, err)
	}
	enchant, err := proto.DecodeEnchantResult(payload(buildEnchantResult(1, 1, 2, 255, 1, 3, 255, 0, 5, 123, "010203")))
	if err != nil || enchant.LapisiaCount != 255 || enchant.MaterialCount != 255 || enchant.Money != 123 {
		t.Fatal("enchant", enchant, err)
	}
	if _, err := state.Apply(payload(buildRemoveItem(&it, false))); err != nil || state.Inventory[[2]uint8{1, 2}].Count != 255 {
		t.Fatal("partial remove", err)
	}
	if _, err := state.Apply(payload(buildRemoveItem(&it, true))); err != nil || state.Inventory[[2]uint8{1, 2}].Count != 0 {
		t.Fatal("full remove", err)
	}
}

func TestNativeQuantityRequestsKeepFollowingFields(t *testing.T) {
	for _, p := range [][]byte{proto.ExchangeAddPacket(1, 2, 255, 7), proto.DuelAddPacket(1, 2, 255, 7)} {
		r := packet.NewReader(p)
		if r.U8() != 1 || r.U8() != 2 || r.Count() != 255 || r.U8() != 7 || r.Err() != nil {
			t.Fatal("offer request", p)
		}
	}
	r := packet.NewReader(proto.EnchantPacket(1, 2, 1, 3, 255, 0, 5))
	if r.U8() != 1 || r.U8() != 2 || r.U8() != 1 || r.U8() != 3 || r.Count() != 255 || r.U8() != 0 || r.U8() != 5 || r.Err() != nil {
		t.Fatal("enchant request")
	}
}
