package app

import (
	"log"
	"math"

	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
)

// Native 0x546630: family, job and gender choose the face camera.
func portraitCamera(ap scene.Appearance) (eye, target render.V3) {
	eye, target = render.V3{.1, 1.45, -.3}, render.V3{.01, 1.5, 0}
	switch ap.Race {
	case 0:
		if ap.Job == 5 {
			if ap.Gender == 0 {
				eye, target = render.V3{-.3, 1.5, -.3}, render.V3{.01, 1.55, 0}
			} else {
				target = render.V3{.05, 1.5, 0}
			}
		} else if ap.Gender == 0 {
			eye, target = render.V3{-.3, 1.55, -.35}, render.V3{.01, 1.6, 0}
		} else {
			eye[0], eye[2], target[0] = -.2, -.35, .11
		}
	case 1:
		if ap.Gender == 0 {
			eye[1], eye[2], target[1] = 1.55, -.35, 1.6
		}
	case 2:
		if ap.Gender == 0 {
			eye[1], eye[2], target[1] = 1.5, -.4, 1.55
		}
	case 3:
		eye[2] = -.4
		if ap.Job == 3 {
			if ap.Gender == 0 {
				eye[1], target[0], target[1] = 1.75, -.09, 1.8
			} else {
				eye[1], target[1] = 1.68, 1.73
			}
		} else if ap.Gender == 0 {
			eye, target = render.V3{-.04, 1.8, -.48}, render.V3{-.04, 1.9, 0}
		} else {
			eye, target = render.V3{.2, 1.68, -.3}, render.V3{.1, 1.73, 0}
		}
	}
	return
}

func (a *App) hudPortrait(w, h int) *render.Texture {
	v := a.wv
	if v == nil {
		return nil
	}
	slot, ok := a.selfSlot()
	if !ok {
		return nil
	}
	ap := a.appearance(&slot)
	if v.portraitLoaded && v.portraitAppearance == ap {
		return v.portrait
	}
	v.portraitLoaded, v.portraitAppearance = true, ap
	if v.portrait != nil {
		v.portrait.Delete()
		v.portrait = nil
	}
	actor, err := scene.BuildCharacter(a.assets, ap, "000_normal")
	if err != nil {
		log.Printf("HUD portrait: %v", err)
		return nil
	}
	a.sprites.Flush()
	// Phoenix PlayerFrameHud captures at 384px with 4x MSAA, independently
	// of the small HUD rectangle. Rendering directly at 64px loses face detail.
	v.portrait, err = render.CaptureTexture(384, 384, func() {
		s := a.actorPreviewScene()
		eye, target := portraitCamera(ap)
		s.SetCameraView(eye, target, render.V3{0, 1, 0})
		s.Proj = render.PerspectiveFovLH(math.Pi/4, 1, .1, 400)
		s.FogStart, s.FogEnd = 0, 0
		s.LightDir = render.V3{-.513, -1, .858}
		s.Begin()
		actor.World = render.Identity()
		actor.Draw(a.skinner, &s)
		s.End()
	})
	a.sprites.Begin(w, h)
	if err != nil {
		log.Printf("HUD portrait: %v", err)
	}
	return v.portrait
}
