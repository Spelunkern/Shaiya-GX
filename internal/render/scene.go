package render

import (
	"github.com/go-gl/gl/v4.5-core/gl"
)

// Programs of the 3D passes. Colours stay in the texture's encoded space
// (no sRGB conversion), like the D3D9 fixed function.
const sceneVS = `#version 450 core
layout(location=0) in vec3 pos;
layout(location=1) in vec3 nrm;
layout(location=2) in vec2 uv;
layout(location=3) in vec2 uv2;
uniform mat4 world;
uniform mat4 viewProj;
uniform mat4 view;
uniform vec3 lightDir;   // raw direction the light travels
uniform float ambient;
uniform int lit;
out vec2 vUV;
out vec2 vUV2;
out float vDiffuse;
out float vDepth;
void main() {
	vec4 w = world * vec4(pos, 1.0); // == D3D v*M (see M4)
	gl_Position = viewProj * w;
	vDepth = (view * w).z;
	vUV = uv; vUV2 = uv2;
	vec3 n = normalize((world * vec4(nrm, 0.0)).xyz);
	vDiffuse = lit == 1 ? clamp(ambient + max(0.0, dot(n, -normalize(lightDir))), 0.0, 1.0) : 1.0;
}`

// mode 0: texture * diffuse (models)
// mode 1: terrain layer: rgb = tile (uv2 wrap), alpha = mask (uv clamp)
// mode 2: lightmap multiply pass (uv)
const sceneFS = `#version 450 core
layout(binding=0) uniform sampler2D tex0;
layout(binding=1) uniform sampler2D tex1;
uniform int mode;
uniform float alphaRef;
uniform vec3 fogColor;
uniform vec2 fogRange; // start, end; end <= start disables
uniform float tileSize; // terrain layer tiling (world units per repeat)
uniform vec4 tint;      // mode 3 colour (sky, clouds)
uniform vec2 uvOffset;
in vec2 vUV;
in vec2 vUV2;
in float vDiffuse;
in float vDepth;
out vec4 frag;
void main() {
	vec4 c;
	if (mode == 1) {
		c = vec4(texture(tex1, vUV2 / tileSize).rgb, texture(tex0, vUV).a);
	} else if (mode == 4) {
		c = texture(tex0, vUV);
		c.rgb *= texture(tex1, vUV2).rgb;
	} else if (mode == 3) {
		frag = texture(tex0, vUV + uvOffset) * tint;
		return;
	} else if (mode == 2) {
		frag = vec4(texture(tex0, vUV).rgb, 1.0);
		return;
	} else {
		c = texture(tex0, vUV);
		c.rgb *= vDiffuse;
	}
	if (c.a <= alphaRef) discard;
	if (fogRange.y > fogRange.x) {
		float f = clamp((fogRange.y - vDepth) / (fogRange.y - fogRange.x), 0.0, 1.0);
		c.rgb = mix(fogColor, c.rgb, f);
	}
	frag = c;
}`

// Scene3D holds the shared 3D program and per-frame camera state.
type Scene3D struct {
	prog                                                      uint32
	uWorld, uViewProj, uView, uLight, uAmb, uLit, uMode, uRef int32
	uFogColor, uFogRange, uTile, uTint, uUVOff                int32
	View, Proj, SkyProj                                       M4
	LightDir                                                  V3
	Ambient                                                   float32
	FogColor                                                  V3
	FogStart, FogEnd                                          float32
}

func NewScene3D() (*Scene3D, error) {
	p, err := Program(sceneVS, sceneFS)
	if err != nil {
		return nil, err
	}
	u := func(n string) int32 { return gl.GetUniformLocation(p, gl.Str(n+"\x00")) }
	return &Scene3D{prog: p, uWorld: u("world"), uViewProj: u("viewProj"), uView: u("view"), uLight: u("lightDir"),
		uAmb: u("ambient"), uLit: u("lit"), uMode: u("mode"), uRef: u("alphaRef"), uFogColor: u("fogColor"), uFogRange: u("fogRange"), uTile: u("tileSize"), uTint: u("tint"), uUVOff: u("uvOffset"),
		LightDir: V3{-1, -1, -1}, Ambient: 128.0 / 255}, nil
}

// Begin sets the frame state: D3D depth range, no culling (EP5 draws
// two-sided), depth test on.
func (s *Scene3D) Begin() {
	gl.ClipControl(gl.LOWER_LEFT, gl.ZERO_TO_ONE)
	gl.Enable(gl.DEPTH_TEST)
	gl.DepthFunc(gl.LEQUAL)
	gl.DepthMask(true)
	gl.Disable(gl.CULL_FACE)
	gl.UseProgram(s.prog)
	vp := s.View.Mul(s.Proj)
	gl.UniformMatrix4fv(s.uViewProj, 1, false, &vp[0])
	gl.UniformMatrix4fv(s.uView, 1, false, &s.View[0])
	gl.Uniform3f(s.uLight, s.LightDir[0], s.LightDir[1], s.LightDir[2])
	gl.Uniform1f(s.uAmb, s.Ambient)
	gl.Uniform3f(s.uFogColor, s.FogColor[0], s.FogColor[1], s.FogColor[2])
	gl.Uniform2f(s.uFogRange, s.FogStart, s.FogEnd)
}

// End restores the 2D state.
func (s *Scene3D) End() {
	gl.Disable(gl.DEPTH_TEST)
	gl.DepthMask(true)
	gl.Disable(gl.BLEND)
}

