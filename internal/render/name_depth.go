package render

import "github.com/go-gl/gl/v4.5-core/gl"

// CaptureNameDepth copies the current frame's depth once before annotations,
// like Phoenix's classic overlay. No per-label CPU readback or raycasts.
func (s *Sprites) CaptureNameDepth(w, h int) {
	s.InvalidateNameDepth()
	if w <= 0 || h <= 0 {
		return
	}
	s.Flush()
	var depthFunc int32
	gl.GetIntegerv(gl.DEPTH_FUNC, &depthFunc)
	s.depthReversed = depthFunc == gl.GEQUAL
	reversed := int32(0)
	if s.depthReversed {
		reversed = 1
	}
	gl.UseProgram(s.prog)
	gl.Uniform1i(gl.GetUniformLocation(s.prog, gl.Str("reversedDepth\x00")), reversed)
	if s.depth == nil || s.depth.W != w || s.depth.H != h {
		if s.depth != nil {
			s.depth.Delete()
		}
		var id uint32
		gl.CreateTextures(gl.TEXTURE_2D, 1, &id)
		gl.TextureStorage2D(id, 1, gl.DEPTH_COMPONENT32F, int32(w), int32(h))
		gl.TextureParameteri(id, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
		gl.TextureParameteri(id, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
		bytes := uint64(w * h * 4)
		addGPUBytes(bytes)
		s.depth = &Texture{ID: id, W: w, H: h, bytes: bytes}
	}
	gl.CopyTextureSubImage2D(s.depth.ID, 0, 0, 0, 0, 0, int32(w), int32(h))
	gl.BindTextureUnit(2, s.depth.ID)
	s.depthValid = true
}

// InvalidateNameDepth prevents clicks from using a previous frame when no
// depth copy was needed in the current one.
func (s *Sprites) InvalidateNameDepth() {
	if s != nil {
		s.depthValid, s.depthReadValid = false, false
	}
}

// NameDepth scopes depth-tested text; all ordinary HUD sprites stay in 2D.
func (s *Sprites) NameDepth(depth float32) {
	if s.nameDepth == depth {
		return
	}
	s.Flush()
	gl.Uniform1f(s.uNameDepth, depth)
	s.nameDepth = depth
}

// NameVisibleAt samples one pixel only on a click, using the same depth and
// tolerance as the label shader. It avoids allowing clicks through walls.
func (s *Sprites) NameVisibleAt(x, y, width, height int, depth float32) bool {
	if s == nil || !s.depthValid || s.depth == nil || s.depth.W != width || s.depth.H != height || x < 0 || y < 0 || x >= width || y >= height {
		return false
	}
	if !s.depthReadValid || s.depthReadX != x || s.depthReadY != y {
		var value float32
		gl.GetTextureSubImage(s.depth.ID, 0, int32(x), int32(height-1-y), 0, 1, 1, 1, gl.DEPTH_COMPONENT, gl.FLOAT, 4, gl.Ptr(&value))
		if s.depthReversed {
			value = 1 - value
		}
		s.depthReadValue = value
		s.depthReadX, s.depthReadY, s.depthReadValid = x, y, true
	}
	return s.depthReadValue+0.000001 >= depth
}
