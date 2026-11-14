package world

import (
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"reflect"
	"testing"
	"time"
)

func TestForgeQuoteMatchesExecutionAndDoesNotMutate(t *testing.T) {
	old := db.MustDefs()
	rates := enchantConfig
	t.Cleanup(func() { db.SetDefsForTest(old); enchantConfig = rates })
	db.SetDefsForTest(&db.Defs{Items: map[db.ItemKey]*db.ItemDef{
		{Type: 1, TypeID: 1}:   {Type: 1, TypeID: 1, Slot: 6, Server: 1, ReqLuc: 94001},
		{Type: 30, TypeID: 1}:  {Type: 30, TypeID: 1, Slot: 1, ReqLevel: 1, ReqIg: 40},
		{Type: 100, TypeID: 1}: {Type: 100, TypeID: 1, Special: specialGemSuccessUp, ReqVg: 20},
		{Type: 100, TypeID: 2}: {Type: 100, TypeID: 2, Special: specialGemExtract, ReqVg: 100},
		{Type: 95, TypeID: 1}:  {Type: 95, TypeID: 1, ReqLevel: 1, ReqIg: 96},
		{Type: 94, TypeID: 1}:  {Type: 94, TypeID: 1},
	}})
	enchantConfig.WeaponRate[0] = 800000
	s, wire := gmWireSession(t, db.Character{ID: 999900, Level: 1, Money: 9000000}, false)
	s.items = []db.Item{{UID: 1, Bag: 1, Slot: 0, Type: 1, TypeID: 1, Count: 1}, {UID: 2, Bag: 1, Slot: 1, Type: 30, TypeID: 1, Count: 2}, {UID: 3, Bag: 1, Slot: 2, Type: 100, TypeID: 1, Count: 1}, {UID: 4, Bag: 1, Slot: 3, Type: 95, TypeID: 1, Count: 1}}
	request := proto.ForgeQuoteRequest{Serial: 42, Kind: proto.ForgeQuoteLink, Item: [2]uint8{1, 0}, Lapis: [2]uint8{1, 1}, Hammer: [2]uint8{1, 2}, Material: [2]uint8{255, 255}, Count: 1}
	before := append([]db.Item(nil), s.items...)
	t.Setenv("RAPTURE_FORGE_COST_PERCENT", "0")
	q := s.forgeQuote(request)
	if !q.Valid || q.Gold != 1040000 || q.Rate != 210000 {
		t.Fatal(q)
	}
	t.Setenv("RAPTURE_FORGE_COST_PERCENT", "100")
	q = s.forgeQuote(request)
	if q.Gold != 1040000 {
		t.Fatal(q)
	}
	s.handleForgeQuote(proto.ForgeQuotePacket(request))
	select {
	case p := <-wire:
		got, err := proto.DecodeForgeQuote(p)
		if err != nil || got != q {
			t.Fatal(got, err)
		}
	case <-time.After(time.Second):
		t.Fatal("no quote response")
	}
	if !reflect.DeepEqual(before, s.items) || s.char.Money != 9000000 {
		t.Fatal("preview mutated character")
	}
	s.handleGemAdd(proto.GemAddHammerPacket(1, 1, 1, 0, 1, 2))
	if s.char.Money != 9000000-q.Gold {
		t.Fatal("quote and charged cost diverge")
	}
	s.items[0].Gems[0] = 1
	s.items = append(s.items, db.Item{UID: 5, Bag: 1, Slot: 4, Type: 100, TypeID: 2, Count: 3})
	request.Hammer = [2]uint8{1, 4}
	request.Count = 0
	request.Kind = proto.ForgeQuoteExtract
	q = s.forgeQuote(request)
	if !q.Valid || q.Gold != 1040000 {
		t.Fatal("extraction quote", q)
	}
	money := s.char.Money
	s.handleGemRemove(proto.GemRemoveHammerPacket([2]uint8{1, 0}, [2]uint8{1, 4}, 0))
	if s.char.Money != money-q.Gold {
		t.Fatal("extraction charged a different cost")
	}
	request.Kind = proto.ForgeQuoteEnchant
	request.Lapis = [2]uint8{1, 3}
	request.Count = 10
	q = s.forgeQuote(request)
	if !q.Valid || q.Rate != 800000 || q.Gold != 2500000 || q.Material != 94001 {
		t.Fatal("missing hint or inaccurate enchant rate", q)
	}
	t.Setenv("RAPTURE_FORGE_COST_PERCENT", "0")
	if s.forgeQuote(request).Gold != 2500000 {
		t.Fatal("obsolete free-cost override affected enchant")
	}
}
