package render

import (
	"math"
	"unsafe"

	"github.com/go-gl/gl/v4.5-core/gl"
)

// RigidGroup identifies a range of world transforms in RigidCrowd.Draw that
// share one static mesh, texture and alpha threshold.
type RigidGroup struct {
	Mesh     *Mesh
	Texture  *Texture
	AlphaRef float32
	First    int
	Count    int
}

type rigidDrawRun struct {
	first int
	index uint32
}

// RigidCrowd packs immutable static attachment meshes and submits crowd
// weapons/shields through bindless multi-draw indirect when supported.
type RigidCrowd struct {
	vao, boundVertex, boundIndex    uint32
	vertices, indices               packedMeshBuffer
	meshes                          map[*Mesh]skinLocation
	placements, references, matrices effectUpload
	commands, materials             effectUpload
	identityRefs                    []uint32
	drawCommands                    []indexedDrawCommand
	drawMaterials                   [][5]uint32
	runs                            []rigidDrawRun
}

func (b *RigidCrowd) location(m *Mesh) (skinLocation, bool) {
	if loc, ok := b.meshes[m]; ok {
		return loc, m.vbo != 0
	}
	const maxVertices, maxIndices = 16 << 20, 8 << 20
	indexBytes := int(m.Count) * int(m.indexBytes)
	vertexBytes := int(m.bytes) - indexBytes
	at := (b.indices.used + 3) &^ 3
	if !m.immutable || m.vbo == 0 || indexBytes <= 0 || vertexBytes <= 0 ||
		b.vertices.used+vertexBytes > maxVertices || at+indexBytes > maxIndices {
		return skinLocation{}, false
	}
	if b.meshes == nil {
		b.meshes = make(map[*Mesh]skinLocation)
	}
	loc := skinLocation{b.vertices.used / vertexSize, at}
	b.vertices.copy(m.vbo, b.vertices.used, vertexBytes, maxVertices)
	b.indices.copy(m.ebo, at, indexBytes, maxIndices)
	b.meshes[m] = loc
	return loc, true
}

func (b *RigidCrowd) plan(groups []RigidGroup) bool {
	b.drawCommands, b.drawMaterials, b.runs = b.drawCommands[:0], b.drawMaterials[:0], b.runs[:0]
	for _, g := range groups {
		if g.Mesh == nil || g.Mesh.Count == 0 || g.Count <= 0 {
			continue
		}
		loc, ok := b.location(g.Mesh)
		if !ok {
			return false
		}
		t := diffuseTexture(g.Texture)
		diffuse := t.bindlessHandle(t.repeat)
		if diffuse == 0 {
			return false
		}
		m := g.Mesh
		if n := len(b.runs); n == 0 || b.runs[n-1].index != m.indexType {
			b.runs = append(b.runs, rigidDrawRun{len(b.drawCommands), m.indexType})
		}
		b.drawCommands = append(b.drawCommands, indexedDrawCommand{
			uint32(m.Count), uint32(g.Count), uint32(loc.index / int(m.indexBytes)), uint32(loc.vertex), uint32(g.First),
		})
		b.drawMaterials = append(b.drawMaterials, [5]uint32{
			uint32(diffuse), uint32(diffuse >> 32), 0, 0, math.Float32bits(g.AlphaRef),
		})
	}
	return len(b.drawCommands) > len(b.runs)
}

