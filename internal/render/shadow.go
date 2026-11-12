package render

import (
	"fmt"

	"github.com/go-gl/gl/v4.5-core/gl"
)

// ShadowMap implements the original actor silhouette capture and projected
// alpha pass (0x5484e0..0x548f70, native_render_shaders projected_shadow_ps).
// One reusable target suffices: each capture is projected before the next actor.
type ShadowMap struct {
	bytes                   uint64
	fbo, depth, color, prog uint32
	receiver                *Mesh
}

const shadowVS = `#version 450 core
layout(location=0) in vec3 pos;
layout(location=2) in vec2 uv;
uniform mat4 viewProj;
uniform mat4 projector;
uniform mat4 view;
uniform vec4 clipPlane;
out vec4 projected;
out float depth;
out vec2 materialUV;
void main() {
 gl_Position = viewProj * vec4(pos,1);
 gl_ClipDistance[0] = dot(vec4(pos,1),clipPlane);
 projected = projector * vec4(pos,1);
 depth = (view * vec4(pos,1)).z;
 materialUV = uv;
}`
const shadowFS = `#version 450 core
layout(binding=0) uniform sampler2D image;
layout(binding=1) uniform sampler2D material;
uniform vec3 fogColor;
uniform vec2 fogRange;
uniform float opacity;
uniform float materialAlphaRef;
in vec4 projected;
in float depth;
in vec2 materialUV;
out vec4 frag;
void main() {
 if (materialAlphaRef >= 0.0 && texture(material, materialUV).a <= materialAlphaRef) discard;
 vec2 uv = projected.xy / projected.w * .5 + .5;
 if (any(lessThan(uv,vec2(0))) || any(greaterThan(uv,vec2(1)))) discard;
 float a = texture(image, uv).a * opacity;
 if (fogRange.y > fogRange.x)
  a *= clamp((fogRange.y-depth)/(fogRange.y-fogRange.x),0,1);
 frag = vec4(0,0,0,a);
}`

