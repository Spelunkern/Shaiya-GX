package render

import (
	"fmt"

	"github.com/go-gl/gl/v4.5-core/gl"
)

// MaxSamples is the driver limit for multisampled color/depth renderbuffers.
func MaxSamples() int {
	var n int32
	gl.GetIntegerv(gl.MAX_SAMPLES, &n)
	return int(n)
}

// MSAATarget is an offscreen multisampled color + float depth target for the
// 3D scene. The UI is drawn after Resolve, so text and HUD stay sharp.
type MSAATarget struct {
	fbo, color, depth uint32
	width, height     int32
	samples           int
	bytes             uint64
	parent            int32
	viewport, scissor [4]int32
	scissored         bool
}

func NewMSAATarget(width, height, samples int) (*MSAATarget, error) {
	if width <= 0 || height <= 0 || samples < 2 {
		return nil, fmt.Errorf("msaa target: invalid %dx%d x%d", width, height, samples)
	}
	t := &MSAATarget{width: int32(width), height: int32(height), samples: samples}
	gl.CreateFramebuffers(1, &t.fbo)
	gl.CreateRenderbuffers(1, &t.color)
	gl.CreateRenderbuffers(1, &t.depth)
	// Formats match the frame target so the resolve blit is legal for depth too.
	gl.NamedRenderbufferStorageMultisample(t.color, int32(samples), gl.RGBA8, t.width, t.height)
	gl.NamedRenderbufferStorageMultisample(t.depth, int32(samples), gl.DEPTH_COMPONENT32F, t.width, t.height)
	gl.NamedFramebufferRenderbuffer(t.fbo, gl.COLOR_ATTACHMENT0, gl.RENDERBUFFER, t.color)
	gl.NamedFramebufferRenderbuffer(t.fbo, gl.DEPTH_ATTACHMENT, gl.RENDERBUFFER, t.depth)
	if status := gl.CheckNamedFramebufferStatus(t.fbo, gl.FRAMEBUFFER); status != gl.FRAMEBUFFER_COMPLETE {
		t.Delete()
		return nil, fmt.Errorf("msaa target: framebuffer %#x", status)
	}
	t.bytes = uint64(width) * uint64(height) * 8 * uint64(samples)
	addGPUBytes(t.bytes)
	return t, nil
}

func (t *MSAATarget) Matches(width, height, samples int) bool {
	return t != nil && int(t.width) == width && int(t.height) == height && t.samples == samples
}

// Begin redirects drawing into the multisampled target, cleared like the
// frame target (opaque black, current depth clear value).
func (t *MSAATarget) Begin() {
	gl.GetIntegerv(gl.DRAW_FRAMEBUFFER_BINDING, &t.parent)
	gl.GetIntegerv(gl.VIEWPORT, &t.viewport[0])
	gl.GetIntegerv(gl.SCISSOR_BOX, &t.scissor[0])
	t.scissored = gl.IsEnabled(gl.SCISSOR_TEST)
	var mask [4]bool
	var depthMask bool
	gl.GetBooleanv(gl.COLOR_WRITEMASK, &mask[0])
	gl.GetBooleanv(gl.DEPTH_WRITEMASK, &depthMask)
	gl.BindFramebuffer(gl.FRAMEBUFFER, t.fbo)
	gl.Viewport(0, 0, t.width, t.height)
	gl.Disable(gl.SCISSOR_TEST)
	gl.ColorMask(true, true, true, true)
	gl.DepthMask(true)
	black := [4]float32{0, 0, 0, 1}
	gl.ClearBufferfv(gl.COLOR, 0, &black[0])
	gl.Clear(gl.DEPTH_BUFFER_BIT)
	gl.ColorMask(mask[0], mask[1], mask[2], mask[3])
	gl.DepthMask(depthMask)
	if t.scissored {
		gl.Enable(gl.SCISSOR_TEST)
	}
}

