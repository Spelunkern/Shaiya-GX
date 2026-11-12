package game

import "github.com/Spelunkern/Shaiya-GX/internal/proto"

// Only server acknowledgements change confirmed allocation. Derived ranges
// become unknown until the server supplies a fresh 0x0526; no combat formulas
// are duplicated in the client to guess them.
func (s *State) applyStatPoints(d proto.StatPoints) {
	s.LastStatPoints = d
	s.Details.StatPoint -= uint16(min(uint32(s.Details.StatPoint), d.Total()))
	s.StatsKnown = false
	for i, c := range s.Slots {
		if c.ID != s.SelfID {
			continue
		}
		addAttributes(&c, d)
		s.Slots[i] = c
	}
}

func addAttributes(c *proto.CharacterSlot, d proto.StatPoints) {
	values := []*uint16{&c.Str, &c.Dex, &c.Rec, &c.Int, &c.Wis, &c.Luc}
	for j, p := range values {
		*p = uint16(min(uint32(*p)+uint32(d[j]), 65535))
	}
}
