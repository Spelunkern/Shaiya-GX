package app

import (
	"log"

	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
)

func (a *App) drawActorShadow(c *scene.Character) {
	if !a.clientSettings().Shadows {
		return
	}
	v := a.wv
	if c == nil || v == nil || v.world == nil || !v.world.HasShadowReceivers() {
		return
	}
	position := render.V3{c.World[12], c.World[13], c.World[14]}
	// Bound capture cost to nearby actors; distant actors keep baked world light.
	if position.Sub(v.eye).Len() > 50 {
		return
	}
	if !v.shadowAttempted {
		v.shadowAttempted = true
		var err error
		v.shadow, err = render.NewShadowMap()
		if err != nil {
			log.Printf("actor shadows: %v", err)
		}
	}
	if v.shadow != nil {
		c.DrawShadow(v.shadow, a.skinner, a.scene3d, v.world)
	}
}