// Resolve averages the samples into the target that was bound at Begin
// (color and depth, so later name/water depth reads still work) and binds it.
func (t *MSAATarget) Resolve() {
	gl.Disable(gl.SCISSOR_TEST)
	gl.BlitNamedFramebuffer(t.fbo, uint32(t.parent), 0, 0, t.width, t.height, 0, 0, t.width, t.height, gl.COLOR_BUFFER_BIT, gl.NEAREST)
	gl.BlitNamedFramebuffer(t.fbo, uint32(t.parent), 0, 0, t.width, t.height, 0, 0, t.width, t.height, gl.DEPTH_BUFFER_BIT, gl.NEAREST)
	gl.BindFramebuffer(gl.FRAMEBUFFER, uint32(t.parent))
	gl.Viewport(t.viewport[0], t.viewport[1], t.viewport[2], t.viewport[3])
	gl.Scissor(t.scissor[0], t.scissor[1], t.scissor[2], t.scissor[3])
	if t.scissored {
		gl.Enable(gl.SCISSOR_TEST)
	}
}

func (t *MSAATarget) Delete() {
	if t == nil {
		return
	}
	gl.DeleteFramebuffers(1, &t.fbo)
	gl.DeleteRenderbuffers(1, &t.color)
	gl.DeleteRenderbuffers(1, &t.depth)
	removeGPUBytes(t.bytes)
	*t = MSAATarget{}
}

// framebufferSamples reports the sample count of a bound framebuffer (0 for
// single-sampled), used to pick blit instead of copy for screen captures.
func framebufferSamples(fbo int32) int32 {
	var samples int32
	gl.GetNamedFramebufferParameteriv(uint32(fbo), gl.SAMPLES, &samples)
	return samples
}

const fxaaFS = `#version 450 core
layout(binding=0) uniform sampler2D sceneTexture;
in vec2 vUV;
out vec4 frag;
float luma(vec3 c){ return dot(c, vec3(0.299, 0.587, 0.114)); }
// FXAA 3.11 console-quality variant: one pass, five taps plus two/four along the edge.
void main(){
 vec2 px = 1.0 / vec2(textureSize(sceneTexture, 0));
 vec3 rgbM = texture(sceneTexture, vUV).rgb;
 vec3 rgbNW = texture(sceneTexture, vUV + vec2(-1.0,-1.0)*px).rgb;
 vec3 rgbNE = texture(sceneTexture, vUV + vec2( 1.0,-1.0)*px).rgb;
 vec3 rgbSW = texture(sceneTexture, vUV + vec2(-1.0, 1.0)*px).rgb;
 vec3 rgbSE = texture(sceneTexture, vUV + vec2( 1.0, 1.0)*px).rgb;
 float lM = luma(rgbM), lNW = luma(rgbNW), lNE = luma(rgbNE), lSW = luma(rgbSW), lSE = luma(rgbSE);
 float lMin = min(lM, min(min(lNW, lNE), min(lSW, lSE)));
 float lMax = max(lM, max(max(lNW, lNE), max(lSW, lSE)));
 if (lMax - lMin < max(0.0312, lMax * 0.125)) { frag = vec4(rgbM, 1.0); return; }
 vec2 dir = vec2(-((lNW + lNE) - (lSW + lSE)), (lNW + lSW) - (lNE + lSE));
 float reduce = max((lNW + lNE + lSW + lSE) * 0.03125, 1.0/128.0);
 float rcp = 1.0 / (min(abs(dir.x), abs(dir.y)) + reduce);
 dir = clamp(dir * rcp, vec2(-8.0), vec2(8.0)) * px;
 vec3 a = 0.5 * (texture(sceneTexture, vUV + dir * (1.0/3.0 - 0.5)).rgb + texture(sceneTexture, vUV + dir * (2.0/3.0 - 0.5)).rgb);
 vec3 b = a * 0.5 + 0.25 * (texture(sceneTexture, vUV + dir * -0.5).rgb + texture(sceneTexture, vUV + dir * 0.5).rgb);
 float lB = luma(b);
 frag = vec4((lB < lMin || lB > lMax) ? a : b, 1.0);
}`

