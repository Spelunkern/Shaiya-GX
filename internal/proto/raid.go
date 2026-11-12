package proto

import "fmt"

type RaidMember struct {
	Slot   uint16
	Member PartyMember
}

type RaidRosterChunk struct {
	More                      bool
	LeaderSlot, SubLeaderSlot uint8
	Division                  uint16
	Kind                      uint8
	AutoJoin                  bool
	Members                   []RaidMember
}

// 0x596740 and Rapture send up to six member records per chunk.
func DecodeRaidRoster(p []byte) (RaidRosterChunk, error) {
	r := NewReader(p)
	var chunk RaidRosterChunk
	chunk.More = r.U8() != 0
	chunk.LeaderSlot, chunk.SubLeaderSlot = r.U8(), r.U8()
	chunk.Division, chunk.Kind, chunk.AutoJoin = r.U16(), r.U8(), r.U8() != 0
	n := int(r.U8())
	if err := r.Err(); err != nil {
		return chunk, err
	}
	if n > 6 {
		return chunk, fmt.Errorf("raid: invalid chunk size %d", n)
	}
	if n > r.Remaining()/69 {
		return chunk, ErrShort
	}
	chunk.Members = make([]RaidMember, n)
	for i := range chunk.Members {
		chunk.Members[i] = RaidMember{r.U16(), readPartyMember(r)}
	}
	return chunk, r.Err()
}

func DecodeRaidJoined(p []byte) (RaidMember, error) {
	r := NewReader(p)
	m := RaidMember{r.U16(), readPartyMember(r)}
	return m, r.Err()
}

func RaidConvertPacket(kind uint8, autoJoin bool, division uint32) []byte {
	w := NewWriter(RaidConvert)
	w.U8(kind)
	if autoJoin {
		w.U8(1)
	} else {
		w.U8(0)
	}
	w.U32(division)
	return w.Bytes()
}
func RaidDivisionPacket(division uint32) []byte { return idPacket(RaidDivision, division) }
func RaidAutoJoinPacket(enabled bool) []byte {
	w := NewWriter(RaidAutoJoin)
	if enabled {
		w.U8(1)
	} else {
		w.U8(0)
	}
	return w.Bytes()
}
func RaidSwapPacket(first, second uint32) []byte {
	w := NewWriter(RaidSwap)
	w.U32(first)
	w.U32(second)
	return w.Bytes()
}
func RaidInvitePacket(id uint32) []byte    { return idPacket(RaidInvite, id) }
func RaidSubLeaderPacket(id uint32) []byte { return idPacket(RaidSubLeader, id) }
func RaidLeaderPacket(id uint32) []byte    { return idPacket(RaidLeader, id) }
func RaidExpelPacket(id uint32) []byte     { return idPacket(RaidExpel, id) }
func RaidDisbandPacket() []byte            { return NewWriter(RaidDisband).Bytes() }
func RaidAnswerPacket(accept bool, id uint32) []byte {
	w := NewWriter(RaidAnswer)
	if accept {
		w.U8(0)
	} else {
		w.U8(1)
	}
	w.U32(id)
	return w.Bytes()
}
func RaidNamePacket(op Opcode, name string) []byte {
	w := NewWriter(op)
	w.Fixed(name, 21, true)
	return w.Bytes()
}
