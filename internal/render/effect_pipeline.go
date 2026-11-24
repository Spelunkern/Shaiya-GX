package render

import (
	"log"
	"math"
	"strings"
	"unsafe"

	"github.com/go-gl/gl/v4.5-core/gl"
)

const effectVS = `#version 450 core
layout(location=0) in vec3 pos;
layout(location=2) in vec2 uv;
struct Instance { mat4 mvp; vec4 color; vec4 pose; };
struct KeyVertex { vec4 position; vec4 uv; };
layout(std430, binding=3) readonly buffer Instances { Instance instances[]; };
layout(std430, binding=5) readonly buffer EffectKeys { KeyVertex keys[]; };
layout(std430, binding=6) readonly buffer ClipWorlds { mat4 worlds[]; };
layout(location=0) uniform vec4 clipPlane;
#ifdef EFFECT_INDIRECT
layout(std430, binding=7) readonly buffer Draws { uvec4 draws[]; };
layout(location=1) uniform int drawOffset;
flat out uvec2 vTexture;
#else
layout(location=1) uniform int instanceBase;
layout(location=2) uniform bool hasKeys;
#endif
out vec2 vUV;
out vec4 vColor;
void main() {
	#ifdef EFFECT_INDIRECT
	uvec4 draw = draws[drawOffset + gl_DrawIDARB];
	vTexture = draw.xy;
	int index = gl_BaseInstanceARB + gl_InstanceID;
	bool hasKeys = draw.z != 0u;
	int keyVertex = int(draw.z)-1 + gl_VertexID - int(draw.w);
	#else
	int index = instanceBase + gl_InstanceID;
	int keyVertex = gl_VertexID;
	#endif
	Instance placement = instances[index];
	vec4 local = vec4(pos, 1.0);
	vUV = uv;
	if (hasKeys) {
		KeyVertex a = keys[int(placement.pose.x) + keyVertex];
		KeyVertex b = keys[int(placement.pose.y) + keyVertex];
		precise vec3 point = a.position.xyz + (b.position.xyz - a.position.xyz) * placement.pose.z;
		precise vec2 coord = a.uv.xy + (b.uv.xy - a.uv.xy) * placement.pose.z;
		local = vec4(point, 1.0);
		vUV = coord;
	}
	gl_Position = placement.mvp * local;
	if (any(notEqual(clipPlane, vec4(0.0))))
		gl_ClipDistance[0] = dot(worlds[index] * local, clipPlane);
	vColor = placement.color;
}`

const effectFS = `#version 450 core
#ifdef EFFECT_INDIRECT
flat in uvec2 vTexture;
#else
layout(binding=0) uniform sampler2D tex0;
#endif
in vec2 vUV;
in vec4 vColor;
out vec4 frag;
void main() {
	#ifdef EFFECT_INDIRECT
	frag = texture(sampler2D(vTexture), vUV) * vColor;
	#else
	frag = texture(tex0, vUV) * vColor;
	#endif
}
`

// The authored effect material is unlit and unfogged. It only consumes a
// projected transform, its color, and optional key selection. The full world
// transform is uploaded separately when a reflection needs the clip plane.
type effectPlacement struct {
	mvp   M4
	color [4]float32
	pose  EffectPose
}

type effectUpload struct {
	id       uint32
	bytes    uint64
	capacity uint64
}

func (u *effectUpload) upload(size int, data unsafe.Pointer) {
	if size <= 0 {
		return
	}
	if u.id == 0 {
		gl.CreateBuffers(1, &u.id)
	}
	if uint64(size) > u.capacity {
		capBytes := max(uint64(4096), u.capacity)
		for capBytes < uint64(size) {
			capBytes *= 2
		}
		removeGPUBytes(u.capacity)
		u.capacity = capBytes
		addGPUBytes(u.capacity)
	}
	gl.NamedBufferData(u.id, int(u.capacity), nil, gl.STREAM_DRAW)
	if data != nil {
		gl.NamedBufferSubData(u.id, 0, size, data)
	}
	u.bytes = uint64(size)
}

func (u *effectUpload) delete() {
	if u.id != 0 {
		gl.DeleteBuffers(1, &u.id)
		removeGPUBytes(u.capacity)
		*u = effectUpload{}
	}
}

type effectPipeline struct {
	tried           bool
	program         uint32
	placements      []effectPlacement
	instances, clip effectUpload
	indirect        effectIndirect
}

