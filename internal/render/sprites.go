package render

import (
	"unsafe"

	"github.com/go-gl/gl/v4.5-core/gl"
)

// Sprites batches textured quads in pixel coordinates (origin top-left),
// the space the original UI is laid out in. Blending is the D3D9 UI state
// SRCALPHA / INVSRCALPHA.
type Sprites struct {
	prog     uint32
	vao, vbo uint32
	uScreen  int32
	verts    []float32
	tex      *Texture
	cap      int
}

const spriteVS = `#version 450 core
layout(location=0) in vec2 pos;
layout(location=1) in vec2 uv;
layout(location=2) in vec4 color;
uniform vec2 screen;
out vec2 vUV;
out vec4 vColor;
void main() {
	vUV = uv;
	vColor = color;
	gl_Position = vec4(pos.x / screen.x * 2.0 - 1.0, 1.0 - pos.y / screen.y * 2.0, 0.0, 1.0);
}`

const spriteFS = `#version 450 core
layout(binding=0) uniform sampler2D tex;
in vec2 vUV;
in vec4 vColor;
out vec4 frag;
void main() { frag = texture(tex, vUV) * vColor; }`

const floatsPerVertex = 8

func NewSprites(maxQuads int) (*Sprites, error) {
	p, err := Program(spriteVS, spriteFS)
	if err != nil {
		return nil, err
	}
	s := &Sprites{prog: p, uScreen: gl.GetUniformLocation(p, gl.Str("screen\x00")), cap: maxQuads}
	gl.CreateBuffers(1, &s.vbo)
	gl.NamedBufferStorage(s.vbo, maxQuads*6*floatsPerVertex*4, nil, gl.DYNAMIC_STORAGE_BIT)
	gl.CreateVertexArrays(1, &s.vao)
	gl.VertexArrayVertexBuffer(s.vao, 0, s.vbo, 0, floatsPerVertex*4)
	for i, n := range []int32{2, 2, 4} {
		off := []uint32{0, 2, 4}[i]
		gl.EnableVertexArrayAttrib(s.vao, uint32(i))
		gl.VertexArrayAttribFormat(s.vao, uint32(i), n, gl.FLOAT, false, off*4)
		gl.VertexArrayAttribBinding(s.vao, uint32(i), 0)
	}
	return s, nil
}

// Begin starts a frame of w x h pixels.
func (s *Sprites) Begin(w, h int) {
	gl.UseProgram(s.prog)
	gl.Uniform2f(s.uScreen, float32(w), float32(h))
	gl.BindVertexArray(s.vao)
	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
	gl.Disable(gl.DEPTH_TEST)
}

// Draw queues a quad at (x,y) of size (w,h) with texture rect (u0,v0)-(u1,v1).
func (s *Sprites) Draw(t *Texture, x, y, w, h, u0, v0, u1, v1 float32, rgba [4]float32) {
	if s.tex != t || len(s.verts) >= s.cap*6*floatsPerVertex {
		s.Flush()
		s.tex = t
	}
	v := func(px, py, u, vv float32) {
		s.verts = append(s.verts, px, py, u, vv, rgba[0], rgba[1], rgba[2], rgba[3])
	}
	v(x, y, u0, v0)
	v(x+w, y, u1, v0)
	v(x+w, y+h, u1, v1)
	v(x, y, u0, v0)
	v(x+w, y+h, u1, v1)
	v(x, y+h, u0, v1)
}

// DrawImage draws a whole texture at its pixel size.
func (s *Sprites) DrawImage(t *Texture, x, y float32) {
	s.Draw(t, x, y, float32(t.W), float32(t.H), 0, 0, 1, 1, [4]float32{1, 1, 1, 1})
}

func (s *Sprites) Flush() {
	if len(s.verts) == 0 || s.tex == nil {
		s.verts = s.verts[:0]
		return
	}
	gl.NamedBufferSubData(s.vbo, 0, len(s.verts)*4, unsafe.Pointer(&s.verts[0]))
	gl.BindTextureUnit(0, s.tex.ID)
	gl.DrawArrays(gl.TRIANGLES, 0, int32(len(s.verts)/floatsPerVertex))
	s.verts = s.verts[:0]
}

// End flushes the remaining quads.
func (s *Sprites) End() { s.Flush() }
