package game

import (
	"fmt"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

type raidRoster struct {
	leader, sub uint8
	division    uint16
	kind        uint8
	auto        bool
	members     []proto.PartyMember
	slots       map[uint16]uint32
}

func (s *State) applyRaid(p []byte) (bool, error) {
	op, party := proto.Op(p), &s.Party
	switch op {
	case proto.RaidRoster:
		chunk, err := proto.DecodeRaidRoster(p)
		if err != nil {
			party.raidPending = nil
			return true, err
		}
		pending := party.raidPending
		if pending == nil {
			pending = &raidRoster{leader: chunk.LeaderSlot, sub: chunk.SubLeaderSlot, division: chunk.Division, kind: chunk.Kind, auto: chunk.AutoJoin, slots: map[uint16]uint32{}}
		} else if pending.leader != chunk.LeaderSlot || pending.sub != chunk.SubLeaderSlot || pending.division != chunk.Division || pending.kind != chunk.Kind || pending.auto != chunk.AutoJoin {
			party.raidPending = nil
			return true, fmt.Errorf("raid: inconsistent roster chunks")
		}
		if pending.kind != 1 || len(pending.members)+len(chunk.Members) > 30 {
			party.raidPending = nil
			return true, fmt.Errorf("raid: invalid roster")
		}
		for _, entry := range chunk.Members {
			if entry.Slot >= 30 || entry.Member.ID == 0 || pending.slots[entry.Slot] != 0 {
				party.raidPending = nil
				return true, fmt.Errorf("raid: invalid slot %d", entry.Slot)
			}
			for _, m := range pending.members {
				if m.ID == entry.Member.ID {
					party.raidPending = nil
					return true, fmt.Errorf("raid: duplicate member %d", m.ID)
				}
			}
			pending.slots[entry.Slot] = entry.Member.ID
			pending.members = append(pending.members, entry.Member)
		}
		if chunk.More {
			party.raidPending = pending
			return true, nil
		}
		leader, sub := pending.slots[uint16(pending.leader)], pending.slots[uint16(pending.sub)]
		if leader == 0 || sub == 0 {
			party.raidPending = nil
			return true, fmt.Errorf("raid: missing leadership")
		}
		party.Members, party.RaidSlots = pending.members, pending.slots
		party.Raid, party.Leader, party.RaidSubLeader = true, leader, sub
		party.RaidDivision, party.RaidAutoJoin = uint32(pending.division), pending.auto
		party.InviteID, party.InviteName, party.RaidInvite, party.raidPending = 0, "", false, nil
		return true, nil
	case proto.RaidJoined:
		m, err := proto.DecodeRaidJoined(p)
		if err != nil {
			return true, err
		}
		if m.Slot >= 30 || m.Member.ID == 0 {
			return true, fmt.Errorf("raid: invalid joining member")
		}
		if party.RaidSlots == nil {
			party.RaidSlots = map[uint16]uint32{}
		}
		if old := party.RaidSlots[m.Slot]; old != 0 && old != m.Member.ID {
			return true, fmt.Errorf("raid: occupied slot")
		}
		for slot, id := range party.RaidSlots {
			if id == m.Member.ID && slot != m.Slot {
				delete(party.RaidSlots, slot)
			}
		}
		party.RaidSlots[m.Slot] = m.Member.ID
		if old := party.Member(m.Member.ID); old != nil {
			*old = m.Member
		} else {
			party.Members = append(party.Members, m.Member)
		}
		return true, nil
	case proto.RaidConvert:
		r := proto.NewReader(p)
		kind, auto, div, leader, sub := r.U8(), r.U8(), r.U32(), r.U32(), r.U32()
		if err := r.Err(); err != nil {
			return true, err
		}
		if kind != 1 || leader >= uint32(len(party.Members)) || sub >= uint32(len(party.Members)) || len(party.Members) > 30 {
			return true, fmt.Errorf("raid: invalid conversion")
		}
		slots := make(map[uint16]uint32, len(party.Members))
		for i, m := range party.Members {
			slots[uint16(i)] = m.ID
		}
		party.Raid, party.RaidSlots, party.RaidDivision, party.RaidAutoJoin = true, slots, div, auto != 0
		party.Leader, party.RaidSubLeader = slots[uint16(leader)], slots[uint16(sub)]
		return true, nil
	case proto.RaidDivision, proto.RaidAutoJoin:
		r := proto.NewReader(p)
		if op == proto.RaidDivision {
			div := r.U32()
			if err := r.Err(); err != nil {
				return true, err
			}
			party.RaidDivision = div
		} else {
			auto := r.U8()
			if err := r.Err(); err != nil {
				return true, err
			}
			party.RaidAutoJoin = auto != 0
		}
		return true, nil
	case proto.RaidSwap:
		r := proto.NewReader(p)
		first, second, leader, sub := r.U32(), r.U32(), r.U32(), r.U32()
		if err := r.Err(); err != nil {
			return true, err
		}
		if first >= 30 || second >= 30 || leader >= 30 || sub >= 30 || !party.Raid || party.RaidSlots == nil {
			return true, fmt.Errorf("raid: invalid swap")
		}
		a, b := party.RaidSlots[uint16(first)], party.RaidSlots[uint16(second)]
		valueAt := func(slot uint32) uint32 {
			if slot == first {
				return b
			}
			if slot == second {
				return a
			}
			return party.RaidSlots[uint16(slot)]
		}
		if valueAt(leader) == 0 || valueAt(sub) == 0 {
			return true, fmt.Errorf("raid: swap lost leadership")
		}
		delete(party.RaidSlots, uint16(first))
		delete(party.RaidSlots, uint16(second))
		if a != 0 {
			party.RaidSlots[uint16(second)] = a
		}
		if b != 0 {
			party.RaidSlots[uint16(first)] = b
		}
		party.Leader, party.RaidSubLeader = party.RaidSlots[uint16(leader)], party.RaidSlots[uint16(sub)]
		return true, nil
	case proto.RaidSubLeader, proto.RaidLeader:
		id, err := proto.DecodeU32(p)
		if err != nil {
			return true, err
		}
		if party.Member(id) != nil {
			if op == proto.RaidLeader {
				party.Leader = id
			} else {
				party.RaidSubLeader = id
			}
		}
		return true, nil
	case proto.RaidInvite, proto.RaidInviteName:
		if op == proto.RaidInvite {
			id, err := proto.DecodeU32(p)
			if err != nil {
				return true, err
			}
			party.InviteID, party.InviteName = id, ""
			if e := s.Entities[id]; e != nil && e.Shape != nil {
				party.InviteName = e.Shape.Name
			}
		} else {
			name, err := proto.DecodePartyInviteName(p)
			if err != nil {
				return true, err
			}
			party.InviteID, party.InviteName = 0, name
			for id, e := range s.Entities {
				if e.Shape != nil && e.Shape.Name == name {
					party.InviteID = id
					break
				}
			}
		}
		party.RaidInvite = true
		party.InviteSerial++
		return true, nil
	case proto.RaidAnswer:
		reply, err := proto.DecodePartyReply(p)
		if err != nil {
			return true, err
		}
		party.Reply = reply
		party.ReplySerial++
		return true, nil
	case proto.RaidError:
		r := proto.NewReader(p)
		code := r.U32()
		if err := r.Err(); err != nil {
			return true, err
		}
		party.RaidError = code
		party.RaidErrorSerial++
		return true, nil
	case proto.RaidDisband:
		party.Members, party.Leader, party.RaidSlots, party.RaidSubLeader, party.Raid, party.raidPending = nil, 0, nil, 0, false, nil
		return true, nil
	case proto.PartyLeaveAlt, proto.RaidExpel:
		id, err := proto.DecodeU32(p)
		if err != nil {
			return true, err
		}
		if id == s.SelfID {
			party.Members, party.Leader, party.RaidSlots, party.RaidSubLeader, party.Raid, party.raidPending = nil, 0, nil, 0, false, nil
			return true, nil
		}
		for i, m := range party.Members {
			if m.ID == id {
				party.Members = append(party.Members[:i], party.Members[i+1:]...)
				break
			}
		}
		for slot, member := range party.RaidSlots {
			if member == id {
				delete(party.RaidSlots, slot)
			}
		}
		return true, nil
	case proto.RaidPools:
		r := proto.NewReader(p)
		id, hp, sp, mp := r.U32(), r.U32(), r.U32(), r.U32()
		if err := r.Err(); err != nil {
			return true, err
		}
		if m := party.Member(id); m != nil {
			m.HP, m.SP, m.MP = hp, sp, mp
		}
		return true, nil
	}
	return false, nil
}
