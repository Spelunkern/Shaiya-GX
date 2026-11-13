package render

import (
	"fmt"
	"github.com/go-gl/gl/v4.5-core/gl"
)

const glowVS = `#version 450 core
out vec2 vUV;
void main(){
 vec2 p=vec2((gl_VertexID<<1)&2,gl_VertexID&2);
 vUV=p;gl_Position=vec4(p*2.0-1.0,0.5,1.0);
}`
const glowFS = `#version 450 core
layout(binding=0) uniform sampler2D sceneTexture;
uniform int composite;
uniform int glowLevel;
in vec2 vUV;
out vec4 frag;
void main(){
 if(composite!=0){frag=vec4(texture(sceneTexture,vUV).rgb,1);return;}
 ivec2 p=ivec2(gl_FragCoord.xy),size=textureSize(sceneTexture,0);
 const ivec2 offsets[5]=ivec2[5](ivec2(-1,0),ivec2(1,0),ivec2(0,1),ivec2(0,-1),ivec2(0,0));
 vec3 sum=vec3(0);
 for(int i=0;i<5;i++){
  ivec2 at=p+offsets[i];
  if(any(lessThan(at,ivec2(0)))||any(greaterThanEqual(at,size)))continue;
  vec3 c=texelFetch(sceneTexture,at,0).rgb*255.0;
  float weight=i==4?35.0:44.0;
  if(glowLevel==1)weight=i==4?17.0:22.0;
  // Native SRCALPHA/ONE accumulates into an eight-bit target after each tap.
  sum=clamp(roundEven(sum+c*(weight/255.0)),0.0,255.0);
 }
 frag=vec4(sum/255.0,1);
}`

// SceneGlow implements the default fixed-function GLOW_LEVEL2 branch:
// 0x54e490, 0x54f860/0x54f250 and 0x54ea70. It operates on the scene only.
type SceneGlow struct {
	program, vao           uint32
	framebuffers, textures [2]uint32
	width, height          int32
	bytes                  uint64
	composite              int32
	level                  int32
	Level                  int
}

func NewSceneGlow() (*SceneGlow, error) {
	p, err := Program(glowVS, glowFS)
	if err != nil {
		return nil, err
	}
	g := &SceneGlow{Level: 2, level: gl.GetUniformLocation(p, gl.Str("glowLevel\x00")), program: p, composite: gl.GetUniformLocation(p, gl.Str("composite\x00"))}
	gl.CreateVertexArrays(1, &g.vao)
	return g, nil
}

