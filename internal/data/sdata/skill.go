package sdata

import (
	"encoding/binary"
	"fmt"
)

// Skill: EP5 SkillDefinition in Parsec; nine levels per group, wire ID = group index + 1 (Rapture SkillID).
// The fixed payload is 116 bytes after name and description.
type Skill struct {
	Name, Description                          string
	Level                                      uint8
	Icon, RequiredLevel                        uint16
	Animation, Sound                           uint16
	VisualCode, Detail                         uint16
	Toggle                                     uint8
	Country                                    uint8
	Classes                                    [6]uint8
	Grow, Points, Show, Attack, Effect, Target uint8
	SP, MP, ResetTime                          uint16
	ReadyTime                                  uint8
	AttackRange                                uint8
	Attribute                                  uint8
	KeepTime                                   uint16
	FixRange                                   uint8
}
type SkillKey struct {
	ID    uint16
	Level uint8
}

func ParseSkills(raw []byte) (map[SkillKey]Skill, error) {
	b, err := Decrypt(raw)
	if err != nil {
		return nil, err
	}
	c := &cursor{b: b}
	n := c.count(9 * 124)
	if n > 65535 {
		return nil, fmt.Errorf("skill: too many groups %d", n)
	}
	out := map[SkillKey]Skill{}
	for id := 0; id < n && c.err == nil; id++ {
		for level := 0; level < 9 && c.err == nil; level++ {
			s := Skill{Name: c.str(), Description: c.str()}
			p := c.take(116)
			s.Level = p[0]
			s.Icon = binary.LittleEndian.Uint16(p[1:])
			s.Animation = uint16(p[3])
			s.VisualCode = binary.LittleEndian.Uint16(p[4:])
			s.Detail = binary.LittleEndian.Uint16(p[22:])
			s.Toggle, s.Sound = p[5], binary.LittleEndian.Uint16(p[6:])
			s.RequiredLevel = binary.LittleEndian.Uint16(p[8:])
			s.Country = p[10]
			copy(s.Classes[:], p[11:17])
			s.Grow, s.Points, s.Show, s.Attack, s.Effect = p[17], p[18], p[19], p[20], p[21]
			s.SP, s.MP = binary.LittleEndian.Uint16(p[40:]), binary.LittleEndian.Uint16(p[42:])
			s.ReadyTime, s.ResetTime, s.Target = p[44], binary.LittleEndian.Uint16(p[45:]), p[56]
			s.AttackRange = p[47]
			s.Attribute = p[49]
			s.KeepTime, s.FixRange = binary.LittleEndian.Uint16(p[59:]), p[111]
			out[SkillKey{uint16(id + 1), s.Level}] = s
		}
	}
	if c.err != nil {
		return nil, c.err
	}
	if c.off != len(b) {
		return nil, fmt.Errorf("skill: %d trailing bytes", len(b)-c.off)
	}
	return out, nil
}

// KeepSeconds follows Rapture's normalized KeepTime/FixRange. A newly applied
// buff lacks a wire duration; snapshots supply their actual remaining seconds.
func (s Skill) KeepSeconds() uint32 {
	n := uint32(s.KeepTime)
	if s.FixRange == 1 {
		n *= 3600
	}
	return n
}
