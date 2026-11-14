package app

import (
	"github.com/go-gl/gl/v4.5-core/gl"
	"math"
)

// EP5 0x56bc40/0x56be20 use ui_rapis001 for success, 002 for failure.
// Its service panels flash Lensflare/flare0.jpg additively over the target
// for 500ms (0x4a8f26). Trigger once from a confirmed operation, never a click.
func (a *App) forgeResult(success bool) {
	name := "Sound/ui_rapis002.ogg"
	a.blacksmith.shineUntil = 0
	if success {
		name = "Sound/ui_rapis001.ogg"
		a.blacksmith.shineUntil = a.clock + .5
	}
	if a.sound != nil {
		a.audioError(name, a.sound.Play(name, 1))
	}
}

func (a *App) drawForgeShine(x, y int) {
	remaining := a.blacksmith.shineUntil - a.clock
	if remaining <= 0 || remaining > .5 || a.assets == nil {
		return
	}
	tex := a.assets.Texture("Lensflare/flare0.jpg")
	if tex == nil {
		return
	}
	sx, sy := a.blacksmithUI.Slot(0)
	if sx < 0 {
		return
	}
	progress := (.5 - remaining) / .5
	alpha := float32(math.Sin(progress * math.Pi))
	a.sprites.Flush()
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE)
	a.sprites.Draw(tex, float32(x+sx-32), float32(y+sy-32), 100, 100, 0, 0, 1, 1, [4]float32{1, 1, 1, alpha})
	a.sprites.Flush()
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
}
