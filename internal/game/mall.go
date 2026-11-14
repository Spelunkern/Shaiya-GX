package game

import (
	"fmt"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

type MallPendingClaim struct {
	Gift        proto.MallStoredItem
	Destination proto.MallClaim
}
type MallState struct {
	Points                                  uint32
	PointsReady, StoredReady                bool
	Stored                                  map[uint8]proto.MallStoredItem
	LastPurchase                            proto.MallPurchase
	LastGift                                proto.MallGiftResult
	PurchaseSerial, GiftSerial, ClaimSerial uint64
	PendingClaim                            *MallPendingClaim
}

func (s *State) applyMall(p []byte) error {
	switch proto.Op(p) {
	case proto.CashPoint:
		points, err := proto.DecodeU32(p)
		if err != nil {
			return err
		}
		s.Mall.Points, s.Mall.PointsReady = points, true
	case proto.PointBuy:
		result, err := proto.DecodeMallPurchase(p)
		if err != nil {
			return err
		}
		if result.HasDetails {
			s.Mall.Points, s.Mall.PointsReady = result.Points, true
		}
		s.Mall.LastPurchase = result
		s.Mall.PurchaseSerial++
	case proto.PointGiftSend:
		result, err := proto.DecodeMallGiftResult(p)
		if err != nil {
			return err
		}
		if result.HasDetails {
			s.Mall.Points, s.Mall.PointsReady = result.Points, true
		}
		s.Mall.LastGift = result
		s.Mall.GiftSerial++
	case proto.PointStoredItemList:
		items, err := proto.DecodeMallStored(p)
		if err != nil {
			return err
		}
		stored := make(map[uint8]proto.MallStoredItem, len(items))
		for _, it := range items {
			stored[it.Slot] = it
		}
		s.Mall.Stored, s.Mall.StoredReady = stored, true
	case proto.PointStoredItemGet:
		claim, err := proto.DecodeMallClaim(p)
		if err != nil {
			return err
		}
		gift, ok := s.Mall.Stored[claim.Slot]
		if !ok {
			return fmt.Errorf("mall: claim for unknown gift slot %d", claim.Slot)
		}
		if s.Mall.PendingClaim != nil {
			return fmt.Errorf("mall: unresolved previous claim")
		}
		delete(s.Mall.Stored, claim.Slot)
		s.Mall.PendingClaim = &MallPendingClaim{Gift: gift, Destination: claim}
		s.Mall.ClaimSerial++
	}
	return nil
}

// 0x2607 contains only the destination and gift slot. Resolve its item
// attributes from the client's item catalog after the server confirms it.
func (s *State) ResolveMallClaim(quality uint16, maxStack uint16) bool {
	claim := s.Mall.PendingClaim
	if claim == nil || maxStack == 0 {
		return false
	}
	item := proto.InventoryItem{Bag: claim.Destination.Bag, Slot: claim.Destination.InventorySlot, Type: claim.Gift.Type, TypeID: claim.Gift.TypeID, Count: min(claim.Gift.Count, maxStack), Quality: quality}
	s.applyItem(item)
	s.Mall.PendingClaim = nil
	return true
}
