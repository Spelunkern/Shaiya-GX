package app

import (
	"log"
	"math"

	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
)

// Creation preview uses Login.wld and the actor from 0x464110/0x464e70.
// Body-bound framing is a delivery adaptation pending original comparison.
func (a *App) drawCreation(width, height int) {
	w, loaded := a.worlds["creation"]
	if !loaded {
		var err error
		w, err = scene.LoadDungeon(a.assets, "world/Login.wld")
		if err != nil {
			log.Printf("creation world: %v", err)
		}
		a.worlds["creation"] = w
	}
	if w == nil {
		return
	}
	d := a.create.Draft
	ap := scene.Appearance{Race: d.Race, Gender: d.Gender, Job: d.Class, Hair: d.Hair, Face: d.Face, Height: d.Height, Helmet: -1}
	if a.createActor == nil || ap != a.createAppearance {
		a.createAppearance = ap
		var err error
		a.createActor, err = scene.BuildCharacter(a.assets, ap, "000_normal")
		if err != nil {
			log.Printf("creation actor: %v", err)
		}
	}
	pos := render.V3{68, 2.131, 66}
	nav := &a.create.Navigation
	if !a.create.Busy {
		nav.Advance(a.frameDT, .2)
	}
	centerY, distance := a.createActor.PortraitFrame(float32(projectionAspect(width, height)), math.Pi/4, ap.HeightScale())
	eyeOffset, targetOffset := render.V3{0, centerY + .3, max(3, distance)}, render.V3{0, centerY, 0}
	if nav.Face {
		y := float32(1.5) + creationFaceHeight(d.Race, d.Height)
		eyeOffset, targetOffset = render.V3{0, y, 1}, render.V3{0, y, 0}
	}
	eye, target := pos.Add(rotatePreview(eyeOffset, nav.Angle)), pos.Add(targetOffset)
	s := a.scene3d
	s.SetCameraView(eye, target, render.V3{0, 1, 0})
	s.Proj = render.PerspectiveFovLH(math.Pi/4, projectionAspect(width, height), 0.1, 400)
	s.SkyProj = render.PerspectiveFovLH(math.Pi/4, projectionAspect(width, height), 10, 100000)
	s.FogColor, s.FogStart, s.FogEnd = w.FogColor, w.FogStart, w.FogEnd
	s.LightDir = render.V3{-0.513, -1, 0.858}
	s.Begin()
	w.Update(float32(a.frameDT))
	if w.Sky != nil {
		w.Sky.Draw(s, eye)
	}
	w.Draw(s)
	if actor := a.createActor; actor != nil {
		scale := render.Identity()
		scale[5] = ap.HeightScale()
		actor.World = scale.Mul(render.Basis(pos, render.V3{0, 0, -1}, render.V3{0, 1, 0}))
		actor.Update(float32(a.frameDT))
		actor.Draw(a.skinner, s)
	}
	w.DrawWater(s)
	w.DrawMapEffects(s, eye)
	s.End()
}
