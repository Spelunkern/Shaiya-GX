package render

import "github.com/go-gl/gl/v4.5-core/gl"

// KeepSceneAlphaOpaque preserves destination alpha for native EFT layers that
// blend against it (notably Dispel). Texture/source alpha still controls RGB
// transparency. Call the returned function before leaving the scene target.
func KeepSceneAlphaOpaque() func() {
	var mask [4]bool
	gl.GetBooleanv(gl.COLOR_WRITEMASK, &mask[0])
	scissored := gl.IsEnabled(gl.SCISSOR_TEST)
	gl.Disable(gl.SCISSOR_TEST)
	gl.ColorMask(false, false, false, true)
	opaque := [4]float32{0, 0, 0, 1}
	gl.ClearBufferfv(gl.COLOR, 0, &opaque[0])
	if scissored {
		gl.Enable(gl.SCISSOR_TEST)
	}
	gl.ColorMask(mask[0], mask[1], mask[2], false)
	return func() { gl.ColorMask(mask[0], mask[1], mask[2], mask[3]) }
}
