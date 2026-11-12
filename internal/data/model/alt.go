package model

import "fmt"

// Action is AL3 metadata (Game.exe 0x552120; Parsec AltAnimation).
type Action struct {
	Name       string
	Mode, Next uint32
	Metadata   [4]float32
}

func ParseALT(b []byte) ([]Action, error) {
	c := &cursor{b: b}
	if string(c.take(3)) != "AL3" {
		return nil, fmt.Errorf("model: invalid ALT signature")
	}
	n := c.count(4)
	out := make([]Action, n)
	for i := range out {
		out[i].Name = c.str()
		if out[i].Name != "" {
			out[i].Mode, out[i].Next = c.u32(), c.u32()
			for j := range out[i].Metadata {
				out[i].Metadata[j] = c.f32()
			}
		}
	}
	if c.err != nil {
		return nil, c.err
	}
	if c.off != len(b) {
		return nil, fmt.Errorf("ALT: %d trailing bytes", len(b)-c.off)
	}
	return out, nil
}
