package db

import (
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

func TestAuctionFiltersBeforePaging(t *testing.T) {
	p := sqliteFixture(t)
	// Use the stock Item definitions for Light/Fury Goddess weapons, a
	// neutral Normal weapon, an Ancient hat, and faction-exempt type 30.
	for _, fixture := range []struct {
		itemType, typeID uint8
		listings         int
	}{{1, 65, 1}, {3, 65, 10}, {9, 1, 1}, {16, 69, 1}, {30, 189, 1}} {
		item, err := json.Marshal(Item{Type: fixture.itemType, TypeID: fixture.typeID, Count: 1, Quality: 999})
		if err != nil {
			t.Fatal(err)
		}
		for i := range fixture.listings {
			// Each listing belongs to a different seller, so the posting key is unique.
			result, err := p.Exec(`INSERT INTO GX_Auctions(Seller,Request,SellerName,Name,Category,ItemType,TypeID,Level,Item,Price,Buyout,Deposit,Ends)
				SELECT ?, 'filter', 'Filter seller', ItemName, 0, Type, TypeID, ReqLevel, ?, 500, 1000, 35, ?
				FROM PS_GameDefs_Items WHERE ItemID=?`,
				int(fixture.itemType)*100+i, item, time.Now().Unix()+3600, int(fixture.itemType)*1000+int(fixture.typeID))
			if err != nil {
				t.Fatal(err)
			}
			if err := requireOne(result); err != nil {
				t.Fatalf("stock fixture %d/%d: %v", fixture.itemType, fixture.typeID, err)
			}
		}
	}
	normal, goddess, ancient, unsupported := uint8(0), uint8(4), uint8(7), uint8(6)
	for _, tc := range []struct {
		name string
		req  proto.AuctionRequest
		want uint8
		rows int
		more bool
	}{
		{"Light Goddess", proto.AuctionRequest{Faction: proto.AuctionLight, NameGrade: &goddess}, 1, 1, false},
		{"Fury Goddess first page", proto.AuctionRequest{Faction: proto.AuctionFury, NameGrade: &goddess}, 3, 8, true},
		{"Fury Goddess second page", proto.AuctionRequest{Faction: proto.AuctionFury, NameGrade: &goddess, Page: 1}, 3, 2, false},
		{"Ancient", proto.AuctionRequest{NameGrade: &ancient}, 16, 1, false},
		{"Fury Normal includes Lapis and neutral", proto.AuctionRequest{Faction: proto.AuctionFury, NameGrade: &normal}, 0, 2, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var req proto.AuctionRequest
			if err := proto.DecodeAuction(proto.AuctionPacket(tc.req), 1024, &req); err != nil {
				t.Fatal(err)
			}
			reply, err := ListAuctions(1, req)
			if err != nil || len(reply.Rows) != tc.rows || reply.More != tc.more {
				t.Fatalf("got %+v, %v", reply, err)
			}
			for _, row := range reply.Rows {
				if tc.want != 0 && row.Type != tc.want {
					t.Fatalf("unexpected match %d/%d", row.Type, row.TypeID)
				}
			}
			if tc.want == 0 && (reply.Rows[0].Type != 30 || reply.Rows[1].Type != 9) {
				t.Fatalf("expected Lapis and the neutral weapon, got %+v", reply.Rows)
			}
		})
	}
	if reply, err := ListAuctions(1, proto.AuctionRequest{}); err != nil || len(reply.Rows) != proto.AuctionPageSize || !reply.More {
		t.Fatalf("unfiltered request %+v, %v", reply, err)
	}
	for _, invalid := range []proto.AuctionRequest{{Faction: 3}, {NameGrade: &unsupported}} {
		if _, err := ListAuctions(1, invalid); !errors.Is(err, ErrAuctionInvalid) {
			t.Fatalf("invalid selectors returned %v", err)
		}
	}
}

func auctionBuyer(t *testing.T, ch Character, slot uint8, name string) Character {
	t.Helper()
	id, err := CreateCharacter(Account{UserUID: int32(ch.AccountID)}, NewCharacter{Slot: slot, Name: name, HP: 100})
	if err != nil {
		t.Fatal(err)
	}
	buyer, err := CharacterByID(id)
	if err != nil {
		t.Fatal(err)
	}
	buyer.Money = 5000
	if err := SaveCharacter(&buyer); err != nil {
		t.Fatal(err)
	}
	return buyer
}