func (p *effectPipeline) init() {
	p.tried = true
	program, err := Program("effect instancing", effectVS, effectFS)
	if err != nil {
		log.Printf("compact effects unavailable: %v", err)
		return
	}
	p.program = program
	if skinIndirectSupported {
		const header = "#version 450 core\n"
		vs := strings.Replace(effectVS, header, header+"#extension GL_ARB_shader_draw_parameters : require\n#define EFFECT_INDIRECT\n", 1)
		fs := strings.Replace(effectFS, header, header+"#extension GL_ARB_bindless_texture : require\n#define EFFECT_INDIRECT\n", 1)
		indirect, err := Program("effect bindless indirect", vs, fs)
		if err != nil {
			log.Printf("effect multi-draw unavailable: %v", err)
		} else {
			p.indirect.program = indirect
		}
	}
}

func (p *effectPipeline) delete() {
	p.instances.delete()
	p.clip.delete()
	p.indirect.delete()
	if p.program != 0 {
		gl.DeleteProgram(p.program)
	}
	*p = effectPipeline{}
}

func (s *Scene3D) drawCompactEffects(q *effectQueue) bool {
	if len(q.worlds) < 32 && len(q.draws) < 8 {
		return false
	}
	p := &s.effectPipeline
	if !p.tried {
		p.init()
	}
	if p.program == 0 {
		return false
	}
	p.placements = p.placements[:0]
	relative := s.cameraValid && s.View == s.cameraView
	perspective := sparseEffectPerspective(s.projection)
	for i, world := range q.worlds {
		mvp := s.effectMVP(world, relative, perspective)
		p.placements = append(p.placements, effectPlacement{mvp, q.colors[i], q.poses[i]})
	}
	p.instances.upload(len(p.placements)*int(unsafe.Sizeof(effectPlacement{})), unsafe.Pointer(&p.placements[0]))
	gl.BindBufferBase(gl.SHADER_STORAGE_BUFFER, 3, p.instances.id)
	if s.ClipPlane != [4]float32{} {
		p.clip.upload(len(q.worlds)*int(unsafe.Sizeof(M4{})), unsafe.Pointer(&q.worlds[0]))
		gl.BindBufferBase(gl.SHADER_STORAGE_BUFFER, 6, p.clip.id)
	}
	if p.indirect.plan(q.draws) {
		p.indirect.draw(s)
	} else {
		p.drawSerial(s, q.draws)
	}
	gl.UseProgram(s.prog)
	return true
}

// The ordinary perspective projection can add jitter in the third row or
// reverse depth without introducing other nonzero entries.
func sparseEffectPerspective(p M4) bool {
	return p[1] == 0 && p[2] == 0 && p[3] == 0 && p[4] == 0 && p[6] == 0 && p[7] == 0 &&
		p[11] == 1 && p[12] == 0 && p[13] == 0 && p[15] == 0
}

func (s *Scene3D) effectMVP(world M4, relative, perspective bool) M4 {
	mv := world.MulAffine(s.View)
	if relative {
		t := cameraRelativeTranslation(world, s.View, s.cameraEye)
		mv[12], mv[13], mv[14] = t[0], t[1], t[2]
	}
	p := s.projection
	// Omitting products with zero is equivalent for finite operands. A
	// nonfinite sum uses the general path, including overflow in this guard.
	sum := mv[0] + mv[1] + mv[2] + mv[3] + mv[4] + mv[5] + mv[6] + mv[7] +
		mv[8] + mv[9] + mv[10] + mv[11] + mv[12] + mv[13] + mv[14] + mv[15]
	if !perspective || math.Float32bits(sum)&0x7f800000 == 0x7f800000 {
		return mv.Mul(p)
	}
	var out M4
	for i := 0; i < len(out); i += 4 {
		z := mv[i+2]
		out[i] = float32(mv[i]*p[0]) + float32(z*p[8])
		out[i+1] = float32(mv[i+1]*p[5]) + float32(z*p[9])
		out[i+2] = float32(z*p[10]) + float32(mv[i+3]*p[14])
		out[i+3] = z
	}
	return out
}

func (p *effectPipeline) drawSerial(s *Scene3D, draws []queuedEffect) {
	gl.UseProgram(p.program)
	gl.Uniform4fv(0, 1, &s.ClipPlane[0])
	gl.Uniform1i(2, 0)
	var boundKeys uint32
	for _, draw := range draws {
		gl.Uniform1i(1, int32(draw.first))
		var keys uint32
		if draw.geometry != nil {
			keys = draw.geometry.keys
		}
		if keys != boundKeys {
			if keys == 0 {
				gl.Uniform1i(2, 0)
			} else {
				gl.BindBufferBase(gl.SHADER_STORAGE_BUFFER, 5, keys)
				if boundKeys == 0 {
					gl.Uniform1i(2, 1)
				}
			}
			boundKeys = keys
		}
		s.effectMaterial(draw.texture, draw.src, draw.dst, draw.repeat)
		gl.BindVertexArray(draw.mesh.vao)
		gl.DrawElementsInstanced(gl.TRIANGLES, draw.mesh.Count, draw.mesh.indexType, nil, int32(draw.end-draw.first))
		drawCalls++
	}
}
