package render

import (
	"fmt"

	"github.com/go-gl/gl/v4.5-core/gl"
)

// WaterTarget owns a reusable color/depth target for the reflected scene.
type WaterTarget struct {
	fbo, color, depth uint32
	width, height     int32
	bytes             uint64
}

func NewWaterTarget(width, height int) (*WaterTarget, error) {
	if width <= 0 || height <= 0 {
		return nil, fmt.Errorf("water target: invalid size %dx%d", width, height)
	}
	t := &WaterTarget{width: int32(width), height: int32(height)}
	gl.CreateFramebuffers(1, &t.fbo)
	gl.CreateTextures(gl.TEXTURE_2D, 1, &t.color)
	gl.TextureStorage2D(t.color, 1, gl.RGBA8, t.width, t.height)
	gl.TextureParameteri(t.color, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TextureParameteri(t.color, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TextureParameteri(t.color, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TextureParameteri(t.color, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	gl.CreateRenderbuffers(1, &t.depth)
	gl.NamedRenderbufferStorage(t.depth, gl.DEPTH_COMPONENT32F, t.width, t.height)
	gl.NamedFramebufferTexture(t.fbo, gl.COLOR_ATTACHMENT0, t.color, 0)
	gl.NamedFramebufferRenderbuffer(t.fbo, gl.DEPTH_ATTACHMENT, gl.RENDERBUFFER, t.depth)
	if status := gl.CheckNamedFramebufferStatus(t.fbo, gl.FRAMEBUFFER); status != gl.FRAMEBUFFER_COMPLETE {
		t.Delete()
		return nil, fmt.Errorf("water target: framebuffer %#x", status)
	}
	t.bytes = uint64(width) * uint64(height) * 8
	addGPUBytes(t.bytes)
	return t, nil
}

// Capture preserves target bindings, viewport, scissor and clear values.
// The caller restores its camera and scene program after the callback.
func (t *WaterTarget) Capture(clear V3, draw func()) {
	var framebuffer, read int32
	var viewport, scissor [4]int32
	var color [4]float32
	var depth float64
	gl.GetIntegerv(gl.DRAW_FRAMEBUFFER_BINDING, &framebuffer)
	gl.GetIntegerv(gl.READ_FRAMEBUFFER_BINDING, &read)
	gl.GetIntegerv(gl.VIEWPORT, &viewport[0])
	gl.GetIntegerv(gl.SCISSOR_BOX, &scissor[0])
	gl.GetFloatv(gl.COLOR_CLEAR_VALUE, &color[0])
	gl.GetDoublev(gl.DEPTH_CLEAR_VALUE, &depth)
	scissored := gl.IsEnabled(gl.SCISSOR_TEST)
	gl.BindFramebuffer(gl.FRAMEBUFFER, t.fbo)
	gl.Viewport(0, 0, t.width, t.height)
	gl.Disable(gl.SCISSOR_TEST)
	gl.DepthMask(true)
	gl.ClearDepth(depth)
	gl.ClearColor(clear[0], clear[1], clear[2], 1)
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	draw()
	gl.BindFramebuffer(gl.DRAW_FRAMEBUFFER, uint32(framebuffer))
	gl.BindFramebuffer(gl.READ_FRAMEBUFFER, uint32(read))
	gl.Viewport(viewport[0], viewport[1], viewport[2], viewport[3])
	gl.Scissor(scissor[0], scissor[1], scissor[2], scissor[3])
	if scissored {
		gl.Enable(gl.SCISSOR_TEST)
	} else {
		gl.Disable(gl.SCISSOR_TEST)
	}
	gl.ClearColor(color[0], color[1], color[2], color[3])
	gl.ClearDepth(depth)
}

func (t *WaterTarget) Delete() {
	if t == nil {
		return
	}
	gl.DeleteFramebuffers(1, &t.fbo)
	gl.DeleteTextures(1, &t.color)
	gl.DeleteRenderbuffers(1, &t.depth)
	removeGPUBytes(t.bytes)
	*t = WaterTarget{}
}

// Present copies the captured color to the restored caller target. Depth stays
// in the float target; annotations sample it before this copy.
func (t *WaterTarget) Present() {
	var draw int32
	var viewport [4]int32
	gl.GetIntegerv(gl.DRAW_FRAMEBUFFER_BINDING, &draw)
	gl.GetIntegerv(gl.VIEWPORT, &viewport[0])
	gl.BlitNamedFramebuffer(t.fbo, uint32(draw), 0, 0, t.width, t.height, viewport[0], viewport[1], viewport[0]+viewport[2], viewport[1]+viewport[3], gl.COLOR_BUFFER_BIT, gl.NEAREST)
}
