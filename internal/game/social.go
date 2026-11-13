package game

import (
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"strings"
)

type SocialState struct {
	Friends, Blocked           map[uint32]proto.SocialContact
	Search                     []proto.PartySearchEntry
	Invitation                 string
	InviteSerial, ResultSerial uint64
	Result                     uint8
}

func (s *State) applySocial(p []byte) error {
	v := &s.Social
	switch proto.Op(p) {
	case proto.FriendList, proto.BlockList:
		rows, err := proto.DecodeSocialList(p)
		if err != nil {
			return err
		}
		list := map[uint32]proto.SocialContact{}
		for _, r := range rows {
			list[r.ID] = r
		}
		if proto.Op(p) == proto.FriendList {
			v.Friends = list
		} else {
			v.Blocked = list
		}
	case proto.PartySearchList:
		rows, err := proto.DecodePartySearch(p)
		if err != nil {
			return err
		}
		v.Search = rows
	case proto.FriendInvite:
		r := proto.NewReader(p)
		name := r.Fixed(21)
		if err := r.Err(); err != nil {
			return err
		}
		v.Invitation = name
		v.InviteSerial++
	case proto.FriendAnswer, proto.SocialError, proto.PartySearchRequest:
		r := proto.NewReader(p)
		result := r.U8()
		if err := r.Err(); err != nil {
			return err
		}
		v.Result = result
		v.ResultSerial++
	case proto.FriendDelete, proto.BlockDelete:
		r := proto.NewReader(p)
		id := r.U32()
		if err := r.Err(); err != nil {
			return err
		}
		if proto.Op(p) == proto.FriendDelete {
			delete(v.Friends, id)
		} else {
			delete(v.Blocked, id)
		}
	case proto.BlockAdd, proto.FriendAdded:
		r := proto.NewReader(p)
		row := proto.SocialContact{ID: r.U32()}
		if proto.Op(p) == proto.FriendAdded {
			row.Class = r.U8()
			row.Online = true
		}
		row.Name = r.Fixed(21)
		if err := r.Err(); err != nil {
			return err
		}
		if proto.Op(p) == proto.FriendAdded {
			if v.Friends == nil {
				v.Friends = map[uint32]proto.SocialContact{}
			}
			v.Friends[row.ID] = row
		} else {
			if v.Blocked == nil {
				v.Blocked = map[uint32]proto.SocialContact{}
			}
			v.Blocked[row.ID] = row
		}
	case proto.FriendPresence:
		r := proto.NewReader(p)
		id := r.U32()
		online := r.U8() != 0
		if err := r.Err(); err != nil {
			return err
		}
		if row, ok := v.Friends[id]; ok {
			row.Online = online
			v.Friends[id] = row
		}
	case proto.FriendMemo, proto.BlockMemo:
		r := proto.NewReader(p)
		id, memo := r.U32(), r.Text()
		if err := r.Err(); err != nil {
			return err
		}
		list := v.Friends
		if proto.Op(p) == proto.BlockMemo {
			list = v.Blocked
		}
		if row, ok := list[id]; ok {
			row.Memo = memo
			list[id] = row
		}
	}
	return nil
}
func (s *State) blockedChat(l ChatLine) bool {
	for id, b := range s.Social.Blocked {
		if l.SenderID == id || l.Sender != "" && strings.EqualFold(l.Sender, b.Name) {
			return true
		}
	}
	return false
}
