package sdata

import "fmt"

// Monster is an EP5 Monster.SData record (Parsec MonsterRecord).
// IDs are the zero-based record indices used by the server.
type Monster struct {
	Name                          string
	Model, Level                  uint16
	HP                            uint32
	Size, Element                 uint8
	NormalTimeMS, ChaseTimeMS     uint32
	NormalStep, ChaseStep         uint8
	AttackTypes, AttackAnimations [3]uint8
}

func ParseMonsters(raw []byte) ([]Monster, error) {
	b, err := Decrypt(raw)
	if err != nil {
		return nil, err
	}
	c := &cursor{b: b}
	n := c.count(35)
	out := make([]Monster, 0, n)
	for i := 0; i < n && c.err == nil; i++ {
		m := Monster{Name: c.str(), Model: c.u16(), Level: c.u16()}
		c.take(1)
		m.HP = c.u32()
		c.take(1)
		m.Size = c.u8()
		m.Element = c.u8()
		m.NormalTimeMS, m.NormalStep = c.u32(), c.u8()
		m.ChaseTimeMS, m.ChaseStep = c.u32(), c.u8()
		for j := range 3 {
			m.AttackTypes[j], m.AttackAnimations[j] = c.u8(), c.u8()
		}
		c.take(3) // AttackPlus3 and quest item
		out = append(out, m)
	}
	if c.err != nil {
		return nil, c.err
	}
	if c.off != len(b) {
		return nil, fmt.Errorf("monster sdata: %d trailing bytes", len(b)-c.off)
	}
	return out, nil
}