// DrawModel draws a mesh with texture * clamped Gouraud diffuse.
func (s *Scene3D) DrawModel(m *Mesh, t *Texture, world M4, alphaRef float32, blend bool) {
	gl.UniformMatrix4fv(s.uWorld, 1, false, &world[0])
	gl.Uniform1i(s.uMode, 0)
	gl.Uniform1i(s.uLit, 1)
	gl.Uniform1f(s.uRef, alphaRef)
	s.blend(blend)
	if t != nil {
		gl.BindTextureUnit(0, t.ID)
	}
	m.Draw()
}

// DrawTerrainLayer draws one layer pass: first opaque, later alpha-blended.
func (s *Scene3D) DrawTerrainLayer(m *Mesh, mask, tile *Texture, tileSize float32, first bool) {
	gl.Uniform1f(s.uTile, tileSize)
	id := Identity()
	gl.UniformMatrix4fv(s.uWorld, 1, false, &id[0])
	gl.Uniform1i(s.uMode, 1)
	gl.Uniform1i(s.uLit, 0)
	if first {
		gl.Uniform1f(s.uRef, -1)
		s.blend(false)
	} else {
		gl.Uniform1f(s.uRef, 7.5/255)
		s.blend(true)
	}
	gl.BindTextureUnit(0, mask.ID)
	gl.BindTextureUnit(1, tile.ID)
	m.Draw()
}

// DrawLightmap multiplies the frame by the terrain lightmap.
func (s *Scene3D) DrawLightmap(m *Mesh, lm *Texture) {
	id := Identity()
	gl.UniformMatrix4fv(s.uWorld, 1, false, &id[0])
	gl.Uniform1i(s.uMode, 2)
	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.DST_COLOR, gl.ZERO)
	gl.BindTextureUnit(0, lm.ID)
	m.Draw()
	gl.Disable(gl.BLEND)
}

func (s *Scene3D) blend(on bool) {
	if on {
		gl.Enable(gl.BLEND)
		gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
	} else {
		gl.Disable(gl.BLEND)
	}
}

// SetWrap sets a texture's addressing (terrain tiles repeat).
func (t *Texture) SetWrap(repeat bool) {
	mode := int32(gl.CLAMP_TO_EDGE)
	if repeat {
		mode = gl.REPEAT
	}
	gl.TextureParameteri(t.ID, gl.TEXTURE_WRAP_S, mode)
	gl.TextureParameteri(t.ID, gl.TEXTURE_WRAP_T, mode)
}

// BeginSky switches to the sky projection (near 10, far 100000) without
// depth; EndSky restores the scene projection and depth.
func (s *Scene3D) BeginSky() {
	vp := s.View.Mul(s.SkyProj)
	gl.UniformMatrix4fv(s.uViewProj, 1, false, &vp[0])
	gl.Disable(gl.DEPTH_TEST)
	gl.DepthMask(false)
}

func (s *Scene3D) EndSky() {
	vp := s.View.Mul(s.Proj)
	gl.UniformMatrix4fv(s.uViewProj, 1, false, &vp[0])
	gl.Enable(gl.DEPTH_TEST)
	gl.DepthMask(true)
	gl.Disable(gl.BLEND)
}

// DrawUnlit draws texture * (1,1,1,alpha) with alpha blending.
func (s *Scene3D) DrawUnlit(m *Mesh, t *Texture, world M4, alpha float32, uvOff [2]float32) {
	gl.UniformMatrix4fv(s.uWorld, 1, false, &world[0])
	gl.Uniform1i(s.uMode, 3)
	gl.Uniform4f(s.uTint, 1, 1, 1, alpha)
	gl.Uniform2f(s.uUVOff, uvOff[0], uvOff[1])
	s.blend(true)
	gl.BindTextureUnit(0, t.ID)
	m.Draw()
}

// DrawLightmapped draws DG geometry: texture * lightmap (clamp), unlit.
func (s *Scene3D) DrawLightmapped(m *Mesh, t, lm *Texture) {
	id := Identity()
	gl.UniformMatrix4fv(s.uWorld, 1, false, &id[0])
	gl.Uniform1i(s.uMode, 4)
	gl.Uniform1f(s.uRef, 0.5)
	s.blend(false)
	if t != nil {
		gl.BindTextureUnit(0, t.ID)
	}
	gl.BindTextureUnit(1, lm.ID)
	m.Draw()
}

// DrawTinted draws texture*tint unlit. blend: alpha blending; additive:
// SRCALPHA/ONE. Blended draws do not write depth.
func (s *Scene3D) DrawTinted(m *Mesh, t *Texture, world M4, tint [4]float32, blend, additive bool) {
	if t == nil {
		return
	}
	gl.UniformMatrix4fv(s.uWorld, 1, false, &world[0])
	gl.Uniform1i(s.uMode, 3)
	gl.Uniform4f(s.uTint, tint[0], tint[1], tint[2], tint[3])
	gl.Uniform2f(s.uUVOff, 0, 0)
	if blend {
		gl.Enable(gl.BLEND)
		if additive {
			gl.BlendFunc(gl.SRC_ALPHA, gl.ONE)
		} else {
			gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
		}
		gl.DepthMask(false)
	} else {
		gl.Disable(gl.BLEND)
	}
	gl.BindTextureUnit(0, t.ID)
	m.Draw()
	gl.DepthMask(true)
	gl.Disable(gl.BLEND)
}
