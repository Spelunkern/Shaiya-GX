package render

import (
	"log"
	"unsafe"

	"github.com/go-gl/gl/v4.5-core/gl"
)

const terrainOverlayLimit = 7
const terrainLayerBlocksUnit = 15

const terrainLayerBlocksGLSL = `
layout(binding=15) uniform usampler2D terrainLayerBlocks;
bool terrainLayerEnabled(vec2 worldXZ, uint layerBit) {
 if (layerBit == 0u) return true;
 ivec2 block = clamp(ivec2(worldXZ / 32.0), ivec2(0), textureSize(terrainLayerBlocks, 0) - 1);
 return (texelFetch(terrainLayerBlocks, block, 0).r & layerBit) != 0u;
}
`

type TerrainSplat struct {
	Mask, Tile *Texture
	TileSize   float32
	LayerBit   uint32 // logical WLD layer slot, even when preceding masks are absent
}

// UploadTerrainLayerBlocks stores one unfiltered byte of layer bits per block.
func UploadTerrainLayerBlocks(side int, blocks []byte) *Texture {
	t := &Texture{W: side, H: side, bytes: uint64(len(blocks))}
	gl.CreateTextures(gl.TEXTURE_2D, 1, &t.ID)
	gl.TextureStorage2D(t.ID, 1, gl.R8UI, int32(side), int32(side))
	var alignment int32
	gl.GetIntegerv(gl.UNPACK_ALIGNMENT, &alignment)
	gl.PixelStorei(gl.UNPACK_ALIGNMENT, 1)
	gl.TextureSubImage2D(t.ID, 0, 0, 0, int32(side), int32(side), gl.RED_INTEGER, gl.UNSIGNED_BYTE, unsafe.Pointer(&blocks[0]))
	gl.PixelStorei(gl.UNPACK_ALIGNMENT, alignment)
	gl.TextureParameteri(t.ID, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	gl.TextureParameteri(t.ID, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	addGPUBytes(t.bytes)
	return t
}

type terrainOverlay struct {
	prog     uint32
	tried    bool
	commands []terrainCommand
	indirect effectUpload
}

// DrawTerrainSplats preserves the opaque base pass and each overlay's native
// blend and depth operations. Compatible overlays share one GL submission.
func (s *Scene3D) DrawTerrainSplats(m *Mesh, layers []TerrainSplat, light, layerBlocks *Texture, ranges int) {
	if len(layers) == 0 || m == nil || m.Count == 0 {
		return
	}
	base := layers[0]
	if ranges == 0 {
		s.DrawTerrainLayer(m, base, light, nil, true)
	} else {
		s.drawTerrainRanges(m, base, light, nil, true, ranges)
	}
	layers = layers[1:]
	if !s.drawTerrainOverlays(m, layers, light, layerBlocks, ranges) {
		for _, layer := range layers {
			if ranges == 0 {
				s.DrawTerrainLayer(m, layer, light, layerBlocks, false)
			} else {
				s.drawTerrainRanges(m, layer, light, layerBlocks, false, ranges)
			}
		}
	}
	s.drawTerrainHalo(m, ranges)
}

func (s *Scene3D) drawTerrainOverlays(m *Mesh, layers []TerrainSplat, light, layerBlocks *Texture, ranges int) bool {
	if len(layers) < 2 || len(layers) > terrainOverlayLimit || s.Shadow || !drawParametersSupported {
		return false
	}
	t := &s.terrainOverlay
	if !t.tried {
		t.tried = true
		p, err := Program("terrain overlay", terrainOverlayVS, terrainOverlayFS)
		if err != nil {
			log.Printf("terrain overlay shader: %v", err)
		} else {
			t.prog = p
		}
	}
	if t.prog == 0 {
		return false
	}

	t.commands = t.commands[:0]
	n := max(ranges, 1)
	for range layers {
		if ranges == 0 {
			t.commands = append(t.commands, terrainCommand{Count: uint32(m.Count), Instances: 1})
		} else {
			t.commands = append(t.commands, s.terrainCommands[:ranges]...)
		}
	}
	t.indirect.upload(len(t.commands)*indexedDrawStride, unsafe.Pointer(&t.commands[0]))
	gl.UseProgram(t.prog)
	mv := s.modelView(Identity())
	mvp := mv.Mul(s.projection)
	gl.UniformMatrix4fv(0, 1, false, &mvp[0])
	gl.UniformMatrix4fv(4, 1, false, &mv[0])
	gl.Uniform4fv(8, 1, &s.ClipPlane[0])
	gl.Uniform3fv(9, 1, &s.FogColor[0])
	gl.Uniform2f(10, s.FogStart, s.FogEnd)
	var sizes [terrainOverlayLimit]float32
	var bits [terrainOverlayLimit]uint32
	if layerBlocks != nil {
		gl.BindTextureUnit(terrainLayerBlocksUnit, layerBlocks.ID)
	}
	for i, layer := range layers {
		sizes[i] = layer.TileSize
		if layerBlocks != nil {
			bits[i] = layer.LayerBit
		}
		gl.BindTextureUnit(uint32(i), layer.Mask.ID)
		gl.BindTextureUnit(uint32(i+terrainOverlayLimit), layer.Tile.ID)
	}
	lit := int32(0)
	if light != nil {
		gl.BindTextureUnit(2*terrainOverlayLimit, light.ID)
		lit = 1
	}
	gl.Uniform1i(12, lit)
	gl.Uniform1i(13, int32(n))
	gl.Uniform1fv(14, int32(len(layers)), &sizes[0])
	gl.Uniform1uiv(21, int32(len(layers)), &bits[0])
	s.blend(true)
	gl.BindVertexArray(m.vao)
	gl.BindBuffer(gl.DRAW_INDIRECT_BUFFER, t.indirect.id)
	gl.MultiDrawElementsIndirect(gl.TRIANGLES, m.indexType, nil, int32(len(t.commands)), 0)
	gl.BindBuffer(gl.DRAW_INDIRECT_BUFFER, 0)
	drawCalls++
	gl.UseProgram(s.prog)
	return true
}

const terrainOverlayVS = `#version 450 core
#extension GL_ARB_shader_draw_parameters : require
layout(location=0) in vec3 pos;
layout(location=2) in vec2 uv;
layout(location=3) in vec2 uv2;
layout(location=0) uniform mat4 worldViewProj;
layout(location=4) uniform mat4 worldView;
layout(location=8) uniform vec4 clipPlane;
layout(location=13) uniform int ranges;
out vec2 vUV,vUV2;
out float vDepth;
flat out int layer;
void main() {
 vec4 p=vec4(pos,1);
 gl_Position=worldViewProj*p;
 gl_ClipDistance[0]=dot(p,clipPlane);
 vDepth=(worldView*p).z;
 vUV=uv; vUV2=uv2;
 layer=gl_DrawIDARB/ranges;
}`

const terrainOverlayFS = `#version 450 core
layout(binding=0) uniform sampler2D masks[7];
layout(binding=7) uniform sampler2D tiles[7];
layout(binding=14) uniform sampler2D lightmap;
layout(location=9) uniform vec3 fogColor;
layout(location=10) uniform vec2 fogRange;
layout(location=12) uniform int lit;
layout(location=14) uniform float tileSizes[7];
layout(location=21) uniform uint layerBits[7];
in vec2 vUV,vUV2;
in float vDepth;
flat in int layer;
out vec4 frag;
` + terrainLayerBlocksGLSL + `
void main() {
 vec2 tileUV=vUV2/tileSizes[layer];
 vec2 ddx=dFdx(tileUV);
 vec2 ddy=dFdy(tileUV);
 if (!terrainLayerEnabled(vUV2, layerBits[layer])) discard;
 float alpha=texture(masks[layer],vUV).a;
 if (alpha <= 7.5/255.0) discard;
 vec4 c=vec4(textureGrad(tiles[layer],tileUV,ddx,ddy).rgb,alpha);
 if (lit != 0) c.rgb *= texture(lightmap,vUV).rgb;
 if (fogRange.y > fogRange.x) {
  float f=clamp((fogRange.y-vDepth)/(fogRange.y-fogRange.x),0.0,1.0);
  c.rgb=mix(fogColor,c.rgb,f);
 }
 frag=c;
}`
