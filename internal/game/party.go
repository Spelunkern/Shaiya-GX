package game

import (
	"fmt"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

type PartyState struct {
	Members         []proto.PartyMember
	Leader          uint32
	InviteID        uint32
	InviteName      string
	InviteSerial    uint64
	Reply           proto.PartyReply
	ReplySerial     uint64
	Raid            bool
	RaidInvite      bool
	RaidSlots       map[uint16]uint32
	RaidSubLeader   uint32
	RaidDivision    uint32
	RaidAutoJoin    bool
	RaidError       uint32
	RaidErrorSerial uint64
	raidPending     *raidRoster
}

func (p *PartyState) Member(id uint32) *proto.PartyMember {
	for i := range p.Members {
		if p.Members[i].ID == id {
			return &p.Members[i]
		}
	}
	return nil
}
func (s *State) SelfPartyMember() proto.PartyMember {
	m := proto.PartyMember{ID: s.SelfID, HP: s.Pools.HP, MP: s.Pools.MP, SP: s.Pools.SP, MaxHP: s.Details.MaxHP, MaxMP: s.Details.MaxMP, MaxSP: s.Details.MaxSP, X: s.Self.X, Y: s.Self.Y, Z: s.Self.Z}
	for _, c := range s.Slots {
		if c.ID == s.SelfID {
			m.Name, m.Class, m.Level = c.Name, c.Class, c.Level
			break
		}
	}
	return m
}
func (s *State) applyParty(p []byte) error {
	if handled, err := s.applyRaid(p); handled {
		return err
	}
	switch proto.Op(p) {
	case proto.PartyRoster:
		list, err := proto.DecodePartyList(p)
		if err != nil {
			return err
		}
		leader := s.SelfID
		if len(list.Members) > 0 {
			if int(list.LeaderIndex) >= len(list.Members) {
				return fmt.Errorf("party: invalid leader index %d", list.LeaderIndex)
			}
			leader = list.Members[list.LeaderIndex].ID
		}
		seen := map[uint32]bool{}
		for _, m := range list.Members {
			if m.ID == 0 || seen[m.ID] {
				return fmt.Errorf("party: invalid or duplicate member %d", m.ID)
			}
			seen[m.ID] = true
		}
		members := list.Members
		// Rapture enterParty sends the roster before appending the entering self.
		if !seen[s.SelfID] && s.SelfID != 0 {
			members = append(members, s.SelfPartyMember())
		}
		s.Party.Members, s.Party.Leader = members, leader
		s.Party.InviteID, s.Party.InviteName = 0, ""
	case proto.PartyJoined:
		m, err := proto.DecodePartyJoined(p)
		if err != nil {
			return err
		}
		if m.ID == 0 {
			return fmt.Errorf("party: zero member ID")
		}
		if old := s.Party.Member(m.ID); old != nil {
			*old = m
		} else {
			s.Party.Members = append(s.Party.Members, m)
		}
	case proto.PartyInvite:
		id, err := proto.DecodeU32(p)
		if err != nil {
			return err
		}
		s.Party.InviteID, s.Party.InviteName = id, ""
		s.Party.RaidInvite = false
		if e := s.Entities[id]; e != nil && e.Shape != nil {
			s.Party.InviteName = e.Shape.Name
		}
		s.Party.InviteSerial++
	case proto.PartyInviteName:
		name, err := proto.DecodePartyInviteName(p)
		if err != nil {
			return err
		}
		s.Party.InviteID, s.Party.InviteName = 0, name
		s.Party.RaidInvite = false
		for id, e := range s.Entities {
			if e.Shape != nil && e.Shape.Name == name {
				s.Party.InviteID = id
				break
			}
		}
		s.Party.InviteSerial++
	case proto.PartyAnswer:
		reply, err := proto.DecodePartyReply(p)
		if err != nil {
			return err
		}
		s.Party.Reply = reply
		s.Party.ReplySerial++
	case proto.PartyLeft, proto.PartyExpelled:
		id, err := proto.DecodeU32(p)
		if err != nil {
			return err
		}
		if id == s.SelfID {
			s.Party.Members, s.Party.Leader = nil, 0
			return nil
		}
		for i, m := range s.Party.Members {
			if m.ID == id {
				s.Party.Members = append(s.Party.Members[:i], s.Party.Members[i+1:]...)
				break
			}
		}
	case proto.PartyLeader:
		id, err := proto.DecodeU32(p)
		if err != nil {
			return err
		}
		if id != 0 && s.Party.Member(id) != nil {
			s.Party.Leader = id
		}
	case proto.PartyBadge:
		r := proto.NewReader(p)
		id, kind := r.U32(), r.U8()
		if err := r.Err(); err != nil {
			return err
		}
		if e := s.Entities[id]; e != nil {
			e.PartyBadge = kind
		}
		if id == s.SelfID {
			s.Self.PartyBadge = kind
		}
	case proto.PartyPools:
		r := proto.NewReader(p)
		id, hp, sp, mp := r.U32(), r.U32(), r.U32(), r.U32()
		if err := r.Err(); err != nil {
			return err
		}
		if m := s.Party.Member(id); m != nil {
			m.HP, m.SP, m.MP = hp, sp, mp
		}
	case proto.PartyLevel:
		r := proto.NewReader(p)
		id, level := r.U32(), r.U16()
		if err := r.Err(); err != nil {
			return err
		}
		if m := s.Party.Member(id); m != nil {
			m.Level = level
		}
	}
	return nil
}
