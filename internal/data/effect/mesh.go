package effect

import (
	"encoding/binary"
	"fmt"
)

// Mesh3DE follows Parsec Shaiya/3de: a textured triangle mesh and complete
// position/UV snapshots, not skeletal animation or per-vertex deltas.
type Mesh3DE struct {
	Texture     string
	Vertices    []MeshVertex
	Indices     []uint16
	MaxKeyframe int32
	Frames      []MeshFrame
	Trailing    []byte // Unused bytes after the first object, as ignored by 0x557c00.
}
type MeshVertex struct {
	Position [3]float32
	Bone     int32
	UV       [2]float32
}
type FrameVertex struct {
	Position [3]float32
	UV       [2]float32
}
type MeshFrame struct {
	Key      int32
	Vertices []FrameVertex
}

func Parse3DE(b []byte) (*Mesh3DE, error) {
	if len(b) < 20 || len(b) > 64<<20 {
		return nil, fmt.Errorf("3DE: invalid length %d", len(b))
	}
	r := &reader{b: b}
	count := func(limit, size int) int {
		n := int(r.i32())
		if r.err != nil {
			return 0
		}
		if n < 0 || n > limit || n > (len(b)-r.off)/size {
			r.err = fmt.Errorf("invalid count %d at %d", n, r.off-4)
			return 0
		}
		return n
	}
	m := &Mesh3DE{Texture: r.str()}
	m.Vertices = make([]MeshVertex, count(65536, 24))
	for i := range m.Vertices {
		m.Vertices[i] = MeshVertex{r.vec(), r.i32(), [2]float32{r.f32(), r.f32()}}
	}
	m.Indices = make([]uint16, count(1<<20, 6)*3)
	for i := range m.Indices {
		index := binary.LittleEndian.Uint16(b[r.off:])
		r.off += 2
		if int(index) >= len(m.Vertices) {
			return nil, fmt.Errorf("3DE: vertex index %d outside %d vertices", index, len(m.Vertices))
		}
		m.Indices[i] = index
	}
	m.MaxKeyframe = r.i32()
	if m.MaxKeyframe < 0 {
		return nil, fmt.Errorf("3DE: negative maximum keyframe")
	}
	m.Frames = make([]MeshFrame, count(4096, 4+len(m.Vertices)*20))
	if len(m.Frames)*len(m.Vertices) > 1<<20 {
		return nil, fmt.Errorf("3DE: animation exceeds vertex budget")
	}
	for i := range m.Frames {
		f := &m.Frames[i]
		f.Key = r.i32()
		if f.Key < 0 || i > 0 && f.Key < m.Frames[i-1].Key {
			return nil, fmt.Errorf("3DE: invalid keyframe %d", f.Key)
		}
		f.Vertices = make([]FrameVertex, len(m.Vertices))
		for j := range f.Vertices {
			f.Vertices[j] = FrameVertex{r.vec(), [2]float32{r.f32(), r.f32()}}
		}
	}
	if r.err != nil {
		return nil, fmt.Errorf("3DE: %w", r.err)
	}
	m.Trailing = b[r.off:]
	return m, nil
}
