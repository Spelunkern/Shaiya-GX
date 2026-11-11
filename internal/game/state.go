// Package game holds the client's view of the world. It is pure: no
// sockets, no GL. The loop that owns a State feeds it every payload with
// Apply and sends the requests it builds through the network layer.
package game

import (
	"fmt"

	"github.com/Spelunkern/shaiya-go-client/internal/proto"
)

// Phase follows the original screens.
type Phase int

const (
	PhaseConnecting Phase = iota
	PhaseFaction          // account has no country yet
	PhaseCharSelect       // 0x0101 list + 0x0109 received
	PhaseLoading          // 0x0104 accepted, initial state arriving
	PhaseInWorld          // 0x0201 for ourselves received
)

func (p Phase) String() string {
	return [...]string{"connecting", "faction", "char-select", "loading", "in-world"}[p]
}

type Kind uint8

const (
	KindUser Kind = iota + 1
	KindMob
	KindNpc
)

// Entity is anything placed in view.
type Entity struct {
	Kind    Kind
	ID      uint32
	TypeID  uint16 // mob id or npc type id
	NpcType uint8
	X, Y, Z float32
	Motion  uint8
	Dir     uint16
}

// ChatLine is one received message.
type ChatLine struct {
	Op       proto.Opcode
	SenderID uint32
	Sender   string
	Text     string
}

// Event tells the caller what a payload changed; Text is for logs.
type Event struct {
	Op   proto.Opcode
	Text string
}

type State struct {
	Phase   Phase
	Faction proto.Faction
	Slots   map[uint8]proto.CharacterSlot // non-empty slots only

	SelfID  uint32
	MapID   uint16
	Details proto.Details
	Self    Entity

	Entities map[uint32]*Entity
	Chat     []ChatLine

	Unknown map[proto.Opcode]int // opcodes received but not modeled
}

func New() *State {
	return &State{Slots: map[uint8]proto.CharacterSlot{}, Entities: map[uint32]*Entity{}, Unknown: map[proto.Opcode]int{}}
}

const maxChat = 200