func NewShadowMap() (*ShadowMap, error) {
	p, err := Program(shadowVS, shadowFS)
	if err != nil {
		return nil, err
	}
	m := &ShadowMap{prog: p}
	gl.CreateFramebuffers(1, &m.fbo)
	gl.CreateRenderbuffers(1, &m.depth)
	gl.NamedRenderbufferStorage(m.depth, gl.DEPTH_COMPONENT24, 256, 256)
	gl.CreateTextures(gl.TEXTURE_2D, 1, &m.color)
	gl.TextureStorage2D(m.color, 1, gl.RGBA8, 256, 256)
	gl.TextureParameteri(m.color, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TextureParameteri(m.color, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TextureParameteri(m.color, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TextureParameteri(m.color, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	gl.NamedFramebufferTexture(m.fbo, gl.COLOR_ATTACHMENT0, m.color, 0)
	gl.NamedFramebufferRenderbuffer(m.fbo, gl.DEPTH_ATTACHMENT, gl.RENDERBUFFER, m.depth)
	if status := gl.CheckNamedFramebufferStatus(m.fbo, gl.FRAMEBUFFER); status != gl.FRAMEBUFFER_COMPLETE {
		m.Delete()
		return nil, fmt.Errorf("shadow framebuffer: %#x", status)
	}
	m.receiver = NewStreamMesh()
	m.bytes = 256 * 256 * 8
	addGPUBytes(m.bytes)
	return m, nil
}

// Capture preserves the caller's framebuffer, viewport, scissor and clear state.
// The draw callback owns its program, camera and depth state.
func (m *ShadowMap) Capture(draw func()) {
	var fbo, read int32
	var viewport, scissor [4]int32
	var clear [4]float32
	var depth float64
	gl.GetIntegerv(gl.DRAW_FRAMEBUFFER_BINDING, &fbo)
	gl.GetIntegerv(gl.READ_FRAMEBUFFER_BINDING, &read)
	gl.GetIntegerv(gl.VIEWPORT, &viewport[0])
	gl.GetIntegerv(gl.SCISSOR_BOX, &scissor[0])
	gl.GetFloatv(gl.COLOR_CLEAR_VALUE, &clear[0])
	gl.GetDoublev(gl.DEPTH_CLEAR_VALUE, &depth)
	scissored := gl.IsEnabled(gl.SCISSOR_TEST)
	gl.BindFramebuffer(gl.FRAMEBUFFER, m.fbo)
	gl.Viewport(0, 0, 256, 256)
	gl.Disable(gl.SCISSOR_TEST)
	gl.Disable(gl.BLEND)
	gl.DepthMask(true)
	gl.ClearColor(0, 0, 0, 0)
	gl.ClearDepth(1)
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	draw()
	gl.BindFramebuffer(gl.DRAW_FRAMEBUFFER, uint32(fbo))
	gl.BindFramebuffer(gl.READ_FRAMEBUFFER, uint32(read))
	gl.Viewport(viewport[0], viewport[1], viewport[2], viewport[3])
	gl.Scissor(scissor[0], scissor[1], scissor[2], scissor[3])
	if scissored {
		gl.Enable(gl.SCISSOR_TEST)
	}
	gl.ClearColor(clear[0], clear[1], clear[2], clear[3])
	gl.ClearDepth(depth)
}

func (m *ShadowMap) Project(s *Scene3D, projector M4, v []Vertex, idx []uint32, opacity float32) {
	m.ProjectMasked(s, projector, v, idx, opacity, nil)
}

// ProjectMasked applies the same alpha cutoff as the receiver's material.
func (m *ShadowMap) ProjectMasked(s *Scene3D, projector M4, v []Vertex, idx []uint32, opacity float32, material *Texture) {
	m.projectMasked(s, projector, v, idx, opacity, material, .5)
}

// World receivers use the same authored alpha test as their visible surface.
func (m *ShadowMap) ProjectWorldMasked(s *Scene3D, projector M4, v []Vertex, idx []uint32, opacity float32, material *Texture, mani bool) {
	m.projectMasked(s, projector, v, idx, opacity, material, NativeWorldAlphaRef(mani))
}

func (m *ShadowMap) projectMasked(s *Scene3D, projector M4, v []Vertex, idx []uint32, opacity float32, material *Texture, cutoff float32) {
	m.receiver.Stream(v, idx)
	gl.UseProgram(m.prog)
	uniform := func(n string) int32 { return gl.GetUniformLocation(m.prog, gl.Str(n+"\x00")) }
	vp := s.View.Mul(s.depthProjection(s.Proj))
	gl.UniformMatrix4fv(uniform("viewProj"), 1, false, &vp[0])
	gl.UniformMatrix4fv(uniform("projector"), 1, false, &projector[0])
	gl.UniformMatrix4fv(uniform("view"), 1, false, &s.View[0])
	gl.Uniform4fv(uniform("clipPlane"), 1, &s.ClipPlane[0])
	gl.Uniform2f(uniform("fogRange"), s.FogStart, s.FogEnd)
	gl.Uniform1f(uniform("opacity"), opacity)
	alpha := float32(-1)
	if material != nil {
		alpha = cutoff
		gl.BindTextureUnit(1, material.ID)
	}
	gl.Uniform1f(uniform("materialAlphaRef"), alpha)
	gl.BindTextureUnit(0, m.color)
	gl.Enable(gl.DEPTH_TEST)
	if s.ReverseDepth {
		gl.DepthFunc(gl.GEQUAL)
	} else {
		gl.DepthFunc(gl.LEQUAL)
	}
	gl.DepthMask(false)
	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
	gl.Enable(gl.POLYGON_OFFSET_FILL)
	if s.ReverseDepth {
		gl.PolygonOffset(1, 1)
	} else {
		gl.PolygonOffset(-1, -1)
	}
	m.receiver.Draw()
	gl.Disable(gl.POLYGON_OFFSET_FILL)
	gl.Disable(gl.BLEND)
	gl.DepthMask(true)
}

func (m *ShadowMap) Delete() {
	if m == nil {
		return
	}
	m.receiver.Delete()
	gl.DeleteTextures(1, &m.color)
	gl.DeleteFramebuffers(1, &m.fbo)
	gl.DeleteRenderbuffers(1, &m.depth)
	gl.DeleteProgram(m.prog)
	removeGPUBytes(m.bytes)
	m.bytes = 0
}