func (g *SceneGlow) resize(w, h int32) error {
	if g.width == w && g.height == h {
		return nil
	}
	g.releaseTargets()
	gl.CreateFramebuffers(2, &g.framebuffers[0])
	gl.CreateTextures(gl.TEXTURE_2D, 2, &g.textures[0])
	for i := range g.textures {
		id := g.textures[i]
		gl.TextureStorage2D(id, 1, gl.RGBA8, w, h)
		gl.TextureParameteri(id, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
		gl.TextureParameteri(id, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
		gl.TextureParameteri(id, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
		gl.TextureParameteri(id, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
		gl.NamedFramebufferTexture(g.framebuffers[i], gl.COLOR_ATTACHMENT0, id, 0)
		if status := gl.CheckNamedFramebufferStatus(g.framebuffers[i], gl.FRAMEBUFFER); status != gl.FRAMEBUFFER_COMPLETE {
			g.releaseTargets()
			return fmt.Errorf("scene glow target: %#x", status)
		}
	}
	g.width, g.height = w, h
	g.bytes = uint64(w) * uint64(h) * 8
	addGPUBytes(g.bytes)
	return nil
}

// Apply preserves depth, the render target and the caller's graphics state.
// Two color targets are reused; no geometry traversal or readback is needed.
func (g *SceneGlow) Apply() error {
	if g.Level == 0 {
		return nil
	}
	var viewport [4]int32
	gl.GetIntegerv(gl.VIEWPORT, &viewport[0])
	w, h := max(viewport[2]/2, 1), max(viewport[3]/2, 1)
	if err := g.resize(w, h); err != nil {
		return err
	}
	var target, program, vao, active, texture int32
	var factors [4]int32
	var equation [2]int32
	gl.GetIntegerv(gl.DRAW_FRAMEBUFFER_BINDING, &target)
	gl.GetIntegerv(gl.CURRENT_PROGRAM, &program)
	gl.GetIntegerv(gl.VERTEX_ARRAY_BINDING, &vao)
	gl.GetIntegerv(gl.ACTIVE_TEXTURE, &active)
	gl.ActiveTexture(gl.TEXTURE0)
	gl.GetIntegerv(gl.TEXTURE_BINDING_2D, &texture)
	gl.ActiveTexture(uint32(active))
	gl.GetIntegerv(gl.BLEND_SRC_RGB, &factors[0])
	gl.GetIntegerv(gl.BLEND_DST_RGB, &factors[1])
	gl.GetIntegerv(gl.BLEND_SRC_ALPHA, &factors[2])
	gl.GetIntegerv(gl.BLEND_DST_ALPHA, &factors[3])
	gl.GetIntegerv(gl.BLEND_EQUATION_RGB, &equation[0])
	gl.GetIntegerv(gl.BLEND_EQUATION_ALPHA, &equation[1])
	caps := [...]uint32{gl.DEPTH_TEST, gl.BLEND, gl.CULL_FACE, gl.SCISSOR_TEST, gl.CLIP_DISTANCE0}
	var enabled [len(caps)]bool
	for i, cap := range caps {
		enabled[i] = gl.IsEnabled(cap)
		gl.Disable(cap)
	}
	defer func() {
		gl.BindFramebuffer(gl.DRAW_FRAMEBUFFER, uint32(target))
		gl.Viewport(viewport[0], viewport[1], viewport[2], viewport[3])
		gl.UseProgram(uint32(program))
		gl.BindVertexArray(uint32(vao))
		gl.BindTextureUnit(0, uint32(texture))
		gl.BlendFuncSeparate(uint32(factors[0]), uint32(factors[1]), uint32(factors[2]), uint32(factors[3]))
		gl.BlendEquationSeparate(uint32(equation[0]), uint32(equation[1]))
		for i, cap := range caps {
			if enabled[i] {
				gl.Enable(cap)
			} else {
				gl.Disable(cap)
			}
		}
	}()
	gl.BlitNamedFramebuffer(uint32(target), g.framebuffers[0], viewport[0], viewport[1], viewport[0]+viewport[2], viewport[1]+viewport[3], 0, 0, w, h, gl.COLOR_BUFFER_BIT, gl.LINEAR)
	gl.BindFramebuffer(gl.DRAW_FRAMEBUFFER, g.framebuffers[1])
	gl.Viewport(0, 0, w, h)
	gl.UseProgram(g.program)
	gl.BindVertexArray(g.vao)
	gl.BindTextureUnit(0, g.textures[0])
	gl.Uniform1i(g.composite, 0)
	gl.Uniform1i(g.level, int32(g.Level))
	gl.DrawArrays(gl.TRIANGLES, 0, 3)
	drawCalls++
	gl.BindFramebuffer(gl.DRAW_FRAMEBUFFER, uint32(target))
	gl.Viewport(viewport[0], viewport[1], viewport[2], viewport[3])
	gl.BindTextureUnit(0, g.textures[1])
	gl.Uniform1i(g.composite, 1)
	gl.Enable(gl.BLEND)
	gl.BlendEquation(gl.FUNC_ADD)
	gl.BlendFunc(gl.SRC_COLOR, gl.ONE)
	gl.DrawArrays(gl.TRIANGLES, 0, 3)
	drawCalls++
	return nil
}

func (g *SceneGlow) releaseTargets() {
	gl.DeleteFramebuffers(2, &g.framebuffers[0])
	gl.DeleteTextures(2, &g.textures[0])
	g.framebuffers, g.textures = [2]uint32{}, [2]uint32{}
	removeGPUBytes(g.bytes)
	g.bytes = 0
	g.width, g.height = 0, 0
}
func (g *SceneGlow) Delete() {
	if g == nil {
		return
	}
	g.releaseTargets()
	gl.DeleteProgram(g.program)
	gl.DeleteVertexArrays(1, &g.vao)
	g.program, g.vao = 0, 0
}
