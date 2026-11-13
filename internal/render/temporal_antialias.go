package render

import (
	"fmt"
	"github.com/go-gl/gl/v4.5-core/gl"
)

// TemporalJitter is a bounded, repeating subpixel Halton sequence in NDC.
func TemporalJitter(frame uint64, width, height int) [2]float32 {
	if width <= 0 || height <= 0 {
		return [2]float32{}
	}
	halton := func(index, base int) float32 {
		var value float32
		scale := float32(1)
		for index > 0 {
			scale /= float32(base)
			value += scale * float32(index%base)
			index /= base
		}
		return value
	}
	i := int(frame%8) + 1
	return [2]float32{2 * (halton(i, 2) - .5) / float32(width), 2 * (halton(i, 3) - .5) / float32(height)}
}

const temporalFS = `#version 450 core
layout(binding=0) uniform sampler2D currentColor;
layout(binding=1) uniform sampler2D currentDepth;
layout(binding=2) uniform sampler2D previousColor;
layout(binding=3) uniform sampler2D previousDepth;
uniform mat4 reprojection;
uniform vec2 jitter;
uniform int historyValid;
uniform float depthAtInfinity;
in vec2 vUV;
out vec4 frag;
float luma(vec3 c){return dot(c,vec3(.299,.587,.114));}
void main(){
 vec2 px=1.0/vec2(textureSize(currentColor,0));
 vec2 uv=clamp(vUV+jitter*.5,px*.5,vec2(1)-px*.5);
 float depth=texture(currentDepth,uv).r;
 vec3 color=texture(currentColor,uv).rgb;
 gl_FragDepth=depth;
 vec4 clip=reprojection*vec4(uv*2.0-1.0,depth,1);
 float weight=0;
 vec3 history=color;
 if(historyValid!=0 && clip.w>0){
  vec3 previous=clip.xyz/clip.w;
  vec2 huv=previous.xy*.5+.5;
  bool inside=all(greaterThanEqual(huv,px*.5))&&all(lessThanEqual(huv,vec2(1)-px*.5))&&previous.z>=0&&previous.z<=1;
  if(inside){
   float oldDepth=texture(previousDepth,huv).r;
   float tolerance=max(2e-6,abs(previous.z-depthAtInfinity)*.012);
   if(abs(oldDepth-previous.z)<tolerance){
    history=texture(previousColor,huv).rgb;
    vec3 low=color,high=color,mean=vec3(0),square=vec3(0);
    for(int y=-1;y<=1;y++)for(int x=-1;x<=1;x++){
     vec3 c=texture(currentColor,uv+vec2(x,y)*px).rgb;
     low=min(low,c);high=max(high,c);mean+=c;square+=c*c;
    }
    mean/=9.;vec3 deviation=sqrt(max(square/9.-mean*mean,vec3(0)));
    low=max(low,mean-deviation*1.25);high=min(high,mean+deviation*1.25);
    float change=abs(luma(history)-luma(color));
    history=clamp(history,low,high);
    float motion=length((huv-vUV)/px);
    weight=mix(.82,.3,clamp(motion/12.,0.,1.));
    weight*=1.-smoothstep(.06,.22,change);
   }
  }
 }
 frag=vec4(mix(color,history,weight),1);
}`

type temporalSurface struct{ fbo, color, depth uint32 }

// TemporalAA accumulates only scene colour, before glow/UI. History is stable
// (unjittered) colour + depth, with camera reprojection, disocclusion rejection
// and neighbourhood clamping. It does not provide skinned-object motion vectors.
type TemporalAA struct {
	program, vao                                      uint32
	surfaces                                          [3]temporalSurface // current capture and two alternating histories
	width, height                                     int32
	bytes                                             uint64
	valid                                             bool
	index                                             int
	sequence                                          uint64
	previousView, previousProjection                  M4
	projectionLocation, jitterLocation, validLocation int32
	infinityLocation                                  int32
}

