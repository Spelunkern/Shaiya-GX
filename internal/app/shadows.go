package app

import (
	"log"

	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

// Only the local character casts a dynamic shadow. NPCs, mobs, remote players
// and scene objects are outside the current shadow scope.
func (a *App) drawSelfShadow() {
	if !a.clientSettings().Shadows || a.state.Self.Concealed {
		return
	}
	v := a.wv
	if v == nil || v.self == nil || v.world == nil || !v.world.HasShadowReceivers() {
		return
	}
	c := v.self
	position := render.V3{c.World[12], c.World[13], c.World[14]}
	// Skip a detached camera beyond the local shadow range.
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