// FXAA smooths edges of the already drawn 3D scene in one full-screen pass,
// including alpha-tested foliage that MSAA does not cover. Depth is untouched.
type FXAA struct {
	program, vao uint32
	fbo, texture uint32
	width        int32
	height       int32
	bytes        uint64
}

func NewFXAA() (*FXAA, error) {
	p, err := Program(glowVS, fxaaFS)
	if err != nil {
		return nil, err
	}
	f := &FXAA{program: p}
	gl.CreateVertexArrays(1, &f.vao)
	return f, nil
}

func (f *FXAA) resize(w, h int32) error {
	if f.width == w && f.height == h {
		return nil
	}
	f.release()
	gl.CreateFramebuffers(1, &f.fbo)
	gl.CreateTextures(gl.TEXTURE_2D, 1, &f.texture)
	gl.TextureStorage2D(f.texture, 1, gl.RGBA8, w, h)
	gl.TextureParameteri(f.texture, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TextureParameteri(f.texture, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TextureParameteri(f.texture, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TextureParameteri(f.texture, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	gl.NamedFramebufferTexture(f.fbo, gl.COLOR_ATTACHMENT0, f.texture, 0)
	if status := gl.CheckNamedFramebufferStatus(f.fbo, gl.FRAMEBUFFER); status != gl.FRAMEBUFFER_COMPLETE {
		f.release()
		return fmt.Errorf("fxaa target: %#x", status)
	}
	f.width, f.height = w, h
	f.bytes = uint64(w) * uint64(h) * 4
	addGPUBytes(f.bytes)
	return nil
}

// Apply filters the current viewport of the bound draw framebuffer in place.
func (f *FXAA) Apply() error {
	var viewport [4]int32
	gl.GetIntegerv(gl.VIEWPORT, &viewport[0])
	w, h := max(viewport[2], 1), max(viewport[3], 1)
	if err := f.resize(w, h); err != nil {
		return err
	}
	var target, program, vao, active, texture int32
	gl.GetIntegerv(gl.DRAW_FRAMEBUFFER_BINDING, &target)
	gl.GetIntegerv(gl.CURRENT_PROGRAM, &program)
	gl.GetIntegerv(gl.VERTEX_ARRAY_BINDING, &vao)
	gl.GetIntegerv(gl.ACTIVE_TEXTURE, &active)
	gl.ActiveTexture(gl.TEXTURE0)
	gl.GetIntegerv(gl.TEXTURE_BINDING_2D, &texture)
	gl.ActiveTexture(uint32(active))
	caps := [...]uint32{gl.DEPTH_TEST, gl.BLEND, gl.CULL_FACE, gl.SCISSOR_TEST, gl.CLIP_DISTANCE0}
	var enabled [len(caps)]bool
	for i, c := range caps {
		enabled[i] = gl.IsEnabled(c)
		gl.Disable(c)
	}
	gl.BlitNamedFramebuffer(uint32(target), f.fbo, viewport[0], viewport[1], viewport[0]+w, viewport[1]+h, 0, 0, w, h, gl.COLOR_BUFFER_BIT, gl.NEAREST)
	gl.UseProgram(f.program)
	gl.BindVertexArray(f.vao)
	gl.BindTextureUnit(0, f.texture)
	gl.DrawArrays(gl.TRIANGLES, 0, 3)
	drawCalls++
	gl.BindTextureUnit(0, uint32(texture))
	gl.UseProgram(uint32(program))
	gl.BindVertexArray(uint32(vao))
	for i, c := range caps {
		if enabled[i] {
			gl.Enable(c)
		}
	}
	return nil
}

func (f *FXAA) release() {
	gl.DeleteFramebuffers(1, &f.fbo)
	gl.DeleteTextures(1, &f.texture)
	f.fbo, f.texture = 0, 0
	removeGPUBytes(f.bytes)
	f.bytes, f.width, f.height = 0, 0, 0
}

func (f *FXAA) Delete() {
	if f == nil {
		return
	}
	f.release()
	gl.DeleteProgram(f.program)
	gl.DeleteVertexArrays(1, &f.vao)
	f.program, f.vao = 0, 0
}
