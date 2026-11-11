package app

import (
	"log"
	"math"

	"github.com/Spelunkern/shaiya-go-client/internal/render"
	"github.com/Spelunkern/shaiya-go-client/internal/scene"
)

// Character select camera (0x46a5a0, full-body case; Phoenix
// tools/reference/ep5_character_selection.json).
var (
	selectEye    = render.V3{275.0299987792969, 1.9570000171661377, 180.1959991455078}
	selectTarget = render.V3{271.09698486328125, 1.8660000562667847, 178.51499938964844}
)

// selectWorld returns select_a (Light, map 100) or select_b (Fury, 101).
func (a *App) selectWorld(fury bool) *scene.World {
	stem := "select_a"
	if fury {
		stem = "select_b"
	}
	if w, ok := a.worlds[stem]; ok {
		return w
	}
	w, err := scene.LoadField(a.assets, stem)
	if err != nil {
		log.Printf("scene %s: %v", stem, err)
	}
	a.worlds[stem] = w
	for f := range a.assets.Missing {
		log.Printf("scene %s: missing %s", stem, f)
		delete(a.assets.Missing, f)
	}
	return w
}

func (a *App) drawWorld(w *scene.World, eye, target render.V3, width, height int) {
	if w == nil || a.scene3d == nil {
		return
	}
	s := a.scene3d
	s.View = render.LookAtLH(eye, target, render.V3{0, 1, 0})
	s.Proj = render.PerspectiveFovLH(math.Pi/4, float32(width)/float32(height), 0.1, 400)
	s.FogColor, s.FogStart, s.FogEnd = w.FogColor, w.FogStart, w.FogEnd
	// Select maps light (Phoenix account_ep5_render, X mirrored back to raw).
	s.LightDir = render.V3{-0.513, -1, 0.858}
	s.Begin()
	w.Draw(s)
	s.End()
}
