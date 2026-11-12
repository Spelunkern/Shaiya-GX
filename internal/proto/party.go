package proto

import "fmt"

// Normal party records: readers 0x592480/0x5926a0. Effects are skill, level,
// remaining milliseconds; they have no active-buff instance ID on this wire.
type PartyEffect struct {
	Skill     uint16
	Level     uint8
	Remaining uint32
}
type PartyMember struct {
	ID                              uint32
	Name                            string
	Level                           uint16
	Class                           uint8
	HP, MaxHP, MP, MaxMP, SP, MaxSP uint32
	Instance                        uint16
	X, Y, Z                         float32
	Effects                         []PartyEffect
}
type PartyList struct {
	LeaderIndex uint8
	Members     []PartyMember
}
type PartyReply struct {
	Result uint8
	ID     uint32
}

func readPartyMember(r *Reader) (m PartyMember) {
	m.ID, m.Name, m.Level, m.Class = r.U32(), r.Fixed(21), r.U16(), r.U8()
	m.HP, m.MaxHP, m.MP, m.MaxMP, m.SP, m.MaxSP = r.U32(), r.U32(), r.U32(), r.U32(), r.U32(), r.U32()
	m.Instance, m.X, m.Y, m.Z = r.U16(), r.F32(), r.F32(), r.F32()
	n := int(r.U8())
	if n > r.Remaining()/7 {
		r.err = ErrShort
		return m
	}
	m.Effects = make([]PartyEffect, n)
	for i := range m.Effects {
		m.Effects[i] = PartyEffect{r.U16(), r.U8(), r.U32()}
	}
	return m
}
func DecodePartyList(p []byte) (list PartyList, err error) {
	r := NewReader(p)
	list.LeaderIndex = r.U8()
	n := int(r.U8())
	// Native scratch contains eight records; the normal party has at most seven.
	if n > 8 {
		return list, fmt.Errorf("party: invalid member count %d", n)
	}
	if n > r.Remaining()/67 {
		return list, ErrShort
	}
	list.Members = make([]PartyMember, n)
	for i := range list.Members {
		list.Members[i] = readPartyMember(r)
	}
	return list, r.Err()
}
func DecodePartyJoined(p []byte) (PartyMember, error) {
	r := NewReader(p)
	m := readPartyMember(r)
	return m, r.Err()
}
func DecodePartyReply(p []byte) (PartyReply, error) {
	r := NewReader(p)
	m := PartyReply{r.U8(), r.U32()}
	return m, r.Err()
}
func DecodePartyInviteName(p []byte) (string, error) {
	r := NewReader(p)
	s := r.Fixed(21)
	return s, r.Err()
}

// Writers 0x599540, 0x599570, 0x5995b0, 0x5995e0, 0x599610, 0x599640.
// Requests never mutate membership.
func PartyInvitePacket(id uint32) []byte { return idPacket(PartyInvite, id) }

// The native acceptance call 0x490c52 supplies the answering character's ID.
// Rapture reads the result byte and ignores this trailing native field.
func PartyAnswerPacket(accept bool, characterID uint32) []byte {
	w := NewWriter(PartyAnswer)
	if accept {
		w.U8(0)
	} else {
		w.U8(1)
	}
	w.U32(characterID)
	return w.Bytes()
}
func PartyLeavePacket() []byte           { return NewWriter(PartyLeft).Bytes() }
func PartyExpelPacket(id uint32) []byte  { return idPacket(PartyExpelled, id) }
func PartyLeaderPacket(id uint32) []byte { return idPacket(PartyLeader, id) }
func PartyInviteNamePacket(name string) []byte {
	w := NewWriter(PartyInviteName)
	w.Fixed(name, 21, true)
	return w.Bytes()
}