// Draw submits grouped rigid attachments through mapVS/mapFS multi-draw
// indirect. Returns false when unsupported or fallback is required.
func (b *RigidCrowd) Draw(s *Scene3D, groups []RigidGroup, worlds []M4) bool {
	if s.Shadow || len(groups) < 2 || len(worlds) == 0 {
		return false
	}
	p := s.mapPrograms()
	if p.draw == 0 || p.compute == 0 || !b.plan(groups) {
		return false
	}
	if b.vao == 0 {
		gl.CreateVertexArrays(1, &b.vao)
	}
	if b.boundVertex != b.vertices.id || b.boundIndex != b.indices.id {
		bindMeshLayout(b.vao, b.vertices.id, b.indices.id)
		b.boundVertex, b.boundIndex = b.vertices.id, b.indices.id
	}
	if len(worlds) > len(b.identityRefs) {
		old := len(b.identityRefs)
		for i := old; i < len(worlds); i++ {
			b.identityRefs = append(b.identityRefs, uint32(i))
		}
		b.references.upload(len(b.identityRefs)*4, unsafe.Pointer(&b.identityRefs[0]))
	}
	b.placements.upload(len(worlds)*64, unsafe.Pointer(&worlds[0]))
	b.commands.upload(len(b.drawCommands)*indexedDrawStride, unsafe.Pointer(&b.drawCommands[0]))
	b.materials.upload(len(b.drawMaterials)*20, unsafe.Pointer(&b.drawMaterials[0]))
	if size := len(worlds) * 192; uint64(size) > b.matrices.bytes {
		b.matrices.upload(max(size, int(b.matrices.bytes)*2), nil)
	}

	gl.UseProgram(p.compute)
	gl.BindBufferBase(gl.SHADER_STORAGE_BUFFER, 0, b.placements.id)
	gl.BindBufferBase(gl.SHADER_STORAGE_BUFFER, 1, b.references.id)
	gl.BindBufferBase(gl.SHADER_STORAGE_BUFFER, 2, b.matrices.id)
	gl.UniformMatrix4fv(0, 1, false, &s.View[0])
	gl.UniformMatrix4fv(4, 1, false, &s.projection[0])
	gl.Uniform3fv(8, 1, &s.cameraEye[0])
	relative := int32(0)
	if s.cameraValid && s.cameraView == s.View {
		relative = 1
	}
	gl.Uniform1i(9, relative)
	gl.Uniform1ui(10, uint32(len(worlds)))
	gl.DispatchCompute(uint32((len(worlds)+63)/64), 1, 1)
	gl.MemoryBarrier(gl.SHADER_STORAGE_BARRIER_BIT)

	gl.UseProgram(p.draw)
	gl.BindBufferBase(gl.SHADER_STORAGE_BUFFER, 3, b.materials.id)
	gl.Uniform4fv(0, 1, &s.ClipPlane[0])
	gl.Uniform3fv(1, 1, &s.LightDir[0])
	gl.Uniform1f(2, s.Ambient)
	gl.Uniform3fv(3, 1, &s.FogColor[0])
	gl.Uniform2f(4, s.FogStart, s.FogEnd)
	gl.Disable(gl.BLEND)
	gl.Disable(gl.CULL_FACE)
	gl.BindVertexArray(b.vao)
	gl.BindBuffer(gl.DRAW_INDIRECT_BUFFER, b.commands.id)
	for i, run := range b.runs {
		end := len(b.drawCommands)
		if i+1 < len(b.runs) {
			end = b.runs[i+1].first
		}
		gl.Uniform1i(6, int32(run.first))
		gl.MultiDrawElementsIndirect(gl.TRIANGLES, run.index, indexedDrawOffset(run.first*indexedDrawStride), int32(end-run.first), 0)
		drawCalls++
	}
	gl.BindBuffer(gl.DRAW_INDIRECT_BUFFER, 0)
	gl.UseProgram(s.prog)
	s.worldUniformsValid = false
	return true
}

// Delete releases GPU buffers and vertex array owned by the rigid crowd batch.
func (b *RigidCrowd) Delete() {
	if b.vao != 0 {
		gl.DeleteVertexArrays(1, &b.vao)
	}
	b.vertices.delete()
	b.indices.delete()
	for _, buf := range []*effectUpload{&b.placements, &b.references, &b.matrices, &b.commands, &b.materials} {
		buf.delete()
	}
	*b = RigidCrowd{}
}
