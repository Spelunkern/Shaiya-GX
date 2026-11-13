package app

import (
	"log"

	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

// beginSceneAntialias redirects the 3D pass into a multisampled target when an
// MSAA level is selected. It returns the target to resolve, or nil.
func (a *App) beginSceneAntialias(w, h int) *render.MSAATarget {
	samples := ui.AntialiasSamples(a.clientSettings().Antialias)
	if samples == 0 || a.frameTarget == nil || a.msaaFailed {
		if samples == 0 && a.msaaTarget != nil {
			a.msaaTarget.Delete()
			a.msaaTarget = nil
		}
		return nil
	}
	samples = min(samples, render.MaxSamples())
	if samples < 2 {
		return nil
	}
	if !a.msaaTarget.Matches(w, h, samples) {
		a.msaaTarget.Delete()
		var err error
		a.msaaTarget, err = render.NewMSAATarget(w, h, samples)
		if err != nil {
			log.Printf("antialias: %v", err)
			a.msaaFailed = true
			return nil
		}
	}
	a.msaaTarget.Begin()
	return a.msaaTarget
}

// endSceneAntialias resolves MSAA or runs FXAA on the finished 3D scene,
// before glow and before any UI, so text and the HUD are never filtered.
func (a *App) endSceneAntialias(msaa *render.MSAATarget) {
	if msaa != nil {
		msaa.Resolve()
		return
	}
	if a.clientSettings().Antialias != ui.AntialiasFXAA || a.fxaaFailed {
		return
	}
	if a.fxaa == nil {
		var err error
		if a.fxaa, err = render.NewFXAA(); err != nil {
			log.Printf("fxaa: %v", err)
			a.fxaaFailed = true
			return
		}
	}
	if err := a.fxaa.Apply(); err != nil {
		log.Printf("fxaa: %v", err)
		a.fxaaFailed = true
	}
}
