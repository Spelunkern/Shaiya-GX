package render

import (
	"log"
	"strings"
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

// StealthOpacity gives concealment its subtle fade while preserving original colors.
// Native texture factor replaces the authored texture alpha; multiplying
// that alpha made otherwise opaque clothing disappear.
const StealthOpacity float32 = .65

// GhostOpacity is the spirit of a dead player: Game.exe 0x414fb0 draws state-2
// characters with texture factor 0x64FFFFFF, an alpha of 100/255.
const GhostOpacity float32 = 100.0 / 255

const MaxBones = 256 // EP5 bone indices occupy one byte.
const skinVertexStride = int(unsafe.Sizeof(SkinVertex{}))

const skinVS = `#version 450 core
layout(location=0) in vec3 pos;
layout(location=1) in vec3 nrm;
layout(location=2) in vec2 uv;
layout(location=3) in vec3 skin; // weight, bone0, bone1
layout(std140, binding=0) uniform BonePalette { mat4 bones[256]; };
uniform int boneCount;
struct CrowdInstance { mat4 model; mat4 modelView; mat4 modelViewProj; ivec4 palette; };
layout(std430, binding=4) readonly buffer CrowdInstances { CrowdInstance instances[]; };
layout(std430, binding=5) readonly buffer CrowdPalettes { mat4 crowdBones[]; };
#ifdef SKIN_INDIRECT
struct SkinMaterial { uvec2 handle; float alphaRef; float ambient; };
layout(std430, binding=6) readonly buffer CrowdMaterials { SkinMaterial materials[]; };
uniform int drawOffset;
flat out uvec2 vSkinTexture;
flat out float vSkinAlphaRef;
mat4 skinBone(int i, int base) { return crowdBones[base+i]; }
#else
uniform int crowdBase;
mat4 skinBone(int i, int base) {
	return crowdBase == 0 ? bones[i] : crowdBones[base+i];
}
#endif
uniform mat4 world;
uniform mat4 worldViewProj;
uniform mat4 worldView;
uniform vec4 clipPlane;
uniform vec3 lightDir;
uniform float ambient;
uniform float actorOpacity;
out vec2 vUV;
out vec2 vUV2;
out float vDiffuse;
out float vDepth;
out vec4 vEffectColor;
void main() {
	vec4 p = vec4(pos, 1.0);
	vec3 n = nrm;
	int count = boneCount;
	int base = 0;
	float instanceFade = 0.0;
	float materialAmbient = ambient;
	mat4 model = world, mv = worldView, mvp = worldViewProj;
	#ifdef SKIN_INDIRECT
	SkinMaterial material = materials[drawOffset+gl_DrawIDARB];
	vSkinTexture = material.handle;
	vSkinAlphaRef = material.alphaRef;
	materialAmbient = material.ambient;
	{
		CrowdInstance instance = instances[gl_BaseInstanceARB+gl_InstanceID];
	#else
	if (crowdBase != 0) {
		CrowdInstance instance = instances[crowdBase-1+gl_InstanceID];
	#endif
		count = instance.palette.y;
		base = instance.palette.x;
		instanceFade = intBitsToFloat(instance.palette.z);
		model = instance.model;
		mv = instance.modelView;
		mvp = instance.modelViewProj;
	}
	vEffectColor = vec4(instanceFade, 1.0, 1.0, actorOpacity);
	if (count > 0) {
		p = vec4(0.0); n = vec3(0.0);
		if (skin.x > 0.0) {
			mat4 b = skinBone(clamp(int(skin.y), 0, count-1), base);
			p += skin.x * (b * vec4(pos, 1.0));
			n += skin.x * (b * vec4(nrm, 0.0)).xyz;
		}
		if (skin.x < 1.0) {
			mat4 b = skinBone(clamp(int(skin.z), 0, count-1), base);
			p += (1.0-skin.x) * (b * vec4(pos, 1.0));
			n += (1.0-skin.x) * (b * vec4(nrm, 0.0)).xyz;
		}
	}
	vec4 local = vec4(p.xyz, 1.0);
	gl_Position = mvp * local;
	gl_ClipDistance[0] = dot(model * local, clipPlane);
	vDepth = (mv * local).z;
	vUV = uv; vUV2 = vec2(0.0);
	n = normalize((model * vec4(n, 0.0)).xyz);
	vDiffuse = clamp(materialAmbient + max(0.0, dot(n, -normalize(lightDir))), 0.0, 1.0);
}`

// SkinMesh is a skinned triangle list.
type SkinMesh struct {
	vao, vbo, ebo uint32
	Count         int32
	vertexCount   int
	indexType     uint32
	bytes         uint64
}

func NewSkinMesh(v []SkinVertex, idx []uint32) *SkinMesh {
	m := &SkinMesh{Count: int32(len(idx)), vertexCount: len(v)}
	if len(v) == 0 || len(idx) == 0 {
		return m
	}
	gl.CreateBuffers(1, &m.vbo)
	gl.NamedBufferStorage(m.vbo, len(v)*skinVertexStride, unsafe.Pointer(&v[0]), 0)
	gl.CreateBuffers(1, &m.ebo)
	var indexBytes uintptr
	m.indexType, indexBytes = uploadMeshIndices(m.ebo, idx)
	m.bytes = uint64(len(v)*skinVertexStride + len(idx)*int(indexBytes))
	addGPUBytes(m.bytes)
	gl.CreateVertexArrays(1, &m.vao)
	bindSkinMeshLayout(m.vao, m.vbo, m.ebo)
	return m
}

func bindSkinMeshLayout(vao, vbo, ebo uint32) {
	gl.VertexArrayVertexBuffer(vao, 0, vbo, 0, int32(skinVertexStride))
	gl.VertexArrayElementBuffer(vao, ebo)
	for i, a := range []struct {
		n   int32
		off uint32
	}{{3, 0}, {3, 12}, {2, 24}, {3, 32}} {
		gl.EnableVertexArrayAttrib(vao, uint32(i))
		gl.VertexArrayAttribFormat(vao, uint32(i), a.n, gl.FLOAT, false, a.off)
		gl.VertexArrayAttribBinding(vao, uint32(i), 0)
	}
}

func (m *SkinMesh) Delete() {
	if m == nil {
		return
	}
	gl.DeleteVertexArrays(1, &m.vao)
	gl.DeleteBuffers(1, &m.vbo)
	gl.DeleteBuffers(1, &m.ebo)
	removeGPUBytes(m.bytes)
	*m = SkinMesh{}
}

// Skinner draws skinned meshes with the shared fragment stage.
type Skinner struct {
	uniformWorld, uniformView, uniformVP               M4
	camera                                             *Scene3D
	view, projection                                   M4
	palette                                            uint32
	paletteStride, paletteSlots, paletteSlot           int
	paletteBytes                                       uint64
	prog                                               uint32
	uBoneCount, uWorld, uViewProj, uView, uLight, uAmb int32
	uMode, uRef, uFogColor, uFogRange                  int32
	uClipPlane                                         int32
	uOpacity                                           int32
	uCorpseFade                                        int32
	opacity                                            float32
	uCrowdBase                                         int32
	crowdBuffers                                       [3]uint32
	crowdBytes                                         [3]uint64
	indirect                                           *Skinner
	uCrowdDrawOffset                                   int32
}

func NewSkinner() (*Skinner, error) {
	p, err := Program("skinned instancing", skinVS, sceneFS)
	if err != nil {
		return nil, err
	}
	var align int32
	gl.GetIntegerv(gl.UNIFORM_BUFFER_OFFSET_ALIGNMENT, &align)
	if align < 256 {
		align = 256
	}
	stride := ((MaxBones*64 + int(align) - 1) / int(align)) * int(align)
	const slots = 64
	totalBytes := stride * slots
	var palette uint32
	gl.CreateBuffers(1, &palette)
	gl.NamedBufferData(palette, totalBytes, nil, gl.STREAM_DRAW)
	addGPUBytes(uint64(totalBytes))
	k := skinProgram(p, palette, stride, slots, uint64(totalBytes))
	if skinIndirectSupported {
		vertex := strings.Replace(skinVS, "#version 450 core\n", "#version 450 core\n#extension GL_ARB_shader_draw_parameters : require\n#define SKIN_INDIRECT\n", 1)
		fragment := strings.Replace(sceneFS, "#version 450 core\n", "#version 450 core\n#extension GL_ARB_bindless_texture : require\n#define SKIN_INDIRECT\n", 1)
		indirect, err := Program("skinned bindless indirect", vertex, fragment)
		if err != nil {
			log.Printf("skinned multi-draw unavailable: %v", err)
		} else {
			k.indirect = skinProgram(indirect, palette, stride, slots, uint64(totalBytes))
		}
	}
	return k, nil
}

func skinProgram(p, palette uint32, stride, slots int, paletteBytes uint64) *Skinner {
	u := func(n string) int32 { return gl.GetUniformLocation(p, gl.Str(n+"\x00")) }
	return &Skinner{prog: p, uCrowdBase: u("crowdBase"), uCrowdDrawOffset: u("drawOffset"), palette: palette, paletteStride: stride, paletteSlots: slots, paletteBytes: paletteBytes, uBoneCount: u("boneCount"), uWorld: u("world"), uViewProj: u("worldViewProj"), uView: u("worldView"),
		uLight: u("lightDir"), uAmb: u("ambient"), uMode: u("mode"), uRef: u("alphaRef"),
		uFogColor: u("fogColor"), uFogRange: u("fogRange"), uClipPlane: u("clipPlane"), uOpacity: u("actorOpacity"), uCorpseFade: u("corpseFade")}
}

// Begin binds the program with the scene's camera, light and fog.
func (k *Skinner) Begin(s *Scene3D, ambient float32) {
	// A creature can follow UI previews, shadows or transparent emitters.
	// Establish its own depth contract instead of inheriting the previous pass.
	gl.Enable(gl.DEPTH_TEST)
	s.applyDepthMode()
	gl.DepthMask(true)
	gl.Disable(gl.BLEND)
	gl.Disable(gl.CULL_FACE)
	gl.Disable(gl.POLYGON_OFFSET_FILL)
	gl.UseProgram(k.prog)
	k.SetOpacity(1)
	k.SetCorpseFade(0)
	gl.BindBufferRange(gl.UNIFORM_BUFFER, 0, k.palette, 0, k.paletteStride)
	gl.Uniform4fv(k.uClipPlane, 1, &s.ClipPlane[0])
	k.view, k.projection = s.View, s.depthProjection(s.Proj)
	k.camera = s
	gl.Uniform3f(k.uLight, s.LightDir[0], s.LightDir[1], s.LightDir[2])
	gl.Uniform1f(k.uAmb, ambient)
	gl.Uniform3f(k.uFogColor, s.FogColor[0], s.FogColor[1], s.FogColor[2])
	gl.Uniform2f(k.uFogRange, s.FogStart, s.FogEnd)
	// Native skin uses material/actor opacity, not DDS alpha, after the
	// alpha-test decision. Low-alpha shoulder textures must still write an
	// opaque surface into offscreen world and portrait buffers.
	gl.Uniform1i(k.uMode, 10)
	if s.Shadow {
		gl.Uniform1i(k.uMode, 7)
	}
}

// SetPose uploads the per-bone skin matrices (B*F, row-vector).
func (k *Skinner) SetPose(world M4, bones []M4) {
	k.SetWorld(world)
	k.SetPalette(bones)
}

// SetWorld sets the transforms shared by all parts of an actor.
func (k *Skinner) SetWorld(world M4) {
	mv := k.camera.modelView(world)
	mvp := mv.Mul(k.projection)
	k.uniformWorld, k.uniformView, k.uniformVP = world, mv, mvp
	gl.UniformMatrix4fv(k.uWorld, 1, false, &k.uniformWorld[0])
	gl.UniformMatrix4fv(k.uView, 1, false, &k.uniformView[0])
	gl.UniformMatrix4fv(k.uViewProj, 1, false, &k.uniformVP[0])
}

// SetPalette selects the inverse-bind palette for subsequent parts.
func (k *Skinner) SetPalette(bones []M4) {
	n := min(len(bones), MaxBones)
	gl.Uniform1i(k.uBoneCount, int32(n))
	if n > 0 {
		if k.paletteSlot >= k.paletteSlots {
			gl.NamedBufferData(k.palette, int(k.paletteBytes), nil, gl.STREAM_DRAW)
			k.paletteSlot = 0
		}
		offset := k.paletteSlot * k.paletteStride
		gl.NamedBufferSubData(k.palette, offset, n*64, unsafe.Pointer(&bones[0][0]))
		gl.BindBufferRange(gl.UNIFORM_BUFFER, 0, k.palette, offset, k.paletteStride)
		k.paletteSlot++
	}
}

func (k *Skinner) Delete() {
	if k == nil || k.palette == 0 {
		return
	}
	gl.DeleteBuffers(1, &k.palette)
	gl.DeleteProgram(k.prog)
	if k.indirect != nil {
		gl.DeleteProgram(k.indirect.prog)
		k.indirect = nil
	}
	for i, buffer := range k.crowdBuffers {
		if buffer != 0 {
			gl.DeleteBuffers(1, &k.crowdBuffers[i])
			removeGPUBytes(k.crowdBytes[i])
		}
	}
	removeGPUBytes(k.paletteBytes)
	k.palette, k.prog = 0, 0
}

func (k *Skinner) Draw(m *SkinMesh, t *Texture, alphaRef float32) {
	k.draw(m, t, alphaRef, false)
}

// DrawTwoSided preserves D3DCULL_NONE for thin character surfaces.
func (k *Skinner) DrawTwoSided(m *SkinMesh, t *Texture, alphaRef float32) {
	k.draw(m, t, alphaRef, true)
}

func (k *Skinner) draw(m *SkinMesh, t *Texture, alphaRef float32, twoSided bool) {
	if m.Count == 0 {
		return
	}
	gl.Uniform1f(k.uRef, alphaRef)
	// Native skinned surfaces use clockwise front faces, including opaque NPCs.
	if twoSided {
		gl.Disable(gl.CULL_FACE)
	} else {
		gl.Enable(gl.CULL_FACE)
		gl.FrontFace(gl.CW)
		gl.CullFace(gl.BACK)
	}
	if k.opacity < 1 {
		gl.Enable(gl.BLEND)
		gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
		gl.DepthMask(true)
	} else {
		gl.Disable(gl.BLEND)
	}
	bindDiffuse(t)
	gl.BindVertexArray(m.vao)
	gl.DrawElements(gl.TRIANGLES, m.Count, m.indexType, nil)
	gl.DepthMask(true)
	gl.Disable(gl.CULL_FACE)
	gl.FrontFace(gl.CCW)
	gl.Disable(gl.BLEND)
	drawCalls++
}

// SetAmbient selects a part's native lighting. Cloak shoulders disable D3D
// lighting and therefore use their texture at full brightness.
func (k *Skinner) SetAmbient(ambient float32) {
	gl.Uniform1f(k.uAmb, ambient)
}

// SetOpacity applies a scoped ghost alpha; Begin restores opaque drawing.
func (k *Skinner) SetOpacity(alpha float32) {
	k.opacity = alpha
	gl.Uniform1f(k.uOpacity, alpha)
	if alpha < 1 {
		gl.Uniform1i(k.uMode, 10)
	}
}

// SetCorpseFade keeps depth writes and opaque drawing, with no blending pass.
func (k *Skinner) SetCorpseFade(fade float32) { gl.Uniform1f(k.uCorpseFade, min(1, max(0, fade))) }