// Apply updates the state from one payload. Malformed packets are
// reported as an error and leave the state as it was.
func (s *State) Apply(p []byte) (Event, error) {
	op := proto.Op(p)
	ev := Event{Op: op}
	fail := func(err error) (Event, error) { return ev, fmt.Errorf("0x%04X: %w", uint16(op), err) }
	switch op {
	case proto.AccountFaction:
		f, err := proto.DecodeFaction(p)
		if err != nil {
			return fail(err)
		}
		s.Faction = f
		if f.Faction > 1 {
			s.Phase = PhaseFaction
		} else if s.Phase < PhaseLoading {
			s.Phase = PhaseCharSelect
		}
		ev.Text = fmt.Sprintf("faction=%d maxMode=%d", f.Faction, f.MaxMode)
	case proto.CharacterList:
		c, err := proto.DecodeCharacterSlot(p)
		if err != nil {
			return fail(err)
		}
		if c.ID == 0 {
			delete(s.Slots, c.Slot)
		} else {
			s.Slots[c.Slot] = c
			ev.Text = fmt.Sprintf("slot %d: %s lv%d id=%d", c.Slot, c.Name, c.Level, c.ID)
		}
	case proto.CreateCharacter:
		v, err := proto.DecodeU8(p)
		if err != nil {
			return fail(err)
		}
		ev.Text = fmt.Sprintf("create result=%d", v)
	case proto.DeleteCharacter, proto.RestoreCharacter:
		r, err := proto.DecodeIDResult(p)
		if err != nil {
			return fail(err)
		}
		ev.Text = fmt.Sprintf("result=%d id=%d", r.Result, r.ID)
	case proto.SelectCharacter:
		r, err := proto.DecodeIDResult(p)
		if err != nil {
			return fail(err)
		}
		if r.Result == 0 {
			s.SelfID, s.Phase = r.ID, PhaseLoading
			s.Entities = map[uint32]*Entity{}
		}
		ev.Text = fmt.Sprintf("select result=%d id=%d", r.Result, r.ID)
	case proto.CharacterDetails:
		d, err := proto.DecodeDetails(p)
		if err != nil {
			return fail(err)
		}
		s.Details = d
		s.Self = Entity{Kind: KindUser, ID: s.SelfID, X: d.X, Y: d.Y, Z: d.Z, Dir: d.Dir}
		ev.Text = fmt.Sprintf("details hp=%d pos=(%.1f,%.1f,%.1f) money=%d", d.MaxHP, d.X, d.Y, d.Z, d.Money)
	case proto.CharacterMapTeleport:
		m, err := proto.DecodeMapPosition(p)
		if err != nil {
			return fail(err)
		}
		if m.ID == s.SelfID {
			s.MapID, s.Self.X, s.Self.Y, s.Self.Z = m.MapID, m.X, m.Y, m.Z
		}
		ev.Text = fmt.Sprintf("map %d pos=(%.1f,%.1f,%.1f)", m.MapID, m.X, m.Y, m.Z)
	case proto.CharacterEnteredMap:
		u, err := proto.DecodeUserEnter(p)
		if err != nil {
			return fail(err)
		}
		if u.ID == s.SelfID {
			s.Self.X, s.Self.Y, s.Self.Z = u.X, u.Y, u.Z
			s.Phase = PhaseInWorld
		} else {
			s.Entities[u.ID] = &Entity{Kind: KindUser, ID: u.ID, X: u.X, Y: u.Y, Z: u.Z}
		}
		ev.Text = fmt.Sprintf("user %d enters", u.ID)
	case proto.UserLeave, proto.MobLeave, proto.MapNpcLeave:
		id, err := proto.DecodeU32(p)
		if err != nil {
			return fail(err)
		}
		delete(s.Entities, id)
	case proto.CharacterMove:
		m, err := proto.DecodeUserMove(p)
		if err != nil {
			return fail(err)
		}
		e := &s.Self
		if m.ID != s.SelfID {
			if e = s.Entities[m.ID]; e == nil {
				e = &Entity{Kind: KindUser, ID: m.ID}
				s.Entities[m.ID] = e
			}
		}
		e.Motion, e.Dir, e.X, e.Y, e.Z = m.Motion, m.Dir, m.X, m.Y, m.Z
	case proto.MobEnter:
		m, err := proto.DecodeMobEnter(p)
		if err != nil {
			return fail(err)
		}
		s.Entities[m.ID] = &Entity{Kind: KindMob, ID: m.ID, TypeID: m.MobID, X: m.X, Z: m.Z}
	case proto.MobMove:
		m, err := proto.DecodeMobMove(p)
		if err != nil {
			return fail(err)
		}
		if e := s.Entities[m.ID]; e != nil {
			e.Motion, e.X, e.Z = m.Motion, m.X, m.Z
		}
	case proto.MapNpcEnter:
		n, err := proto.DecodeNpcEnter(p)
		if err != nil {
			return fail(err)
		}
		s.Entities[n.ID] = &Entity{Kind: KindNpc, ID: n.ID, NpcType: n.Type, TypeID: n.TypeID, X: n.X, Y: n.Y, Z: n.Z, Dir: n.Angle}
	case proto.ChatNormal, proto.ChatWorld, proto.ChatParty, proto.ChatGuild, proto.ChatMap, proto.ChatNormalAdmin, proto.ChatWorldAdmin:
		c, err := proto.DecodeChatByID(p)
		if err != nil {
			return fail(err)
		}
		s.addChat(ChatLine{Op: op, SenderID: c.SenderID, Text: c.Text})
		ev.Text = fmt.Sprintf("chat %d: %s", c.SenderID, c.Text)
	case proto.ChatWhisper, proto.ChatShout, proto.ChatWorldShout:
		c, err := proto.DecodeChatByName(p)
		if err != nil {
			return fail(err)
		}
		s.addChat(ChatLine{Op: op, Sender: c.Sender, Text: c.Text})
		ev.Text = fmt.Sprintf("chat %s: %s", c.Sender, c.Text)
	case proto.Logout:
		s.Phase, s.SelfID = PhaseCharSelect, 0
		s.Entities = map[uint32]*Entity{}
		ev.Text = "back to character select"
	default:
		s.Unknown[op]++
	}
	return ev, nil
}

func (s *State) addChat(l ChatLine) {
	s.Chat = append(s.Chat, l)
	if len(s.Chat) > maxChat {
		s.Chat = s.Chat[len(s.Chat)-maxChat:]
	}
}
