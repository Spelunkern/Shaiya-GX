package model

import "fmt"

type CloakPin struct {
	Vertex, Bone int
	Offset       [3]float32
}
type CloakRecord struct {
	Body, Texture, Shoulder, Mode int
	Columns, Rows                 int
	Pins                          []CloakPin
}
type CloakTable struct {
	Bodies, Shoulders []string
	Records           []CloakRecord
}

// ParseCloakPC is the original 0x585770 format: indexed 132-byte names,
// 524-byte configurations and twenty 24-byte pin descriptors per record.
func ParseCloakPC(raw []byte) (*CloakTable, error) {
	c := &cursor{b: raw}
	table := &CloakTable{}
	names := func() []string {
		n := c.count(132)
		out := make([]string, n)
		for range n {
			id := int(c.u32())
			p := c.take(128)
			name := string(p)
			for i, x := range p {
				if x == 0 {
					name = string(p[:i])
					break
				}
			}
			if id < 0 || id >= n {
				c.err = fmt.Errorf("cloak: name index %d/%d", id, n)
			} else {
				out[id] = name
			}
		}
		return out
	}
	table.Bodies = names()
	n := c.count(524)
	table.Records = make([]CloakRecord, n)
	for i := range table.Records {
		end := c.off + 524
		r := &table.Records[i]
		r.Body, r.Texture, r.Mode, r.Shoulder = int(c.u32()), int(c.u32()), int(c.u32()), int(c.u32())
		r.Columns, r.Rows = int(c.u32())+1, int(c.u32())+1
		c.take(20)
		for range 20 {
			active, vertex, bone := c.u32(), int(c.u32()), int(c.u32())
			offset := [3]float32{c.f32(), c.f32(), c.f32()}
			if active != 0 {
				r.Pins = append(r.Pins, CloakPin{vertex, bone, offset})
			}
		}
		if c.off != end || r.Columns < 2 || r.Columns > 32 || r.Rows < 2 || r.Rows > 128 {
			return nil, fmt.Errorf("cloak: invalid grid at record %d", i)
		}
	}
	table.Shoulders = names()
	if c.err != nil {
		return nil, c.err
	}
	if c.off != len(raw) {
		return nil, fmt.Errorf("cloak: trailing bytes")
	}
	for _, r := range table.Records {
		if r.Body < 0 || r.Body >= len(table.Bodies) || r.Shoulder < 0 || r.Shoulder >= len(table.Shoulders) {
			return nil, fmt.Errorf("cloak: invalid mesh index")
		}
		for _, pin := range r.Pins {
			if pin.Vertex >= r.Columns*r.Rows || pin.Bone >= 256 {
				return nil, fmt.Errorf("cloak: invalid pin")
			}
		}
	}
	return table, nil
}

// Cloth bodies have no version/bone-count pair: their leading zero is the
// bone count itself. Preserve vertex order: it is also the simulation grid.
func ParseCloth3DC(raw []byte) (*Mesh3DC, error) {
	c := &cursor{b: raw}
	if c.u32() != 0 {
		return nil, fmt.Errorf("cloak: expected boneless mesh")
	}
	m := &Mesh3DC{}
	n := c.count(40)
	m.Vertices = make([]SkinVertex, n)
	for i := range m.Vertices {
		v := &m.Vertices[i]
		v.Pos = [3]float32{c.f32(), c.f32(), c.f32()}
		c.take(8)
		v.Normal = [3]float32{c.f32(), c.f32(), c.f32()}
		v.UV = [2]float32{c.f32(), c.f32()}
	}
	n = c.count(6)
	m.Faces = make([][3]uint16, n)
	for i := range m.Faces {
		m.Faces[i] = [3]uint16{c.u16(), c.u16(), c.u16()}
		for _, x := range m.Faces[i] {
			if int(x) >= len(m.Vertices) {
				return nil, fmt.Errorf("cloak: invalid triangle")
			}
		}
	}
	if c.err != nil {
		return nil, c.err
	}
	if c.off != len(raw) {
		return nil, fmt.Errorf("cloak: trailing mesh bytes")
	}
	return m, nil
}

func ParseClothTextures(raw []byte) ([]string, error) {
	c := &cursor{b: raw}
	n := c.count(256)
	out := make([]string, n)
	for i := range out {
		p := c.take(256)
		for j, x := range p {
			if x == 0 {
				p = p[:j]
				break
			}
		}
		out[i] = string(p)
	}
	if c.err != nil {
		return nil, c.err
	}
	if c.off != len(raw) {
		return nil, fmt.Errorf("cloak: trailing texture bytes")
	}
	return out, nil
}
