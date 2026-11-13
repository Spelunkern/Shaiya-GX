package world

import (
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"github.com/Spelunkern/Shaiya-GX/server/internal/packet"
	"sort"
	"strings"
	"sync"
	"time"
)

// Native EP5 0x2201..0x220e, writers 0x59bb30..0x59bd80.
var socialState = struct {
	sync.Mutex
	invites   map[uint32]socialInvite
	searching map[uint32]bool
}{invites: map[uint32]socialInvite{}, searching: map[uint32]bool{}}

type socialInvite struct {
	from  uint32
	until time.Time
}

func socialList(charID uint32, blocked bool) ([]byte, error) {
	rows, err := db.SocialContacts(charID, blocked)
	if err != nil {
		return nil, err
	}
	op := packet.Opcode(0x2201)
	if blocked {
		op = 0x2208
	}
	w := packet.New(op)
	w.U8(uint8(min(len(rows), 50)))
	for _, r := range rows[:min(len(rows), 50)] {
		w.U32(r.ID)
		if !blocked {
			w.U8(r.Class)
			online := uint8(0)
			if FindPlayerByID(r.ID) != nil {
				online = 1
			}
			w.U8(online)
		}
		w.FixedString(r.Name, 21)
		memo := truncateSocialMemo(r.Memo)
		w.U8(uint8(len(memo) + 1))
		w.Bytes(append([]byte(memo), 0))
	}
	return w.MustFinish(), nil
}
func truncateSocialMemo(v string) string {
	v = strings.ReplaceAll(v, "\x00", "")
	if len(v) > 50 {
		return v[:50]
	}
	return v
}
func (s *Session) sendSocial() {
	if s.char == nil {
		return
	}
	for _, blocked := range []bool{false, true} {
		p, err := socialList(s.char.ID, blocked)
		if err != nil {
			s.log.Printf("social list: %v", err)
			continue
		}
		s.send(p, "native social list")
	}

}
func (s *Session) tickSocial() {
	if s.char == nil || time.Since(s.socialChecked) < 3*time.Second {
		return
	}
	s.socialChecked = time.Now()
	s.sendSocial()
}
func (s *Session) handleSocial(op packet.Opcode, p []byte) {
	if s.char == nil {
		return
	}
	result := func(op packet.Opcode, code uint8) {
		w := packet.New(op)
		w.U8(code)
		s.send(w.MustFinish(), "social result")
	}
	id := func() uint32 {
		if len(p) < 6 {
			return 0
		}
		v, _ := packet.U32(p[2:])
		return v
	}
	name := func() string {
		if len(p) < 23 {
			return ""
		}
		return strings.TrimSpace(strings.TrimRight(string(p[2:23]), "\x00"))
	}
	switch op {
	case 0x2202:
		target := FindPlayerByName(name())
		if target == nil || target.CharacterID == s.char.ID || target.Faction() != s.faction || target.MapID() != s.char.MapID || db.SocialBlocked(target.CharacterID, s.char.ID) {
			result(0x2203, 2)
			return
		}
		socialState.Lock()
		previous, busy := socialState.invites[target.CharacterID]
		busy = busy && time.Now().Before(previous.until)
		if !busy {
			socialState.invites[target.CharacterID] = socialInvite{s.char.ID, time.Now().Add(30 * time.Second)}
		}
		socialState.Unlock()
		if busy {
			result(0x2203, 0)
			return
		}
		w := packet.New(0x2202)
		w.FixedString(s.char.Name, 21)
		if err := target.Send(w.MustFinish(), "friend invitation"); err != nil {
			s.log.Printf("friend invite: %v", err)
		}
	case 0x2203:
		if len(p) < 3 {
			return
		}
		socialState.Lock()
		invite, ok := socialState.invites[s.char.ID]
		delete(socialState.invites, s.char.ID)
		socialState.Unlock()
		if !ok || time.Now().After(invite.until) {
			return
		}
		sender := FindPlayerByID(invite.from)
		if sender == nil {
			return
		}
		code := uint8(0)
		if p[2] == 1 && sender.Faction() == s.faction && !db.SocialBlocked(s.char.ID, sender.CharacterID) {
			if err := db.AddFriends(sender.Snapshot(), *s.char); err != nil {
				s.log.Printf("add friends: %v", err)
			} else {
				code = 1
			}
		}
		result(0x2203, code)
		w := packet.New(0x2203)
		w.U8(code)
		if err := sender.Send(w.MustFinish(), "friend invitation result"); err != nil {
			s.log.Printf("friend result: %v", err)
		}
		s.sendSocial()
		if data, err := socialList(sender.CharacterID, false); err == nil {
			if err = sender.Send(data, "friend list"); err != nil {
				s.log.Printf("friend list send: %v", err)
			}
		}
	case 0x2205, 0x220a:
		target := id()
		if target == 0 {
			return
		}
		if err := db.RemoveSocial(s.char.ID, target, op == 0x220a); err != nil {
			s.log.Printf("remove social: %v", err)
			return
		}
		w := packet.New(op)
		w.U32(target)
		s.send(w.MustFinish(), "social removal")
		s.sendSocial()
	case 0x2209:
		target, err := db.SocialCharacter(name())
		if err != nil || target.ID == s.char.ID {
			result(0x220e, 0)
			return
		}
		if err = db.AddSocial(s.char.ID, target, true); err != nil {
			s.log.Printf("block: %v", err)
			result(0x220e, 0)
			return
		}
		w := packet.New(0x2209)
		w.U32(target.ID)
		w.FixedString(target.Name, 21)
		s.send(w.MustFinish(), "block added")
		result(0x220e, 1)
		s.sendSocial()
	case 0x2206, 0x220b:
		if len(p) < 7 {
			return
		}
		n := int(p[6])
		if n < 1 || n > 51 || len(p) != 7+n {
			return
		}
		memo := strings.TrimRight(string(p[7:]), "\x00")
		if err := db.SocialMemo(s.char.ID, id(), memo, op == 0x220b); err != nil {
			s.log.Printf("social memo: %v", err)
			return
		}
		w := packet.New(op)
		w.U32(id())
		w.U8(uint8(len(memo) + 1))
		w.Bytes(append([]byte(memo), 0))
		s.send(w.MustFinish(), "social memo")
		s.sendSocial()
	case 0x220c:
		socialState.Lock()
		socialState.searching[s.char.ID] = true
		socialState.Unlock()
		result(0x220c, 1)
		s.sendPartySearch()
	}
}
func (s *Session) sendPartySearch() {
	rows := []db.Character{}
	for _, p := range AllPlayers() {
		socialState.Lock()
		registered := socialState.searching[p.CharacterID]
		socialState.Unlock()
		if !registered {
			continue
		}
		if p.CharacterID == s.char.ID || p.Faction() != s.faction {
			continue
		}
		partyMu.Lock()
		available := partyOf(p) == nil
		partyMu.Unlock()
		if available {
			rows = append(rows, p.Snapshot())
		}
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Name < rows[j].Name })
	w := packet.New(0x220d)
	w.U8(uint8(min(len(rows), 100)))
	for _, r := range rows[:min(len(rows), 100)] {
		w.U8(uint8(min(r.Level, 255)))
		w.U8(r.Class)
		w.FixedString(r.Name, 21)
	}
	s.send(w.MustFinish(), "party search list")
}

func clearSocialSession(id uint32) {
	socialState.Lock()
	defer socialState.Unlock()
	delete(socialState.searching, id)
	delete(socialState.invites, id)
	for target, invite := range socialState.invites {
		if invite.from == id {
			delete(socialState.invites, target)
		}
	}
}
