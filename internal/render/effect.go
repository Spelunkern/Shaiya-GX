package render

import (
	"unsafe"

	"github.com/go-gl/gl/v4.5-core/gl"
)

// DrawEffectBatch preserves particle order within one emitter while sharing
// geometry, material and draw call; each instance has its own tint.
func (s *Scene3D) DrawEffectBatch(m *Mesh, t *Texture, worlds []M4, colors [][4]float32, source, destination int32) {
	src, dst, ok := EFTBlend(source, destination)
	if !ok || t == nil || m == nil || m.Count == 0 || len(worlds) == 0 || len(worlds) != len(colors) {
		return
	}
	s.SetInstances(worlds)
	s.setInstanceColors(colors)
	gl.Uniform1i(s.uInstanced, 1)
	gl.Uniform1i(s.uEffectColors, 1)
	gl.Uniform1i(s.uMode, 3)
	gl.Uniform1i(s.uLit, 0)
	gl.Uniform2f(s.uUVOff, 0, 0)
	gl.Uniform4f(s.uTint, 1, 1, 1, 1)
	gl.Enable(gl.BLEND)
	gl.BlendFunc(src, dst)
	gl.DepthMask(false)
	t.SetWrap(true)
	gl.BindTextureUnit(0, t.ID)
	s.drawInstances(m)
	t.SetWrap(false)
	gl.Uniform1i(s.uEffectColors, 0)
}

func (s *Scene3D) setInstanceColors(colors [][4]float32) {
	if s.effectColorBuffer == 0 {
		gl.CreateBuffers(1, &s.effectColorBuffer)
	}
	bytes := uint64(len(colors) * 16)
	gl.NamedBufferData(s.effectColorBuffer, int(bytes), unsafe.Pointer(&colors[0][0]), gl.STREAM_DRAW)
	removeGPUBytes(s.effectColorBytes)
	s.effectColorBytes = bytes
	addGPUBytes(bytes)
	gl.BindBufferBase(gl.SHADER_STORAGE_BUFFER, 3, s.effectColorBuffer)
}

// DrawGroundItems uses the native unlit texture-factor colour while retaining
// scene fog and depth. Independent factors share a single instanced draw.
func (s *Scene3D) DrawGroundItems(m *Mesh, t *Texture, worlds []M4, colors [][4]float32) {
	if m == nil || m.Count == 0 || t == nil || len(worlds) == 0 || len(worlds) != len(colors) {
		return
	}
	s.SetInstances(worlds)
	s.setInstanceColors(colors)
	gl.Uniform1i(s.uInstanced, 1)
	gl.Uniform1i(s.uEffectColors, 1)
	gl.Uniform1i(s.uMode, 8)
	gl.Uniform1i(s.uLit, 0)
	gl.Uniform1f(s.uRef, .5)
	s.blend(false)
	gl.DepthMask(true)
	bindDiffuse(t)
	s.drawInstances(m)
	gl.Uniform1i(s.uEffectColors, 0)
}

// DrawEffectMesh shares one GPU stream across every 3DE particle and cast.
func (s *Scene3D) DrawEffectMesh(vertices []Vertex, indices []uint32, t *Texture, world M4, color [4]float32, source, destination int32) {
	if s.effectStream == nil {
		s.effectStream = NewStreamMesh()
	}
	s.effectStream.Stream(vertices, indices)
	s.DrawEffect(s.effectStream, t, world, color, source, destination)
}

// DrawEffectMeshBatch shares one sampled 3DE pose among consecutive
// particles, preserving their order and individual transforms and colors.
func (s *Scene3D) DrawEffectMeshBatch(vertices []Vertex, indices []uint32, t *Texture, worlds []M4, colors [][4]float32, source, destination int32) {
	if len(worlds) == 0 || len(worlds) != len(colors) {
		return
	}
	if s.effectStream == nil {
		s.effectStream = NewStreamMesh()
	}
	s.effectStream.Stream(vertices, indices)
	if len(worlds) == 1 {
		s.DrawEffect(s.effectStream, t, worlds[0], colors[0], source, destination)
	} else {
		s.DrawEffectBatch(s.effectStream, t, worlds, colors, source, destination)
	}
}

// EFTBlend follows the authored catalog at Game.exe 0x6ccc34, not D3D enums.
func EFTBlend(source, destination int32) (uint32, uint32, bool) {
	f := [...]uint32{gl.ZERO, gl.ONE, gl.SRC_COLOR, gl.ONE_MINUS_SRC_COLOR, gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA, gl.DST_ALPHA, gl.ONE_MINUS_DST_ALPHA, gl.DST_COLOR, gl.ONE_MINUS_DST_COLOR, gl.SRC_ALPHA_SATURATE, gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA}
	if source < 0 || source >= int32(len(f)) || destination < 0 || destination >= int32(len(f)) {
		return 0, 0, false
	}
	s, d := f[source], f[destination]
	if source == 11 {
		d = gl.ONE_MINUS_SRC_ALPHA
	} else if source == 12 {
		d = gl.SRC_ALPHA
	}
	return s, d, true
}

// DrawEffect renders texture * vertex-equivalent tint, depth tested without
// depth writes (0x55cb10, native_effect_ports.cpp and native_controller_state.cpp).
func (s *Scene3D) DrawEffect(m *Mesh, t *Texture, world M4, color [4]float32, source, destination int32) {
	src, dst, ok := EFTBlend(source, destination)
	if !ok || t == nil {
		return
	}
	s.setWorld(world)
	gl.Uniform1i(s.uMode, 3)
	gl.Uniform1i(s.uLit, 0)
	gl.Uniform2f(s.uUVOff, 0, 0)
	gl.Uniform4fv(s.uTint, 1, &color[0])
	gl.Enable(gl.BLEND)
	gl.BlendFunc(src, dst)
	gl.DepthMask(false)
	t.SetWrap(true)
	gl.BindTextureUnit(0, t.ID)
	m.Draw()
	t.SetWrap(false)
}