func NewTemporalAA() (*TemporalAA, error) {
	p, e := Program(glowVS, temporalFS)
	if e != nil {
		return nil, e
	}
	t := &TemporalAA{program: p, projectionLocation: gl.GetUniformLocation(p, gl.Str("reprojection\x00")), jitterLocation: gl.GetUniformLocation(p, gl.Str("jitter\x00")), validLocation: gl.GetUniformLocation(p, gl.Str("historyValid\x00"))}
	t.infinityLocation = gl.GetUniformLocation(p, gl.Str("depthAtInfinity\x00"))
	gl.CreateVertexArrays(1, &t.vao)
	return t, nil
}
func (t *TemporalAA) Reset() {
	if t != nil {
		t.valid = false
		t.sequence = 0
	}
}
func (t *TemporalAA) NextJitter(w, h int) [2]float32 {
	j := TemporalJitter(t.sequence, w, h)
	t.sequence++
	return j
}
func (t *TemporalAA) resize(w, h int32) error {
	if t.width == w && t.height == h {
		return nil
	}
	t.release()
	for i := range t.surfaces {
		p := &t.surfaces[i]
		gl.CreateFramebuffers(1, &p.fbo)
		for slot, destination := range []*uint32{&p.color, &p.depth} {
			gl.CreateTextures(gl.TEXTURE_2D, 1, destination)
			format, attachment, filter := uint32(gl.RGBA8), uint32(gl.COLOR_ATTACHMENT0), int32(gl.LINEAR)
			if slot == 1 {
				format, attachment, filter = gl.DEPTH_COMPONENT32F, gl.DEPTH_ATTACHMENT, gl.NEAREST
			}
			gl.TextureStorage2D(*destination, 1, format, w, h)
			gl.TextureParameteri(*destination, gl.TEXTURE_MIN_FILTER, filter)
			gl.TextureParameteri(*destination, gl.TEXTURE_MAG_FILTER, filter)
			gl.TextureParameteri(*destination, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
			gl.TextureParameteri(*destination, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
			gl.NamedFramebufferTexture(p.fbo, attachment, *destination, 0)
		}
		if status := gl.CheckNamedFramebufferStatus(p.fbo, gl.FRAMEBUFFER); status != gl.FRAMEBUFFER_COMPLETE {
			t.release()
			return fmt.Errorf("temporal framebuffer %#x", status)
		}
	}
	t.width, t.height = w, h
	t.bytes = uint64(w) * uint64(h) * 24
	addGPUBytes(t.bytes)
	return nil
}
func temporalHistoryCompatible(view, projection, previousView, previousProjection M4) bool {
	if projection != previousProjection {
		return false
	}
	a, b := view.Inverse(), previousView.Inverse()
	delta := V3{a[12] - b[12], a[13] - b[13], a[14] - b[14]}
	forward := V3{view[2], view[6], view[10]}
	old := V3{previousView[2], previousView[6], previousView[10]}
	return delta.Len() < 10 && forward.Dot(old) > .5
}

// Apply resolves jittered input into stable history and copies both resolved
// colour and depth back. Separate read/draw bindings and GL state are preserved.
func (t *TemporalAA) Apply(view, projection M4, jitter [2]float32) error {
	if projection[0] == 0 || projection[5] == 0 {
		t.Reset()
		return nil
	}
	var viewport [4]int32
	gl.GetIntegerv(gl.VIEWPORT, &viewport[0])
	w, h := viewport[2], viewport[3]
	if w <= 0 || h <= 0 {
		return nil
	}
	if e := t.resize(w, h); e != nil {
		return e
	}
	if t.valid && !temporalHistoryCompatible(view, projection, t.previousView, t.previousProjection) {
		t.valid = false
	}
	currentProjection := jitterProjection(projection, jitter)
	reprojection := currentProjection.Inverse().Mul(view.Inverse()).Mul(t.previousView).Mul(t.previousProjection)
	var target, read, program, vao, active, depthFunc int32
	gl.GetIntegerv(gl.DRAW_FRAMEBUFFER_BINDING, &target)
	gl.GetIntegerv(gl.READ_FRAMEBUFFER_BINDING, &read)
	gl.GetIntegerv(gl.CURRENT_PROGRAM, &program)
	gl.GetIntegerv(gl.VERTEX_ARRAY_BINDING, &vao)
	gl.GetIntegerv(gl.ACTIVE_TEXTURE, &active)
	gl.GetIntegerv(gl.DEPTH_FUNC, &depthFunc)
	var textures [4]int32
	for i := range textures {
		gl.ActiveTexture(gl.TEXTURE0 + uint32(i))
		gl.GetIntegerv(gl.TEXTURE_BINDING_2D, &textures[i])
	}
	gl.ActiveTexture(uint32(active))
	var depthMask bool
	var colorMask [4]bool
	gl.GetBooleanv(gl.DEPTH_WRITEMASK, &depthMask)
	gl.GetBooleanv(gl.COLOR_WRITEMASK, &colorMask[0])
	caps := [...]uint32{gl.DEPTH_TEST, gl.BLEND, gl.CULL_FACE, gl.SCISSOR_TEST, gl.CLIP_DISTANCE0}
	var enabled [len(caps)]bool
	for i, c := range caps {
		enabled[i] = gl.IsEnabled(c)
		gl.Disable(c)
	}
	defer func() {
		gl.BindFramebuffer(gl.DRAW_FRAMEBUFFER, uint32(target))
		gl.BindFramebuffer(gl.READ_FRAMEBUFFER, uint32(read))
		gl.Viewport(viewport[0], viewport[1], w, h)
		gl.UseProgram(uint32(program))
		gl.BindVertexArray(uint32(vao))
		for i, v := range textures {
			gl.BindTextureUnit(uint32(i), uint32(v))
		}
		gl.ActiveTexture(uint32(active))
		gl.DepthMask(depthMask)
		gl.DepthFunc(uint32(depthFunc))
		gl.ColorMask(colorMask[0], colorMask[1], colorMask[2], colorMask[3])
		for i, c := range caps {
			if enabled[i] {
				gl.Enable(c)
			} else {
				gl.Disable(c)
			}
		}
	}()
	current := t.surfaces[0]
	output := t.surfaces[1+t.index]
	previous := t.surfaces[2-t.index]
	gl.BlitNamedFramebuffer(uint32(target), current.fbo, viewport[0], viewport[1], viewport[0]+w, viewport[1]+h, 0, 0, w, h, gl.COLOR_BUFFER_BIT|gl.DEPTH_BUFFER_BIT, gl.NEAREST)
	gl.BindFramebuffer(gl.DRAW_FRAMEBUFFER, output.fbo)
	gl.Viewport(0, 0, w, h)
	gl.Enable(gl.DEPTH_TEST)
	gl.DepthFunc(gl.ALWAYS)
	gl.DepthMask(true)
	gl.ColorMask(true, true, true, true)
	gl.UseProgram(t.program)
	gl.BindVertexArray(t.vao)
	gl.BindTextureUnit(0, current.color)
	gl.BindTextureUnit(1, current.depth)
	gl.BindTextureUnit(2, previous.color)
	gl.BindTextureUnit(3, previous.depth)
	gl.UniformMatrix4fv(t.projectionLocation, 1, false, &reprojection[0])
	gl.Uniform2f(t.jitterLocation, jitter[0], jitter[1])
	valid := int32(0)
	if t.valid {
		valid = 1
	}
	gl.Uniform1i(t.validLocation, valid)
	infinity := projection[10]
	if projection[11] != 0 {
		infinity /= projection[11]
	}
	gl.Uniform1f(t.infinityLocation, infinity)
	gl.DrawArrays(gl.TRIANGLES, 0, 3)
	drawCalls++
	gl.BlitNamedFramebuffer(output.fbo, uint32(target), 0, 0, w, h, viewport[0], viewport[1], viewport[0]+w, viewport[1]+h, gl.COLOR_BUFFER_BIT|gl.DEPTH_BUFFER_BIT, gl.NEAREST)
	t.previousView, t.previousProjection = view, projection
	t.valid = true
	t.index = 1 - t.index
	return nil
}
func (t *TemporalAA) release() {
	for i := range t.surfaces {
		p := &t.surfaces[i]
		gl.DeleteFramebuffers(1, &p.fbo)
		gl.DeleteTextures(1, &p.color)
		gl.DeleteTextures(1, &p.depth)
		*p = temporalSurface{}
	}
	removeGPUBytes(t.bytes)
	t.bytes = 0
	t.width, t.height = 0, 0
	t.valid = false
}
func (t *TemporalAA) Delete() {
	if t != nil {
		t.release()
		gl.DeleteProgram(t.program)
		gl.DeleteVertexArrays(1, &t.vao)
		*t = TemporalAA{}
	}
}

func jitterProjection(p M4, jitter [2]float32) M4 {
	if jitter == [2]float32{} {
		return p
	}
	for row := 0; row < 4; row++ {
		p[row*4] += jitter[0] * p[row*4+3]
		p[row*4+1] += jitter[1] * p[row*4+3]
	}
	return p
}

// VisibilityProjection matches raster jitter while the original Proj remains
// stable for input rays and UI/name anchors. Reflections/shadows stay unjittered.
func (s *Scene3D) VisibilityProjection() M4 {
	if s.Shadow || s.ClipPlane != [4]float32{} {
		return s.Proj
	}
	return jitterProjection(s.Proj, s.TemporalJitter)
}
func (s *Scene3D) StableGPUProjection() M4 {
	copy := *s
	copy.TemporalJitter = [2]float32{}
	return copy.depthProjection(s.Proj)
}
