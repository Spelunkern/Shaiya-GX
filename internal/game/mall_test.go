package game

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

func TestMallStateUsesConfirmedPackets(t *testing.T) {
	s := New()
	points := proto.NewWriter(proto.CashPoint)
	points.U32(100)
	if _, err := s.Apply(points.Bytes()); err != nil {
		t.Fatal(err)
	}
	list := proto.NewWriter(proto.PointStoredItemList)
	list.U8(2)
	list.Raw([]byte{5, 25, 4, 7, 12, 1, 3, 1})
	if _, err := s.Apply(list.Bytes()); err != nil {
		t.Fatal(err)
	}
	if !s.Mall.PointsReady || s.Mall.Points != 100 || !s.Mall.StoredReady || len(s.Mall.Stored) != 2 {
		t.Fatal(s.Mall)
	}
	claim := proto.NewWriter(proto.PointStoredItemGet)
	claim.Raw([]byte{5, 2, 3})
	if _, err := s.Apply(claim.Bytes()); err != nil {
		t.Fatal(err)
	}
	if _, ok := s.Mall.Stored[5]; ok || s.Mall.PendingClaim == nil || len(s.Inventory) != 0 {
		t.Fatal("claim applied before item metadata", s.Mall)
	}
	if !s.ResolveMallClaim(44, 5) {
		t.Fatal("confirmed claim did not resolve")
	}
	if got := s.Inventory[[2]uint8{2, 3}]; got.Type != 25 || got.TypeID != 4 || got.Count != 5 || got.Quality != 44 {
		t.Fatal("claimed item", got)
	}
	bad := proto.NewWriter(proto.PointStoredItemList)
	bad.U8(2)
	bad.Raw([]byte{2, 25, 1, 1, 2, 25, 2, 1})
	if _, err := s.Apply(bad.Bytes()); err == nil {
		t.Fatal("duplicate gift slot accepted")
	}
	if len(s.Mall.Stored) != 1 || s.Mall.Stored[12].Type != 1 {
		t.Fatal("bad gift snapshot modified state")
	}
	buy := proto.NewWriter(proto.PointBuy)
	buy.U8(2)
	if _, err := s.Apply(buy.Bytes()); err != nil {
		t.Fatal(err)
	}
	if s.Mall.Points != 100 || s.Mall.LastPurchase.Result != 2 || s.Mall.PurchaseSerial != 1 {
		t.Fatal("short rejection changed points", s.Mall)
	}
}
