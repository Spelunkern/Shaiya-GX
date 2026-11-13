package render

import (
	"math"
	"unsafe"

	"github.com/go-gl/gl/v4.5-core/gl"
)

// Vertex is the common 3D vertex: position, normal, two UV sets.
type Vertex struct {
	Pos, Normal V3
	UV, UV2     [2]float32
}

const vertexSize = int(unsafe.Sizeof(Vertex{}))

// Mesh is an indexed triangle list on the GPU.
type Mesh struct {
	minimum, maximum V3
	boundsValid      bool
	instanceBuffer   uint32
	bytes            uint64
	vao, vbo, ebo    uint32
	Count            int32
	indexType        uint32
	indexBytes       uintptr
	vertexCapacity   int
}

func NewMesh(v []Vertex, idx []uint32) *Mesh {
	m := &Mesh{Count: int32(len(idx))}
	m.updateBounds(v)
	if len(v) == 0 || len(idx) == 0 {
		return m
	}
	gl.CreateBuffers(1, &m.vbo)
	gl.NamedBufferStorage(m.vbo, len(v)*vertexSize, unsafe.Pointer(&v[0]), 0)
	gl.CreateBuffers(1, &m.ebo)
	m.indexType, m.indexBytes = uploadMeshIndices(m.ebo, idx)
	m.bindLayout()
	m.bytes = uint64(len(v)*vertexSize + len(idx)*int(m.indexBytes))
	addGPUBytes(m.bytes)
	return m
}

// NewAnimatedMesh stores the topology once and reserves a fixed vertex
// buffer for successive poses of the same mesh.
func NewAnimatedMesh(vertices int, idx []uint32) *Mesh {
	m := &Mesh{Count: int32(len(idx)), vertexCapacity: vertices}
	if vertices <= 0 || len(idx) == 0 {
		m.Count = 0
		return m
	}
	gl.CreateBuffers(1, &m.vbo)
	gl.NamedBufferStorage(m.vbo, vertices*vertexSize, nil, gl.DYNAMIC_STORAGE_BIT)
	gl.CreateBuffers(1, &m.ebo)
	m.indexType, m.indexBytes = uploadMeshIndices(m.ebo, idx)
	m.bindLayout()
	m.bytes = uint64(vertices*vertexSize + len(idx)*int(m.indexBytes))
	addGPUBytes(m.bytes)
	return m
}

// UpdateVertices changes only the pose; indices and storage remain intact.
func (m *Mesh) UpdateVertices(v []Vertex) {
	if m == nil || m.vbo == 0 || len(v) != m.vertexCapacity || len(v) == 0 {
		return
	}
	m.updateBounds(v)
	gl.NamedBufferSubData(m.vbo, 0, len(v)*vertexSize, unsafe.Pointer(&v[0]))
}

func (m *Mesh) bindLayout() {
	gl.CreateVertexArrays(1, &m.vao)
	gl.VertexArrayVertexBuffer(m.vao, 0, m.vbo, 0, int32(vertexSize))
	gl.VertexArrayElementBuffer(m.vao, m.ebo)
	attrs := []struct {
		n   int32
		off uint32
	}{{3, 0}, {3, 12}, {2, 24}, {2, 32}}
	for i, a := range attrs {
		gl.EnableVertexArrayAttrib(m.vao, uint32(i))
		gl.VertexArrayAttribFormat(m.vao, uint32(i), a.n, gl.FLOAT, false, a.off)
		gl.VertexArrayAttribBinding(m.vao, uint32(i), 0)
	}
}

// NewStreamMesh creates a reusable triangle stream; it owns no immutable storage.
func NewStreamMesh() *Mesh {
	m := &Mesh{}
	gl.CreateBuffers(1, &m.vbo)
	gl.CreateBuffers(1, &m.ebo)
	m.bindLayout()
	return m
}

// Stream replaces the buffers of a mesh created by NewStreamMesh.
func (m *Mesh) Stream(v []Vertex, idx []uint32) {
	m.indexType, m.indexBytes = gl.UNSIGNED_INT, 4
	m.updateBounds(v)
	m.Count = int32(len(idx))
	if len(v) == 0 || len(idx) == 0 {
		m.Count = 0
		return
	}
	gl.NamedBufferData(m.vbo, len(v)*vertexSize, unsafe.Pointer(&v[0]), gl.STREAM_DRAW)
	gl.NamedBufferData(m.ebo, len(idx)*4, unsafe.Pointer(&idx[0]), gl.STREAM_DRAW)
	removeGPUBytes(m.bytes)
	m.bytes = uint64(len(v)*vertexSize + len(idx)*4)
	addGPUBytes(m.bytes)
}

func (m *Mesh) Draw() {
	if m.Count == 0 {
		return
	}
	gl.BindVertexArray(m.vao)
	gl.DrawElements(gl.TRIANGLES, m.Count, m.indexType, nil)
	drawCalls++
}

// Delete releases owned GPU storage; meshes referenced by shared assets must
// only be deleted after their last resident world cell is gone.
func (m *Mesh) Delete() {
	if m == nil {
		return
	}
	gl.DeleteVertexArrays(1, &m.vao)
	gl.DeleteBuffers(1, &m.vbo)
	gl.DeleteBuffers(1, &m.ebo)
	removeGPUBytes(m.bytes)
	m.bytes = 0
	m.vao, m.vbo, m.ebo, m.Count = 0, 0, 0, 0
}

func (m *Mesh) updateBounds(v []Vertex) {
	m.boundsValid = false
	if len(v) == 0 {
		return
	}
	lo, hi := v[0].Pos, v[0].Pos
	for _, p := range v {
		for axis, x := range p.Pos {
			if math.IsNaN(float64(x)) || math.IsInf(float64(x), 0) {
				return
			}
			lo[axis] = min(lo[axis], x)
			hi[axis] = max(hi[axis], x)
		}
	}
	m.minimum, m.maximum, m.boundsValid = lo, hi, true
}

// Preserve exact topology while halving index storage/bandwidth when its range
// fits uint16. Large meshes retain uint32; streamed buffers remain unchanged.
func uploadMeshIndices(buffer uint32, idx []uint32) (uint32, uintptr) {
	narrow := true
	for _, i := range idx {
		if i > 65535 {
			narrow = false
			break
		}
	}
	if narrow {
		indices := make([]uint16, len(idx))
		for i, v := range idx {
			indices[i] = uint16(v)
		}
		gl.NamedBufferStorage(buffer, len(indices)*2, unsafe.Pointer(&indices[0]), 0)
		return gl.UNSIGNED_SHORT, 2
	}
	gl.NamedBufferStorage(buffer, len(idx)*4, unsafe.Pointer(&idx[0]), 0)
	return gl.UNSIGNED_INT, 4
}

// Bounds is conservative and includes unused vertices, matching the uploaded mesh.
func (m *Mesh) Bounds() (V3, V3, bool) {
	if m == nil {
		return V3{}, V3{}, false
	}
	return m.minimum, m.maximum, m.boundsValid
}
