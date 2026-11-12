package render

import "github.com/go-gl/gl/v4.5-core/gl"

// ClearBackground replaces only the colour buffer and preserves the caller's
// clear colour; the world depth and later UI passes keep their existing state.
func ClearBackground(color V3) {
	clear := [4]float32{color[0], color[1], color[2], 1}
	gl.ClearBufferfv(gl.COLOR, 0, &clear[0])
}

// InViewport draws a 3D UI region without clearing its background or changing
// the enclosing viewport/scissor. Coordinates are framebuffer pixels from top left.
func InViewport(x, y, w, h, screenHeight int, draw func()) {
	var viewport, scissor [4]int32
	var clearDepth float64
	gl.GetIntegerv(gl.VIEWPORT, &viewport[0])
	gl.GetIntegerv(gl.SCISSOR_BOX, &scissor[0])
	gl.GetDoublev(gl.DEPTH_CLEAR_VALUE, &clearDepth)
	enabled := gl.IsEnabled(gl.SCISSOR_TEST)
	gl.Viewport(int32(x), int32(screenHeight-y-h), int32(w), int32(h))
	gl.Scissor(int32(x), int32(screenHeight-y-h), int32(w), int32(h))
	gl.Enable(gl.SCISSOR_TEST)
	gl.DepthMask(true)
	// UI previews use ordinary depth independently of the world's reverse Z.
	gl.ClearDepth(1)
	gl.Clear(gl.DEPTH_BUFFER_BIT)
	draw()
	gl.ClearDepth(clearDepth)
	gl.Viewport(viewport[0], viewport[1], viewport[2], viewport[3])
	gl.Scissor(scissor[0], scissor[1], scissor[2], scissor[3])
	if !enabled {
		gl.Disable(gl.SCISSOR_TEST)
	}
}
