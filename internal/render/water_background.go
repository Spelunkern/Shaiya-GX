package render

import "github.com/go-gl/gl/v4.5-core/gl"

// CaptureWaterBackground copies the already drawn scene to avoid a second
// world traversal. The result is screen-space refraction, so foreground
// geometry near water edges may differ from a clipped native capture.
func (s *Scene3D) CaptureWaterBackground() {
	var viewport [4]int32
	gl.GetIntegerv(gl.VIEWPORT, &viewport[0])
	w, h := int(viewport[2]), int(viewport[3])
	if w <= 0 || h <= 0 {
		return
	}
	if s.waterBackground == nil || s.waterBackground.W != w || s.waterBackground.H != h {
		if s.waterBackground != nil {
			s.waterBackground.Delete()
		}
		s.waterBackgroundFor = 0 // a recycled texture name must be reattached
		var id uint32
		gl.CreateTextures(gl.TEXTURE_2D, 1, &id)
		gl.TextureStorage2D(id, 1, gl.RGBA8, int32(w), int32(h))
		gl.TextureParameteri(id, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
		gl.TextureParameteri(id, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
		gl.TextureParameteri(id, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
		gl.TextureParameteri(id, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
		bytes := uint64(w) * uint64(h) * 4
		addGPUBytes(bytes)
		s.waterBackground = &Texture{ID: id, W: w, H: h, bytes: bytes}
	}
	var read, draw int32
	gl.GetIntegerv(gl.READ_FRAMEBUFFER_BINDING, &read)
	gl.GetIntegerv(gl.DRAW_FRAMEBUFFER_BINDING, &draw)
	if draw != 0 && framebufferSamples(draw) > 0 {
		// A multisampled scene cannot be copied; resolve it with a blit.
		if s.waterBackgroundFBO == 0 || s.waterBackgroundFor != s.waterBackground.ID {
			if s.waterBackgroundFBO == 0 {
				gl.CreateFramebuffers(1, &s.waterBackgroundFBO)
			}
			gl.NamedFramebufferTexture(s.waterBackgroundFBO, gl.COLOR_ATTACHMENT0, s.waterBackground.ID, 0)
			s.waterBackgroundFor = s.waterBackground.ID
		}
		scissored := gl.IsEnabled(gl.SCISSOR_TEST)
		gl.Disable(gl.SCISSOR_TEST)
		gl.BlitNamedFramebuffer(uint32(draw), s.waterBackgroundFBO, viewport[0], viewport[1], viewport[0]+int32(w), viewport[1]+int32(h), 0, 0, int32(w), int32(h), gl.COLOR_BUFFER_BIT, gl.NEAREST)
		if scissored {
			gl.Enable(gl.SCISSOR_TEST)
		}
		return
	}
	if read != draw {
		gl.BindFramebuffer(gl.READ_FRAMEBUFFER, uint32(draw))
	}
	gl.CopyTextureSubImage2D(s.waterBackground.ID, 0, 0, 0, viewport[0], viewport[1], int32(w), int32(h))
	if read != draw {
		gl.BindFramebuffer(gl.READ_FRAMEBUFFER, uint32(read))
	}
}

func (s *Scene3D) ReleaseWaterBackground() {
	if s != nil && s.waterBackground != nil {
		s.waterBackground.Delete()
		s.waterBackground = nil
	}
	if s != nil && s.waterBackgroundFBO != 0 {
		gl.DeleteFramebuffers(1, &s.waterBackgroundFBO)
		s.waterBackgroundFBO, s.waterBackgroundFor = 0, 0
	}
}
