package app

import (
	"log"

	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

// beginSceneAntialias redirects the 3D pass into a multisampled target when an
// MSAA level is selected. It returns the target to resolve, or nil.
func (a *App) beginSceneAntialias(w, h int) *render.MSAATarget {
	mode := a.clientSettings().Antialias
	if mode != a.aaMode || a.aaSize != [2]int{w, h} {
		a.msaaFailed, a.fxaaFailed, a.taaFailed = false, false, false
		a.taa.Reset()
		a.aaMode = mode
		a.aaSize = [2]int{w, h}
	}
	a.scene3d.TemporalJitter = [2]float32{}
	if mode != ui.AntialiasTAA {
		a.taa.Delete()
		a.taa = nil
	}
	if mode != ui.AntialiasFXAA {
		a.fxaa.Delete()
		a.fxaa = nil
	}
	if mode == ui.AntialiasTAA && !a.taaFailed && a.frameTarget != nil && !a.loading.active {
		if a.taa == nil {
			var err error
			a.taa, err = render.NewTemporalAA()
			if err != nil {
				log.Printf("taa: %v", err)
				a.taaFailed = true
			}
		}
		if a.taa != nil {
			context := [4]int{int(a.screen), 0, a.chars.Selected, 0}
			if a.state != nil {
				context[1] = int(a.state.MapID)
			}
			if a.creating {
				context[3] = 1
			}
			if context != a.aaContext || a.frameDT > .2 {
				a.taa.Reset()
			}
			a.aaContext = context
			a.scene3d.TemporalJitter = a.taa.NextJitter(w, h)
		}
	}
	samples := ui.AntialiasSamples(mode)
	if samples == 0 || a.frameTarget == nil || a.msaaFailed {
		if samples == 0 && a.msaaTarget != nil {
			a.msaaTarget.Delete()
			a.msaaTarget = nil
		}
		return nil
	}
	requested := samples
	samples = 0
	for _, n := range a.aaSamples {
		if n <= requested {
			samples = max(samples, n)
		}
	}
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
	defer func() { a.scene3d.TemporalJitter = [2]float32{} }()
	if a.loading.active {
		a.taa.Reset()
	}
	if a.clientSettings().Antialias == ui.AntialiasTAA && a.taa != nil && !a.taaFailed && !a.loading.active && a.frameTarget != nil {
		if err := a.taa.Apply(a.scene3d.View, a.scene3d.StableGPUProjection(), a.scene3d.TemporalJitter); err != nil {
			log.Printf("taa: %v", err)
			a.taaFailed = true
		}
		return
	}
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
