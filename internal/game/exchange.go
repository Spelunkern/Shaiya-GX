package game

import (
	"fmt"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

type ExchangeOwnOffer struct {
	Source [2]uint8
	Item   proto.ExchangeItem
}
type ExchangeState struct {
	InviteID                                           uint32
	InviteSerial, AnswerSerial, EndSerial              uint64
	Answer, EndResult                                  uint8
	Active                                             bool
	PartnerID                                          uint32
	Own                                                [proto.ExchangeSlots]ExchangeOwnOffer
	Other                                              [proto.ExchangeSlots]proto.ExchangeItem
	OwnMoney, OtherMoney                               uint32
	OwnReady, OtherReady, OwnConfirmed, OtherConfirmed bool
	PendingRemove                                      [proto.ExchangeSlots]bool
	Desync                                             bool
}

// MarkExchangeRemove records only the in-flight request, never the offered item.
// Rapture's removal acknowledgement carries a slot but no side; this identifies
// the local removal in the common nonconcurrent case.
func (s *State) MarkExchangeRemove(slot uint8) {
	if slot < proto.ExchangeSlots {
		s.Exchange.PendingRemove[slot] = true
	}
}
func (s *State) applyExchange(p []byte) error {
	x := &s.Exchange
	switch proto.Op(p) {
	case proto.ExchangeRequest:
		id, err := proto.DecodeExchangeID(p)
		if err != nil {
			return err
		}
		x.InviteID = id
		x.InviteSerial++
	case proto.ExchangeAnswer:
		value, err := proto.DecodeExchangeByte(p)
		if err != nil {
			return err
		}
		x.Answer = value
		x.AnswerSerial++
	case proto.ExchangeStart:
		id, result, err := proto.DecodeExchangeStart(p)
		if err != nil {
			return err
		}
		if result != 0 {
			return fmt.Errorf("exchange start result %d", result)
		}
		inviteSerial, answerSerial, endSerial := x.InviteSerial, x.AnswerSerial, x.EndSerial
		*x = ExchangeState{Active: true, PartnerID: id, InviteSerial: inviteSerial, AnswerSerial: answerSerial, EndSerial: endSerial}
	case proto.ExchangeEnd:
		value, err := proto.DecodeExchangeByte(p)
		if err != nil {
			return err
		}
		x.Active = false
		x.EndResult = value
		x.EndSerial++
		x.Own = [proto.ExchangeSlots]ExchangeOwnOffer{}
		x.Other = [proto.ExchangeSlots]proto.ExchangeItem{}
		x.OwnMoney, x.OtherMoney = 0, 0
		x.OwnReady, x.OtherReady, x.OwnConfirmed, x.OtherConfirmed = false, false, false, false
		x.PendingRemove = [proto.ExchangeSlots]bool{}
	case proto.ExchangeReady, proto.ExchangeConfirm:
		v, err := proto.DecodeExchangeUpdate(p)
		if err != nil {
			return err
		}
		if v.Kind < 1 || v.Kind > 3 || v.Value > 1 {
			return fmt.Errorf("invalid exchange update %d/%d", v.Kind, v.Value)
		}
		if proto.Op(p) == proto.ExchangeReady {
			if v.Kind&1 != 0 {
				x.OwnReady = v.Value == 0
			}
			if v.Kind&2 != 0 {
				x.OtherReady = v.Value == 0
			}
		} else {
			if v.Kind&1 != 0 {
				x.OwnConfirmed = v.Value != 0
			}
			if v.Kind&2 != 0 {
				x.OtherConfirmed = v.Value != 0
			}
		}
	case proto.ExchangeAddMoney:
		v, err := proto.DecodeExchangeMoney(p)
		if err != nil {
			return err
		}
		switch v.Kind {
		case 1:
			x.OwnMoney = v.Money
		case 2:
			x.OtherMoney = v.Money
		default:
			return fmt.Errorf("invalid exchange money side %d", v.Kind)
		}
	case proto.ExchangeAdd:
		v, err := proto.DecodeExchangeOwnItem(p)
		if err != nil {
			return err
		}
		it, ok := s.Inventory[[2]uint8{v.Bag, v.Slot}]
		if !ok {
			return fmt.Errorf("exchange offered unknown item %d/%d", v.Bag, v.Slot)
		}
		x.Own[v.Destination] = ExchangeOwnOffer{Source: [2]uint8{v.Bag, v.Slot}, Item: proto.ExchangeItem{Type: it.Type, TypeID: it.TypeID, Count: v.Count, Quality: it.Quality, Gems: it.Gems, Craft: it.Craft}}
	case proto.ExchangeAddDest:
		slot, it, err := proto.DecodeExchangePartnerItem(p)
		if err != nil {
			return err
		}
		if it.Type == 0 && x.PendingRemove[slot] && x.Other[slot].Type != 0 {
			// The packet does not identify which player removed the item.
			// Preserve both offers and cancel instead of guessing.
			x.Desync = true
		} else if it.Type == 0 && x.PendingRemove[slot] {
			x.Own[slot] = ExchangeOwnOffer{}
			x.PendingRemove[slot] = false
		} else {
			x.Other[slot] = it
		}
	}
	return nil
}
