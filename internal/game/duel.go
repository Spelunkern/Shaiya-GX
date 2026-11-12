package game

import (
	"fmt"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

type DuelPhase uint8

const (
	DuelNone DuelPhase = iota
	DuelOutgoing
	DuelIncoming
	DuelWager
	DuelCounting
	DuelFighting
)

type DuelState struct {
	Phase                       DuelPhase
	Opponent                    uint32
	RequestSerial, ResultSerial uint64
	Result                      proto.DuelResponse
	CenterX, CenterZ            float32
	Own                         [proto.ExchangeSlots]ExchangeOwnOffer
	Other                       [proto.ExchangeSlots]proto.ExchangeItem
	OwnMoney, OtherMoney        uint32
	OwnReady, OtherReady        bool
	Outcome, WindowResult       uint8
}

func (s *State) applyDuel(p []byte) error {
	d := &s.Duel
	switch proto.Op(p) {
	case proto.DuelRequest:
		req, err := proto.DecodeDuelChallenge(p)
		if err != nil {
			return err
		}
		if req.Requester != s.SelfID && req.Target != s.SelfID {
			return fmt.Errorf("duel request not addressed to self")
		}
		d.RequestSerial++
		if req.Requester == s.SelfID {
			d.Phase, d.Opponent = DuelOutgoing, req.Target
		} else {
			d.Phase, d.Opponent = DuelIncoming, req.Requester
		}
	case proto.DuelAnswer, proto.DuelEnd:
		res, err := proto.DecodeDuelResponse(p)
		if err != nil {
			return err
		}
		d.Result = res
		d.ResultSerial++
		if proto.Op(p) == proto.DuelEnd || res.Result != 1 {
			d.Phase = DuelNone
		}
	case proto.DuelWindow:
		id, result, err := proto.DecodeExchangeStart(p)
		if err != nil {
			return err
		}
		if result != 0 {
			return fmt.Errorf("duel window result %d", result)
		}
		serial, resultSerial := d.RequestSerial, d.ResultSerial
		*d = DuelState{Phase: DuelWager, Opponent: id, RequestSerial: serial, ResultSerial: resultSerial}
	case proto.DuelCountdown:
		x, z, err := proto.DecodeDuelCenter(p)
		if err != nil {
			return err
		}
		d.CenterX, d.CenterZ, d.Phase = x, z, DuelCounting
	case proto.DuelStart:
		if err := proto.DecodeDuelEmpty(p); err != nil {
			return err
		}
		d.Phase = DuelFighting
	case proto.DuelResult:
		value, err := proto.DecodeExchangeByte(p)
		if err != nil {
			return err
		}
		d.Outcome, d.Phase = value, DuelNone
		d.ResultSerial++
	case proto.DuelWindowEnd:
		value, err := proto.DecodeExchangeByte(p)
		if err != nil {
			return err
		}
		d.WindowResult = value
		if value != 0 {
			d.Phase = DuelNone
		}
	case proto.DuelReady:
		u, err := proto.DecodeExchangeUpdate(p)
		if err != nil {
			return err
		}
		if u.Kind < 1 || u.Kind > 3 || u.Value > 1 {
			return fmt.Errorf("invalid duel readiness %d/%d", u.Kind, u.Value)
		}
		if u.Kind&1 != 0 {
			d.OwnReady = u.Value == 0
		}
		if u.Kind&2 != 0 {
			d.OtherReady = u.Value == 0
		}
	case proto.DuelMoney:
		u, err := proto.DecodeExchangeMoney(p)
		if err != nil {
			return err
		}
		switch u.Kind {
		case 1:
			d.OwnMoney = u.Money
		case 2:
			d.OtherMoney = u.Money
		default:
			return fmt.Errorf("invalid duel money side %d", u.Kind)
		}
	case proto.DuelAdd:
		v, err := proto.DecodeExchangeOwnItem(p)
		if err != nil {
			return err
		}
		it, ok := s.Inventory[[2]uint8{v.Bag, v.Slot}]
		if !ok {
			return fmt.Errorf("duel offered unknown item %d/%d", v.Bag, v.Slot)
		}
		d.Own[v.Destination] = ExchangeOwnOffer{Source: [2]uint8{v.Bag, v.Slot}, Item: proto.ExchangeItem{Type: it.Type, TypeID: it.TypeID, Count: v.Count, Quality: it.Quality, Gems: it.Gems, Craft: it.Craft}}
	case proto.DuelAddOther:
		slot, it, err := proto.DecodeExchangePartnerItem(p)
		if err != nil {
			return err
		}
		d.Other[slot] = it
	case proto.DuelRemove:
		u, err := proto.DecodeExchangeUpdate(p)
		if err != nil {
			return err
		}
		if u.Value >= proto.ExchangeSlots {
			return fmt.Errorf("invalid duel wager slot %d", u.Value)
		}
		switch u.Kind {
		case 1:
			d.Own[u.Value] = ExchangeOwnOffer{}
		case 2:
			d.Other[u.Value] = proto.ExchangeItem{}
		default:
			return fmt.Errorf("invalid duel removal side %d", u.Kind)
		}
	}
	return nil
}
