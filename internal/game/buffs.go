package game

import (
	"fmt"
	"sort"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

// The native map is ordered by instance ID. Refreshes replace the same slot;
// no local countdown removes authoritative effects.
func (s *State) addBuff(b proto.ActiveBuff) error {
	for i, old := range s.Buffs {
		if old.Instance == b.Instance {
			s.Buffs[i] = b
			return nil
		}
	}
	if len(s.Buffs) >= 256 {
		return fmt.Errorf("game: too many active buffs")
	}
	s.Buffs = append(s.Buffs, b)
	sort.Slice(s.Buffs, func(i, j int) bool { return s.Buffs[i].Instance < s.Buffs[j].Instance })
	return nil
}

func (s *State) removeBuff(id uint32) {
	kept := s.Buffs[:0]
	for _, b := range s.Buffs {
		if b.Instance != id {
			kept = append(kept, b)
		}
	}
	clear(s.Buffs[len(kept):])
	s.Buffs = kept
}
