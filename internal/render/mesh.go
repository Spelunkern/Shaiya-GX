package render

import (
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
	vao, vbo, ebo uint32
	Count         int32
}

func NewMesh(v []Vertex, idx []uint32) *Mesh {
	m := &Mesh{Count: int32(len(idx))}
	if len(v) == 0 || len(idx) == 0 {
		return m
	}
	gl.CreateBuffers(1, &m.vbo)
	gl.NamedBufferStorage(m.vbo, len(v)*vertexSize, unsafe.Pointer(&v[0]), 0)
	gl.CreateBuffers(1, &m.ebo)
	gl.NamedBufferStorage(m.ebo, len(idx)*4, unsafe.Pointer(&idx[0]), 0)
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
	return m
}

func (m *Mesh) Draw() {
	if m.Count == 0 {
		return
	}
	gl.BindVertexArray(m.vao)
	gl.DrawElements(gl.TRIANGLES, m.Count, gl.UNSIGNED_INT, nil)
}
