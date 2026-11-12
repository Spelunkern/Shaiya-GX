package render

import (
	"unsafe"

	"github.com/go-gl/gl/v4.5-core/gl"
)

// Delete releases the scene's streaming buffer and shader on the GL thread.
func (s *Scene3D) Delete() {
	if s == nil || s.prog == 0 {
		return
	}
	if s.instanceBuffer != 0 {
		gl.DeleteBuffers(1, &s.instanceBuffer)
		removeGPUBytes(s.instanceBytes)
		s.instanceBuffer, s.instanceBytes = 0, 0
	}
	if s.effectColorBuffer != 0 {
		gl.DeleteBuffers(1, &s.effectColorBuffer)
		removeGPUBytes(s.effectColorBytes)
		s.effectColorBuffer, s.effectColorBytes = 0, 0
	}
	if s.effectStream != nil {
		s.effectStream.Delete()
	}
	s.ReleaseWaterBackground()
	if s.terrainIndirect != 0 {
		gl.DeleteBuffers(1, &s.terrainIndirect)
		removeGPUBytes(s.terrainIndirectBytes)
	}
	gl.DeleteProgram(s.prog)
	s.prog = 0
}

// SetInstances uploads one model's placements once; all its material parts
// reuse them. Camera products are computed before local vertices, preserving
// the precision fix for animated and static models.
func (s *Scene3D) SetInstances(worlds []M4) {
	s.instanceData = s.instanceData[:0]
	for _, world := range worlds {
		mv := s.modelView(world)
		s.instanceData = append(s.instanceData, world, mv, mv.Mul(s.projection))
	}
	s.instanceCount = len(worlds)
	if s.instanceCount == 0 {
		return
	}
	if s.instanceBuffer == 0 {
		gl.CreateBuffers(1, &s.instanceBuffer)
	}
	bytes := uint64(len(s.instanceData) * 64)
	gl.NamedBufferData(s.instanceBuffer, int(bytes), unsafe.Pointer(&s.instanceData[0][0]), gl.STREAM_DRAW)
	removeGPUBytes(s.instanceBytes)
	s.instanceBytes = bytes
	addGPUBytes(bytes)
}
func (s *Scene3D) DrawInstanced(m *Mesh, t *Texture, alphaRef float32) {
	if s.instanceCount == 0 || m == nil || m.Count == 0 {
		return
	}
	gl.Uniform1i(s.uInstanced, 1)
	gl.Uniform1i(s.uMode, 0)
	gl.Uniform1i(s.uLit, 1)
	gl.Uniform1f(s.uRef, alphaRef)
	s.blend(false)
	bindDiffuse(t)
	s.drawInstances(m)
}

func (s *Scene3D) drawInstances(m *Mesh) {
	if m.instanceBuffer != s.instanceBuffer {
		gl.VertexArrayVertexBuffer(m.vao, 1, s.instanceBuffer, 0, 3*64)
		for i := uint32(0); i < 12; i++ {
			location := 4 + i
			gl.EnableVertexArrayAttrib(m.vao, location)
			gl.VertexArrayAttribFormat(m.vao, location, 4, gl.FLOAT, false, i*16)
			gl.VertexArrayAttribBinding(m.vao, location, 1)
		}
		gl.VertexArrayBindingDivisor(m.vao, 1, 1)
		m.instanceBuffer = s.instanceBuffer
	}
	gl.BindVertexArray(m.vao)
	gl.DrawElementsInstanced(gl.TRIANGLES, m.Count, gl.UNSIGNED_INT, nil, int32(s.instanceCount))
	drawCalls++
}
