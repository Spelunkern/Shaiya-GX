package render

import (
	"unsafe"

	"github.com/go-gl/gl/v4.5-core/gl"
)

// SkinVertex is the GPU layout of a skinned vertex: two bone influences
// (EP5 3DC: weight w for bone 0, 1-w for bone 1).
type SkinVertex struct {
	Pos, Normal V3
	UV          [2]float32
	Weight      float32
	Bone0       float32
	Bone1       float32
}

const MaxBones = 128

const skinVS = `#version 450 core
layout(location=0) in vec3 pos;
layout(location=1) in vec3 nrm;
layout(location=2) in vec2 uv;
layout(location=3) in vec3 skin; // weight, bone0, bone1
uniform mat4 bones[128];
uniform mat4 world;
uniform mat4 viewProj;
uniform mat4 view;
uniform vec3 lightDir;
uniform float ambient;
out vec2 vUV;
out vec2 vUV2;
out float vDiffuse;
out float vDepth;
void main() {
	mat4 b0 = bones[int(skin.y)];
	mat4 b1 = bones[int(skin.z)];
	vec4 p = skin.x * (b0 * vec4(pos, 1.0)) + (1.0 - skin.x) * (b1 * vec4(pos, 1.0));
	vec3 n = skin.x * (b0 * vec4(nrm, 0.0)).xyz + (1.0 - skin.x) * (b1 * vec4(nrm, 0.0)).xyz;
	vec4 w = world * vec4(p.xyz, 1.0);
	gl_Position = viewProj * w;
	vDepth = (view * w).z;
	vUV = uv; vUV2 = vec2(0.0);
	n = normalize((world * vec4(n, 0.0)).xyz);
	vDiffuse = clamp(ambient + max(0.0, dot(n, -normalize(lightDir))), 0.0, 1.0);
}`

// SkinMesh is a skinned triangle list.
type SkinMesh struct {
	vao, vbo, ebo uint32
	Count         int32
}

func NewSkinMesh(v []SkinVertex, idx []uint32) *SkinMesh {
	m := &SkinMesh{Count: int32(len(idx))}
	if len(v) == 0 || len(idx) == 0 {
		return m
	}
	size := int(unsafe.Sizeof(SkinVertex{}))
	gl.CreateBuffers(1, &m.vbo)
	gl.NamedBufferStorage(m.vbo, len(v)*size, unsafe.Pointer(&v[0]), 0)
	gl.CreateBuffers(1, &m.ebo)
	gl.NamedBufferStorage(m.ebo, len(idx)*4, unsafe.Pointer(&idx[0]), 0)
	gl.CreateVertexArrays(1, &m.vao)
	gl.VertexArrayVertexBuffer(m.vao, 0, m.vbo, 0, int32(size))
	gl.VertexArrayElementBuffer(m.vao, m.ebo)
	for i, a := range []struct {
		n   int32
		off uint32
	}{{3, 0}, {3, 12}, {2, 24}, {3, 32}} {
		gl.EnableVertexArrayAttrib(m.vao, uint32(i))
		gl.VertexArrayAttribFormat(m.vao, uint32(i), a.n, gl.FLOAT, false, a.off)
		gl.VertexArrayAttribBinding(m.vao, uint32(i), 0)
	}
	return m
}

// Skinner draws skinned meshes with the shared fragment stage.
type Skinner struct {
	prog                                           uint32
	uBones, uWorld, uViewProj, uView, uLight, uAmb int32
	uMode, uRef, uFogColor, uFogRange              int32
}

func NewSkinner() (*Skinner, error) {
	p, err := Program(skinVS, sceneFS)
	if err != nil {
		return nil, err
	}
	u := func(n string) int32 { return gl.GetUniformLocation(p, gl.Str(n+"\x00")) }
	return &Skinner{prog: p, uBones: u("bones"), uWorld: u("world"), uViewProj: u("viewProj"), uView: u("view"),
		uLight: u("lightDir"), uAmb: u("ambient"), uMode: u("mode"), uRef: u("alphaRef"),
		uFogColor: u("fogColor"), uFogRange: u("fogRange")}, nil
}

// Begin binds the program with the scene's camera, light and fog.
func (k *Skinner) Begin(s *Scene3D, ambient float32) {
	gl.UseProgram(k.prog)
	vp := s.View.Mul(s.Proj)
	gl.UniformMatrix4fv(k.uViewProj, 1, false, &vp[0])
	gl.UniformMatrix4fv(k.uView, 1, false, &s.View[0])
	gl.Uniform3f(k.uLight, s.LightDir[0], s.LightDir[1], s.LightDir[2])
	gl.Uniform1f(k.uAmb, ambient)
	gl.Uniform3f(k.uFogColor, s.FogColor[0], s.FogColor[1], s.FogColor[2])
	gl.Uniform2f(k.uFogRange, s.FogStart, s.FogEnd)
	gl.Uniform1i(k.uMode, 0)
}

// SetPose uploads the per-bone skin matrices (B*F, row-vector).
func (k *Skinner) SetPose(world M4, bones []M4) {
	gl.UniformMatrix4fv(k.uWorld, 1, false, &world[0])
	if n := min(len(bones), MaxBones); n > 0 {
		gl.UniformMatrix4fv(k.uBones, int32(n), false, &bones[0][0])
	}
}

func (k *Skinner) Draw(m *SkinMesh, t *Texture, alphaRef float32) {
	if m.Count == 0 {
		return
	}
	gl.Uniform1f(k.uRef, alphaRef)
	gl.Disable(gl.BLEND)
	if t != nil {
		gl.BindTextureUnit(0, t.ID)
	}
	gl.BindVertexArray(m.vao)
	gl.DrawElements(gl.TRIANGLES, m.Count, gl.UNSIGNED_INT, nil)
}
