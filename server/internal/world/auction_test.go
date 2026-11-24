package world

import (
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
	"testing"
	"time"
)

func TestAuctionContactFollowsBoardNotMapOrFaction(t *testing.T) {
	old := npcDefs
	t.Cleanup(func() { npcDefs = old })
	npcDefs[0] = []npcDef{{Type: 1, TypeID: 1, ShopType: proto.AuctionBoard}, {Type: 1, TypeID: 2}}
	for i, zone := range []uint16{42, 60421} {
		board := uint32(3500000 + i*2)
		AddEntities(zone, MapEntities{NPCs: []MapNpc{{ID: board, Type: 1, TypeID: 1, PosX: 10, PosY: 2, PosZ: 10}, {ID: board + 1, Type: 1, TypeID: 2, PosX: 10, PosY: 2, PosZ: 10}}})
		s := &Session{char: &db.Character{MapID: zone, CurrentHP: 100, PosX: 10, PosY: 2, PosZ: 10}, faction: uint8(i)}
		s.useNpc(board + 1)
		if s.auctionContact {
			t.Fatal("ordinary shop opened market")
		}
		s.useNpc(board)
		if !s.auctionContact || !s.auctionNPC(board) {
			t.Fatal("board not recognized", zone)
		}
		s.char.PosX = 21
		if s.auctionNPC(board) {
			t.Fatal("market stayed available outside reach")
		}
		s.char.PosX = 10
		s.dead = true
		if s.auctionNPC(board) {
			t.Fatal("dead character retained market authorization")
		}
		s.useNpc(0)
	}
}

func TestAuctionUnavailableUsesWorldFraming(t *testing.T) {
	s, packets := gmWireSession(t, db.Character{CurrentHP: 100}, false)
	s.dispatch(packet.Opcode(proto.GXAuction), proto.AuctionPacket(proto.AuctionRequest{NPC: 42, Sequence: 7}))
	select {
	case body := <-packets:
		var r proto.AuctionReply
		if err := proto.DecodeAuction(body, 12002, &r); err != nil || r.Sequence != 7 || r.NPC != 0 || r.Error == "" {
			t.Fatal("unexpected response", r, err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("missing response")
	}
}

func TestAuctionTradingAcrossMapsAndFactions(t *testing.T) {
	oldDefs, oldNPCs := db.MustDefs(), npcDefs
	t.Cleanup(func() { db.SetDefsForTest(oldDefs); npcDefs = oldNPCs })
	db.SetDefsForTest(&db.Defs{Items: map[db.ItemKey]*db.ItemDef{{Type: 25, TypeID: 1}: {Type: 25, TypeID: 1, Name: "Auction test stack", Count: 99}}})
	npcDefs[0] = []npcDef{{Type: 1, TypeID: 1, ShopType: proto.AuctionBoard}}
	makeSession := func(name string, zone uint16, faction uint8, npc uint32) (*Session, <-chan []byte) {
		account := testAccount(t)
		id, err := db.CreateCharacter(account, db.NewCharacter{Slot: 0, Name: name, HP: 100})
		if err != nil {
			t.Fatal(err)
		}
		ch, err := db.CharacterByID(id)
		if err != nil {
			t.Fatal(err)
		}
		ch.MapID, ch.PosX, ch.PosY, ch.PosZ, ch.Money = zone, 10, 2, 10, 3000
		if err = db.SaveCharacter(&ch); err != nil {
			t.Fatal(err)
		}
		AddEntities(zone, MapEntities{NPCs: []MapNpc{{ID: npc, Type: 1, TypeID: 1, PosX: 10, PosY: 2, PosZ: 10}}})
		s, frames := gmWireSession(t, ch, false)
		s.faction = faction
		s.dispatch(packet.NpcTalk, proto.NpcTalkPacket(npc))
		t.Cleanup(func() { s.useNpc(0) })
		return s, frames
	}
	seller, sellerFrames := makeSession("AuctionSeller", 42, FactionLight, 3500040)
	buyer, buyerFrames := makeSession("AuctionCustomer", 60422, FactionDark, 3500041)
	item := db.Item{Bag: 1, Slot: 0, Type: 25, TypeID: 1, Count: 5, Quality: 37, CraftName: "00000000000000000053"}
	if err := db.AddCharacterItem(seller.char.ID, &item); err != nil {
		t.Fatal(err)
	}
	seller.items = []db.Item{item}
	request := func(s *Session, ch <-chan []byte, r proto.AuctionRequest) proto.AuctionReply {
		t.Helper()
		r.NPC = s.usingNpc
		s.dispatch(packet.Opcode(proto.GXAuction), proto.AuctionPacket(r))
		timeout := time.After(3 * time.Second)
		for {
			select {
			case frame := <-ch:
				if proto.Op(frame) != proto.GXAuction {
					continue
				}
				var reply proto.AuctionReply
				if err := proto.DecodeAuction(frame, 12002, &reply); err != nil || reply.Error != "" {
					t.Fatal(reply, err)
				}
				return reply
			case <-timeout:
				t.Fatal("market did not respond to", r.Action)
			}
		}
	}
	post := proto.AuctionRequest{Action: "post", View: proto.AuctionSelling, Source: "test-post", Item: &proto.InventoryItem{Bag: 1, Type: 25, TypeID: 1, Count: 5, Quality: item.Quality, Craft: item.CraftName}, Count: 2, Start: 500, Buyout: 1000}
	list := request(seller, sellerFrames, post)
	if len(list.Rows) != 1 || len(seller.items) != 1 || seller.items[0].Count != 3 || seller.char.Money != 2965 {
		t.Fatal(list, seller.items, seller.char.Money)
	}
	filtered := request(buyer, buyerFrames, proto.AuctionRequest{Category: 5, Subcategory: 2})
	if len(filtered.Rows) != 0 {
		t.Fatal("consumable appeared under Mounts", filtered)
	}
	goddess := uint8(4)
	filtered = request(buyer, buyerFrames, proto.AuctionRequest{Faction: proto.AuctionFury, NameGrade: &goddess})
	if len(filtered.Rows) != 0 {
		t.Fatal("Normal item appeared under the Goddess grade", filtered)
	}
	normal := uint8(0)
	browse := request(buyer, buyerFrames, proto.AuctionRequest{
		Query: "test stack", Category: 5, Subcategory: 1, Faction: proto.AuctionFury, NameGrade: &normal,
	})
	if len(browse.Rows) != 1 || browse.Rows[0].ID != list.Rows[0].ID {
		t.Fatal(browse)
	}
	r := browse.Rows[0]
	request(buyer, buyerFrames, proto.AuctionRequest{Action: "buy", ID: r.ID, Version: r.Version, Amount: r.Buyout})
	results := request(buyer, buyerFrames, proto.AuctionRequest{View: proto.AuctionItems})
	if len(results.Rows) != 1 || results.Rows[0].Count != 2 || buyer.char.Money != 2000 {
		t.Fatal(results, buyer.char.Money)
	}
	request(buyer, buyerFrames, proto.AuctionRequest{Action: "collect", ID: results.Rows[0].ID, View: proto.AuctionItems})
	if len(buyer.items) != 1 || buyer.items[0].Count != 2 || buyer.items[0].UID == item.UID ||
		buyer.items[0].Quality != item.Quality || buyer.items[0].CraftName != item.CraftName {
		t.Fatal("split stack did not survive custody", buyer.items)
	}
	request(seller, sellerFrames, proto.AuctionRequest{Action: "collectall", View: proto.AuctionGold})
	if seller.char.Money != 3980 {
		t.Fatal("incorrect net seller settlement", seller.char.Money)
	}
}
