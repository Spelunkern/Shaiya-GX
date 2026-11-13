package proto

import "fmt"

const (
	FriendList         Opcode = 0x2201
	FriendInvite       Opcode = 0x2202
	FriendAnswer       Opcode = 0x2203
	FriendAdded        Opcode = 0x2204
	FriendDelete       Opcode = 0x2205
	FriendMemo         Opcode = 0x2206
	FriendPresence     Opcode = 0x2207
	BlockList          Opcode = 0x2208
	BlockAdd           Opcode = 0x2209
	BlockDelete        Opcode = 0x220A
	BlockMemo          Opcode = 0x220B
	PartySearchRequest Opcode = 0x220C
	PartySearchList    Opcode = 0x220D
	SocialError        Opcode = 0x220E
)

type SocialContact struct {
	ID         uint32
	Class      uint8
	Online     bool
	Name, Memo string
}
type PartySearchEntry struct {
	Level, Class uint8
	Name         string
}

// Native readers 0x595140/0x5953a0: bounded count, fixed Latin-1 name and memo.
func DecodeSocialList(p []byte) ([]SocialContact, error) {
	r := NewReader(p)
	n := int(r.U8())
	if n > 50 {
		return nil, fmt.Errorf("social list count %d", n)
	}
	rows := make([]SocialContact, n)
	for i := range rows {
		v := &rows[i]
		v.ID = r.U32()
		if Op(p) == FriendList {
			v.Class = r.U8()
			v.Online = r.U8() != 0
		}
		v.Name = r.Fixed(21)
		v.Memo = r.Text()
		if len(v.Memo) > 50 {
			return nil, fmt.Errorf("social memo too long")
		}
	}
	if err := r.Err(); err != nil {
		return nil, err
	}
	if r.Remaining() != 0 {
		return nil, fmt.Errorf("social list trailing data")
	}
	return rows, nil
}
func DecodePartySearch(p []byte) ([]PartySearchEntry, error) {
	r := NewReader(p)
	n := int(r.U8())
	if n > 100 {
		return nil, fmt.Errorf("party search count")
	}
	rows := make([]PartySearchEntry, n)
	for i := range rows {
		rows[i] = PartySearchEntry{r.U8(), r.U8(), r.Fixed(21)}
	}
	if err := r.Err(); err != nil {
		return nil, err
	}
	if r.Remaining() != 0 {
		return nil, fmt.Errorf("party search trailing data")
	}
	return rows, nil
}
func SocialNamePacket(op Opcode, name string) []byte {
	w := NewWriter(op)
	w.Fixed(name, 21, true)
	return w.Bytes()
}
func SocialIDPacket(op Opcode, id uint32) []byte { w := NewWriter(op); w.U32(id); return w.Bytes() }
func FriendAnswerPacket(accept bool) []byte {
	w := NewWriter(FriendAnswer)
	if accept {
		w.U8(1)
	} else {
		w.U8(0)
	}
	return w.Bytes()
}

func SocialMemoPacket(op Opcode, id uint32, memo string) []byte {
	w := NewWriter(op)
	w.U32(id)
	w.Text(memo)
	return w.Bytes()
}
