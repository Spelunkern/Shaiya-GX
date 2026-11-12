package model

import "fmt"

// Creature is a MON visual definition (Parsec MonRecord/MonObject).
type Creature struct {
	Name        string
	Unk         uint8
	Animations  [9]string // walk, run, attack1/2/3, death, breath, damage, idle
	Sounds      [4]string
	Effects     []string
	Parts       [][2]string // mesh, texture
	Height      float32
	BoneEffects [][2]int32
}

func ParseMON(b []byte) ([]Creature, error) {
	c := &cursor{b: b}
	sig := string(c.take(3))
	if sig != "MO2" && sig != "MO4" {
		return nil, fmt.Errorf("model: bad MON signature %q", sig)
	}
	n := c.count(85)
	out := make([]Creature, 0, n)
	for i := 0; i < n && c.err == nil; i++ {
		m := Creature{Name: c.str(), Unk: c.u8()}
		for j := range m.Animations {
			m.Animations[j] = c.str()
		}
		for j := range m.Sounds {
			m.Sounds[j] = c.str()
		}
		effects := 4
		if sig == "MO4" {
			effects = 5
		}
		for j := 0; j < effects; j++ {
			m.Effects = append(m.Effects, c.str())
		}
		for j, k := 0, c.count(8); j < k; j++ {
			m.Parts = append(m.Parts, [2]string{c.str(), c.str()})
		}
		m.Height = c.f32()
		for j, k := 0, c.count(8); j < k; j++ {
			m.BoneEffects = append(m.BoneEffects, [2]int32{c.i32(), c.i32()})
		}
		out = append(out, m)
	}
	if c.err != nil {
		return nil, c.err
	}
	if c.off != len(b) {
		return nil, fmt.Errorf("MON: %d trailing bytes", len(b)-c.off)
	}
	return out, nil
}
