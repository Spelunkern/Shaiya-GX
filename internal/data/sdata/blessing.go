package sdata

import "fmt"

type BlessingEffect struct {
	Type  uint8
	Value uint16
}
type BlessingTier struct {
	Country uint8
	Kills   uint32
	Level   uint16
	Effects [6]BlessingEffect
}

// ParseBlessing reads EP5 KillStatus.SData (the server uses 25-byte records).
func ParseBlessing(raw []byte) ([]BlessingTier, error) {
	b, err := Decrypt(raw)
	if err != nil {
		return nil, err
	}
	c := &cursor{b: b}
	n := c.count(25)
	out := make([]BlessingTier, 0, n)
	for i := 0; i < n && c.err == nil; i++ {
		t := BlessingTier{Country: c.u8(), Kills: c.u32(), Level: c.u16()}
		for j := range t.Effects {
			t.Effects[j] = BlessingEffect{Type: c.u8(), Value: c.u16()}
		}
		if t.Country > 1 {
			return nil, fmt.Errorf("blessing: invalid country %d", t.Country)
		}
		out = append(out, t)
	}
	if c.err != nil {
		return nil, c.err
	}
	if c.off != len(b) {
		return nil, fmt.Errorf("blessing: %d trailing bytes", len(b)-c.off)
	}
	return out, nil
}

func BlessingMaximum(tiers []BlessingTier, country uint8) uint32 {
	var result uint32
	for _, tier := range tiers {
		if tier.Country == country {
			result = max(result, tier.Kills)
		}
	}
	return result
}
