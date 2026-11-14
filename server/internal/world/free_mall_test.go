package world

import (
	"fmt"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"testing"
	"time"
)

func TestFreeMallPurchaseAndGiftWithZeroBalance(t *testing.T) {
	old := db.MustDefs()
	t.Cleanup(func() { db.SetDefsForTest(old) })
	defs, err := db.ReloadDefsForTest()
	if err != nil {
		t.Fatal(err)
	}
	testDefs := *defs
	testDefs.Products = map[string]*db.Product{"FREE_TEST": {Code: "FREE_TEST", Name: "Free testing rune", Items: []db.ProductItem{{Type: 100, TypeID: 171, Count: 255}}}}
	db.SetDefsForTest(&testDefs)
	pool, err := db.Pool(db.GameData)
	if err != nil {
		t.Fatal(err)
	}
	for uid := 991101; uid <= 991102; uid++ {
		_, err = pool.Exec(`INSERT INTO PS_UserData_Users_Master(UserUID,UserID,Pw,Admin,AdminLevel,UseQueue,Status,Leave,UserType,Point) VALUES(?1,?2,'local',0,0,0,0,0,'N',0)`, uid, fmt.Sprintf("free_mall_%d", uid))
		if err != nil {
			t.Fatal(err)
		}
	}
	create := func(uid uint32, name string) uint32 {
		t.Helper()
		id, err := db.CreateCharacter(db.Account{UserUID: int32(uid), UserID: fmt.Sprintf("free_mall_%d", uid)}, db.NewCharacter{Slot: 0, Name: name, MapID: 1, HP: 100, MP: 100, SP: 100})
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	sender := create(991101, "FreeMallSender")
	create(991102, "FreeMallRecipient")
	s, packets := gmWireSession(t, db.Character{ID: sender, AccountID: 991101, CurrentHP: 100}, false)
	s.accountID = 991101
	s.points = 0
	next := func(op proto.Opcode) []byte {
		t.Helper()
		timeout := time.After(3 * time.Second)
		for {
			select {
			case p := <-packets:
				if proto.Op(p) == op {
					return p
				}
			case <-timeout:
				t.Fatal("missing free mall response", op)
				return nil
			}
		}
	}
	s.handlePointBuy(proto.MallBuyPacket("FREE_TEST"))
	purchase, err := proto.DecodeMallPurchase(next(proto.PointBuy))
	if err != nil || purchase.Result != pointBuyOK || purchase.Cost != 0 || s.points != 0 || len(s.items) != 1 || s.items[0].Count != 255 {
		t.Fatal("zero-point purchase failed", purchase, s.items, err)
	}
	s.handleGiftSend(proto.MallGiftPacket("FREE_TEST", "FreeMallRecipient"))
	gift, err := proto.DecodeMallGiftResult(next(proto.PointGiftSend))
	if err != nil || gift.Result != db.GiftOK || gift.Cost != 0 || s.points != 0 {
		t.Fatal("zero-point gift failed", gift, err)
	}
	gifts, err := db.GiftBox(991102)
	if err != nil || len(gifts) != 1 || gifts[0].Type != 100 || gifts[0].TypeID != 171 || gifts[0].Count != 255 {
		t.Fatal("gift contents", gifts, err)
	}
	var points uint32
	if err = pool.QueryRow(`SELECT Point FROM PS_UserData_Users_Master WHERE UserUID=991101`).Scan(&points); err != nil || points != 0 {
		t.Fatal("free purchase/gift changed balance", points, err)
	}
	t.Log("zero-point purchase and gift granted the expected rune stacks without charging")
}
