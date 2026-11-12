package render

import (
	"fmt"

	"github.com/go-gl/gl/v4.5-core/gl"
)

// CaptureTexture renders a 4x MSAA snapshot with its own depth buffer, following
// Phoenix PlayerFrameHud. The caller must
// flush pending sprites first and restore its drawing program afterwards.
func CaptureTexture(w, h int, draw func()) (*Texture, error) {
	var framebuffer, readFramebuffer int32
	var viewport, scissor [4]int32
	var clear [4]float32
	var colorMask [4]bool
	var clearDepth float64
	gl.GetIntegerv(gl.DRAW_FRAMEBUFFER_BINDING, &framebuffer)
	gl.GetIntegerv(gl.READ_FRAMEBUFFER_BINDING, &readFramebuffer)
	gl.GetIntegerv(gl.VIEWPORT, &viewport[0])
	gl.GetIntegerv(gl.SCISSOR_BOX, &scissor[0])
	gl.GetFloatv(gl.COLOR_CLEAR_VALUE, &clear[0])
	gl.GetBooleanv(gl.COLOR_WRITEMASK, &colorMask[0])
	gl.GetDoublev(gl.DEPTH_CLEAR_VALUE, &clearDepth)
	scissored := gl.IsEnabled(gl.SCISSOR_TEST)
	var fbo, drawFBO, depth, msColor, color uint32
	gl.CreateFramebuffers(1, &fbo)
	gl.CreateFramebuffers(1, &drawFBO)
	gl.CreateRenderbuffers(1, &depth)
	gl.CreateRenderbuffers(1, &msColor)
	defer gl.DeleteFramebuffers(1, &fbo)
	defer gl.DeleteFramebuffers(1, &drawFBO)
	defer gl.DeleteRenderbuffers(1, &depth)
	defer gl.DeleteRenderbuffers(1, &msColor)
	gl.NamedRenderbufferStorageMultisample(depth, 4, gl.DEPTH_COMPONENT24, int32(w), int32(h))
	gl.NamedRenderbufferStorageMultisample(msColor, 4, gl.RGBA8, int32(w), int32(h))
	gl.CreateTextures(gl.TEXTURE_2D, 1, &color)
	gl.TextureStorage2D(color, 1, gl.RGBA8, int32(w), int32(h))
	gl.TextureParameteri(color, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TextureParameteri(color, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TextureParameteri(color, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TextureParameteri(color, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	gl.NamedFramebufferTexture(fbo, gl.COLOR_ATTACHMENT0, color, 0)
	gl.NamedFramebufferRenderbuffer(drawFBO, gl.COLOR_ATTACHMENT0, gl.RENDERBUFFER, msColor)
	gl.NamedFramebufferRenderbuffer(drawFBO, gl.DEPTH_ATTACHMENT, gl.RENDERBUFFER, depth)
	for _, target := range []uint32{fbo, drawFBO} {
		if status := gl.CheckNamedFramebufferStatus(target, gl.FRAMEBUFFER); status != gl.FRAMEBUFFER_COMPLETE {
			gl.DeleteTextures(1, &color)
			return nil, fmt.Errorf("snapshot framebuffer: %#x", status)
		}
	}
	gl.BindFramebuffer(gl.FRAMEBUFFER, drawFBO)
	gl.Viewport(0, 0, int32(w), int32(h))
	gl.Disable(gl.SCISSOR_TEST)
	gl.DepthMask(true)
	gl.ColorMask(true, true, true, true)
	gl.ClearColor(0, 0, 0, 1)
	gl.ClearDepth(1)
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	// Phoenix SubViewport.transparent_bg=false: opaque/cutout materials may
	// contain zero texture alpha; only the final HUD mask supplies coverage.
	gl.ColorMask(true, true, true, false)
	draw()
	gl.ColorMask(colorMask[0], colorMask[1], colorMask[2], colorMask[3])
	gl.BlitNamedFramebuffer(drawFBO, fbo, 0, 0, int32(w), int32(h), 0, 0, int32(w), int32(h), gl.COLOR_BUFFER_BIT, gl.NEAREST)
	gl.BindFramebuffer(gl.DRAW_FRAMEBUFFER, uint32(framebuffer))
	gl.BindFramebuffer(gl.READ_FRAMEBUFFER, uint32(readFramebuffer))
	gl.Viewport(viewport[0], viewport[1], viewport[2], viewport[3])
	gl.Scissor(scissor[0], scissor[1], scissor[2], scissor[3])
	if scissored {
		gl.Enable(gl.SCISSOR_TEST)
	}
	gl.ClearColor(clear[0], clear[1], clear[2], clear[3])
	gl.ClearDepth(clearDepth)
	bytes := uint64(w * h * 4)
	addGPUBytes(bytes)
	return &Texture{ID: color, W: w, H: h, bytes: bytes}, nil
}
