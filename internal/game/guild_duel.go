package game

import (
	"fmt"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

type GuildDuelState struct {
	Phase                                    DuelPhase
	Opponent                                 uint32
	OpponentGuild                            uint32
	CenterX, CenterZ                         float32
	Active                                   bool
	RequestSerial, ResultSerial, LeaveSerial uint64
	Response                                 proto.DuelResponse
	Outcome                                  uint8
	LastLeave                                proto.DuelResponse
	deathLeavePending                        bool
}

func (s *State) applyGuildDuel(p []byte) error {
	d := &s.GuildDuel
	switch proto.Op(p) {
	case proto.GuildDuelRequest:
		request, err := proto.DecodeDuelChallenge(p)
		if err != nil {
			return err
		}
		if request.Requester != s.SelfID && request.Target != s.SelfID {
			return fmt.Errorf("guild duel request not addressed to self")
		}
		d.RequestSerial++
		d.deathLeavePending = false
		d.Active = false
		if request.Requester == s.SelfID {
			d.Phase, d.Opponent = DuelOutgoing, request.Target
		} else {
			d.Phase, d.Opponent = DuelIncoming, request.Requester
		}
	case proto.GuildDuelAnswer:
		response, err := proto.DecodeDuelResponse(p)
		if err != nil {
			return err
		}
		d.Response = response
		d.ResultSerial++
		if response.Result != 1 {
			d.Phase = DuelNone
			d.Active = false
		}
	case proto.GuildDuelCountdown:
		center, err := proto.DecodeGuildDuelCenter(p)
		if err != nil {
			return err
		}
		d.CenterX, d.CenterZ, d.OpponentGuild = center.X, center.Z, center.OpponentGuild
		d.Phase, d.Active = DuelCounting, true
	case proto.GuildDuelStart:
		if err := proto.DecodeDuelEmpty(p); err != nil {
			return err
		}
		d.Phase = DuelFighting
	case proto.GuildDuelResult:
		result, err := proto.DecodeU8(p)
		if err != nil {
			return err
		}
		d.Outcome, d.Phase, d.Active = result, DuelNone, false
		d.ResultSerial++
	case proto.GuildDuelLeave:
		leave, err := proto.DecodeDuelResponse(p)
		if err != nil {
			return err
		}
		d.LastLeave = leave
		d.LeaveSerial++
		if leave.Player == s.SelfID {
			// Rapture sends this before 0x0504 only for a death caused by
			// an opponent in the guild duel. Other deaths send it after.
			if leave.Result == 3 && d.Active && !s.Self.Dead {
				d.deathLeavePending = true
			}
			d.Active = false
		}
	}
	return nil
}
