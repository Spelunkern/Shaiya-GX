package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
	"log"
)

// The authored point.EFT mixes one-shot rings with persistent loop emitters.
// Its lifetime belongs to the travel command, not a fixed click timeout.
func (a *App) startTravelEffect() {
	v := a.wv
	v.travelEffect = nil
	x, err := scene.LoadEffect(a.assets, "Effect/point.EFT", 0)
	if err != nil {
		log.Printf("movement marker: %v", err)
		return
	}
	m := render.Identity()
	m[12], m[13], m[14] = v.destination[0], v.destination[1], v.destination[2]
	x.SetTransform(m)
	v.travelEffect = x
}

func (a *App) updateTravelEffect(dt float32) {
	v := a.wv
	if !v.travel {
		v.travelEffect = nil
		return
	}
	if v.travelEffect != nil {
		v.travelEffect.Update(dt)
	}
}