func TestAuctionSettlementAndClaimAtomicity(t *testing.T) {
	p := sqliteFixture(t)
	seller, item := economyCharacter(t)
	bidder := auctionBuyer(t, seller, 1, "AuctionBidder")
	buyer := auctionBuyer(t, seller, 2, "AuctionBuyer")
	item.Gems[0], item.CraftName = 11, "00000000000000000053"
	if err := SaveCharacterItem(seller.ID, &item); err != nil {
		t.Fatal(err)
	}
	deposit := proto.AuctionDeposit(500, 0)
	listing := AuctionListing{Seller: seller.ID, SellerName: seller.Name, Name: "Smith's Blade", Item: item, Price: 500, Buyout: 1200, Deposit: deposit, Ends: time.Now().Unix() + 3600, Request: "first", Category: 1}
	post := CharacterChange{Before: seller, After: seller, ItemsBefore: []Item{item}}
	post.After.Money -= deposit
	if _, err := p.Exec(`CREATE TRIGGER fail_auction BEFORE INSERT ON GX_Auctions BEGIN SELECT RAISE(ABORT,'injected'); END`); err != nil {
		t.Fatal(err)
	}
	if err := PostAuction(listing, post); err == nil {
		t.Fatal("injected insert succeeded")
	}
	items, err := CharacterItems(seller.ID)
	if err != nil || !sameInventory(items, []Item{item}) {
		t.Fatal("posting rollback lost item", items, err)
	}
	if _, err = p.Exec("DROP TRIGGER fail_auction"); err != nil {
		t.Fatal(err)
	}
	if err = PostAuction(listing, post); err != nil {
		t.Fatal(err)
	}
	if err = PostAuction(listing, post); err == nil {
		t.Fatal("duplicate post accepted")
	}
	search, err := ListAuctions(buyer.ID, proto.AuctionRequest{Query: "Smith's", Category: 1})
	if err != nil || len(search.Rows) != 1 || search.Rows[0].Mine || search.Rows[0].Gems != item.Gems {
		t.Fatal(search, err)
	}
	row := search.Rows[0]
	if err = ChangeAuction(bidder.ID, proto.AuctionRequest{Action: "watch", ID: row.ID, Version: row.Version}, nil); err != nil {
		t.Fatal(err)
	}
	watched, err := ListAuctions(bidder.ID, proto.AuctionRequest{View: proto.AuctionWatch})
	if err != nil || len(watched.Rows) != 1 {
		t.Fatal(watched, err)
	}
	bid := CharacterChange{Before: bidder, After: bidder}
	bid.After.Money -= 550
	if err = ChangeAuction(bidder.ID, proto.AuctionRequest{Action: "bid", ID: row.ID, Version: row.Version, Amount: 550}, &bid); err != nil {
		t.Fatal(err)
	}

	payment := CharacterChange{Before: buyer, After: buyer}
	payment.After.Money -= 1200
	buy := proto.AuctionRequest{Action: "buy", ID: row.ID, Version: row.Version, Amount: 1200}
	if err = ChangeAuction(buyer.ID, buy, &payment); !errors.Is(err, ErrAuctionChanged) {
		t.Fatal("stale buy", err)
	}
	buy.Version++
	var wg sync.WaitGroup
	out := make(chan error, 2)
	for range 2 {
		wg.Add(1)
		go func() { defer wg.Done(); out <- ChangeAuction(buyer.ID, buy, &payment) }()
	}
	wg.Wait()
	if a, b := <-out, <-out; (a == nil) == (b == nil) {
		t.Fatal("expected exactly one purchase", a, b)
	}
	for owner, want := range map[uint32]uint32{seller.ID: 1200 + deposit - 24, bidder.ID: 550} {
		claims, err := AuctionClaims(owner, proto.AuctionGold, 0)
		if err != nil || len(claims) != 1 || claims[0].Gold != want {
			t.Fatal(owner, claims, err)
		}
	}
	claims, err := AuctionClaims(buyer.ID, proto.AuctionItems, 0)
	if err != nil || len(claims) != 1 || !sameItemState(claims[0].Item, item) {
		t.Fatal(claims, err)
	}
	claim := claims[0]
	receive := CharacterChange{Before: payment.After, After: payment.After, ItemsAfter: []Item{item}}
	if _, err = p.Exec(`CREATE TRIGGER fail_auction_claim BEFORE INSERT ON PS_GameData_CharItems BEGIN SELECT RAISE(ABORT,'injected'); END`); err != nil {
		t.Fatal(err)
	}
	if err = CollectAuction(claim, receive); err == nil {
		t.Fatal("injected claim succeeded")
	}
	if _, err = p.Exec("DROP TRIGGER fail_auction_claim"); err != nil {
		t.Fatal(err)
	}
	if err = CollectAuction(claim, receive); err != nil {
		t.Fatal(err)
	}
	if err = CollectAuction(claim, receive); !errors.Is(err, ErrAuctionChanged) {
		t.Fatal("claim replay", err)
	}
	items, err = CharacterItems(buyer.ID)
	if err != nil || !sameInventory(items, []Item{item}) {
		t.Fatal(items, err)
	}
}

func TestAuctionExpiryIsIdempotent(t *testing.T) {
	p := sqliteFixture(t)
	seller, item := economyCharacter(t)
	post := CharacterChange{Before: seller, After: seller, ItemsBefore: []Item{item}}
	post.After.Money -= 35
	if err := PostAuction(AuctionListing{Seller: seller.ID, Item: item, Price: 500, Buyout: 1000, Deposit: 35, Ends: time.Now().Unix() - 1, Request: "expired"}, post); err != nil {
		t.Fatal(err)
	}
	if err := ExpireAuctions(); err != nil {
		t.Fatal(err)
	}
	if err := ExpireAuctions(); err != nil {
		t.Fatal(err)
	}
	claims, err := AuctionClaims(seller.ID, proto.AuctionItems, 0)
	if err != nil || len(claims) != 1 {
		t.Fatal(claims, err)
	}
	var state string
	if err = p.QueryRow("SELECT State FROM GX_Auctions").Scan(&state); err != nil || state != "Expired" {
		t.Fatal(state, err)
	}
}
