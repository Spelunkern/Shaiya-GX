// Package model reads character and creature formats: 3DC skinned meshes,
// ANI skeletal clips, MLT part tables and ALT action tables. Layouts follow
// Parsec and Phoenix tools/convert_character.py / convert_ani.py.
package model

import (
	"encoding/binary"
	"fmt"
	"math"
)

type cursor struct {
	b   []byte
	off int
	err error
}

func (c *cursor) take(n int) []byte {
	if c.err != nil || n < 0 || c.off+n > len(c.b) {
		if c.err == nil {
			c.err = fmt.Errorf("model: truncated at %d (+%d of %d)", c.off, n, len(c.b))
		}
		return make([]byte, max(n, 0))
	}
	p := c.b[c.off : c.off+n]
	c.off += n
	return p
}
func (c *cursor) u8() uint8    { return c.take(1)[0] }
func (c *cursor) u16() uint16  { return binary.LittleEndian.Uint16(c.take(2)) }
func (c *cursor) u32() uint32  { return binary.LittleEndian.Uint32(c.take(4)) }
func (c *cursor) i32() int32   { return int32(c.u32()) }
func (c *cursor) f32() float32 { return math.Float32frombits(c.u32()) }
func (c *cursor) count(size int) int {
	n := int(c.i32())
	if c.err == nil && (n < 0 || n*size > len(c.b)-c.off) {
		c.err = fmt.Errorf("model: bad count %d at %d", n, c.off-4)
		return 0
	}
	return n
}
func (c *cursor) str() string {
	b := c.take(c.count(1))
	for i, x := range b {
		if x == 0 {
			return string(b[:i])
		}
	}
	return string(b)
}

// Mat is a D3D row-major matrix as stored in the files (row-vector use).
type Mat [16]float32

func (c *cursor) mat() (m Mat) {
	for i := range m {
		m[i] = c.f32()
	}
	return
}

// Mesh3DC is a skinned character part. Bones hold each bone's matrix B
// (inverse bind); a vertex is skinned as v*B[i0]*F[i0]*w + v*B[i1]*F[i1]*(1-w).
type Mesh3DC struct {
	Bones    []Mat
	Vertices []SkinVertex
	Faces    [][3]uint16
}

type SkinVertex struct {
	Pos    [3]float32
	Weight float32
	Bones  [3]uint8
	Normal [3]float32
	UV     [2]float32
}

// Parse3DC reads the EP5 layout: version:i32 (0), bones, 40-byte vertices.
func Parse3DC(b []byte) (*Mesh3DC, error) {
	c := &cursor{b: b}
	if v := c.i32(); v == 444 {
		return nil, fmt.Errorf("model: EP6 3DC not supported")
	}
	m := &Mesh3DC{}
	nb := c.count(64)
	m.Bones = make([]Mat, nb)
	for i := range m.Bones {
		m.Bones[i] = c.mat()
	}
	nv := c.count(40)
	m.Vertices = make([]SkinVertex, nv)
	for i := range m.Vertices {
		v := &m.Vertices[i]
		v.Pos = [3]float32{c.f32(), c.f32(), c.f32()}
		v.Weight = c.f32()
		v.Bones = [3]uint8{c.u8(), c.u8(), c.u8()}
		c.u8()
		v.Normal = [3]float32{c.f32(), c.f32(), c.f32()}
		v.UV = [2]float32{c.f32(), c.f32()}
	}
	nf := c.count(6)
	m.Faces = make([][3]uint16, nf)
	for i := range m.Faces {
		m.Faces[i] = [3]uint16{c.u16(), c.u16(), c.u16()}
	}
	return m, c.err
}

// ANI is a skeletal clip at 30 frames per second.
type ANI struct {
	Start, End uint32
	Bones      []AniBone
}

type AniBone struct {
	Parent int32
	Rest   Mat // the clip's rest pose (global)
	Rot    []RotKey
	Trans  []TransKey
}

type RotKey struct {
	Frame uint32
	Q     [4]float32 // x y z w
}

type TransKey struct {
	Frame uint32
	T     [3]float32
}

func ParseANI(b []byte) (*ANI, error) {
	c := &cursor{b: b}
	if len(b) >= 6 && string(b[:6]) == "ANI_V2" {
		return nil, fmt.Errorf("model: EP6 ANI not supported")
	}
	a := &ANI{Start: c.u32(), End: c.u32()}
	n := int(c.u16())
	for i := 0; i < n && c.err == nil; i++ {
		bone := AniBone{Parent: c.i32(), Rest: c.mat()}
		nr := c.count(20)
		bone.Rot = make([]RotKey, nr)
		for j := range bone.Rot {
			bone.Rot[j] = RotKey{c.u32(), [4]float32{c.f32(), c.f32(), c.f32(), c.f32()}}
		}
		nt := c.count(16)
		bone.Trans = make([]TransKey, nt)
		for j := range bone.Trans {
			bone.Trans[j] = TransKey{c.u32(), [3]float32{c.f32(), c.f32(), c.f32()}}
		}
		a.Bones = append(a.Bones, bone)
	}
	return a, c.err
}

// MLT is a part table: record i -> mesh + texture + alpha mode.
type MLT struct {
	Meshes   []string
	Textures []string
	Records  []MLTRecord
}

type MLTRecord struct {
	Mesh, Texture int32
	Alpha         int32
}

func ParseMLT(b []byte) (*MLT, error) {
	c := &cursor{b: b}
	sig := string(c.take(3))
	if sig != "ML2" && sig != "MLT" {
		return nil, fmt.Errorf("model: bad MLT signature %q", sig)
	}
	m := &MLT{}
	for i, n := 0, c.count(5); i < n; i++ {
		m.Meshes = append(m.Meshes, c.str())
	}
	for i, n := 0, c.count(5); i < n; i++ {
		m.Textures = append(m.Textures, c.str())
	}
	n := c.count(12)
	for i := 0; i < n; i++ {
		m.Records = append(m.Records, MLTRecord{c.i32(), c.i32(), c.i32()})
	}
	return m, c.err
}
