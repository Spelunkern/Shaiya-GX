package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"testing"
)

func TestFreeCameraKeepsIdleFacingAndSteersOnMovement(t *testing.T) {
	a := inputApp()
	a.wv = &worldView{pos: render.V3{100, 0, 100}, eye: render.V3{95, 2, 100}, facing: render.V3{0, 0, 1}, sendT: 100}
	a.updateWorld(&platform.Input{RightDown: true, MouseLook: true}, .016)
	if a.wv.facing != (render.V3{0, 0, 1}) {
		t.Fatal("idle orbit turned character", a.wv.facing)
	}
	a.wv.moving, a.wv.motion = true, 1
	a.updateWorld(&platform.Input{Down: "W", MouseLook: true}, .016)
	if a.wv.facing[0] < .99 || a.wv.pos[0] <= 100 {
		t.Fatal("movement did not use camera heading")
	}
}
