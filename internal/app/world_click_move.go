package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

// Default USE_MOUSE=true (0x408bf0 / 0x43bfa0). Picking runs after entities,
// loot, NPCs and panels, so a consumed click never becomes a terrain command.
func (a *App) moveToGround(in *platform.Input, width, height int) {
	v := a.wv
	if a.scene3d == nil || !v.world.HasTerrain() {
		return
	}
	origin, direction, ok := clickRay(in.MouseX, in.MouseY, width, height, a.scene3d.View.Mul(a.scene3d.Proj))
	if !ok {
		return
	}
	inside := func(p render.V3) bool {
		size := float32(v.world.WLD.MapSize)
		return p[0] >= 0 && p[2] >= 0 && p[0] <= size && p[2] <= size
	}
	// Bracket the first terrain crossing, then refine it in native world units.
	previous := float32(0)
	for distance := float32(.5); distance <= 400; distance += .5 {
		p := origin.Add(direction.Scale(distance))
		if !inside(p) {
			previous = distance
			continue
		}
		if p[1] > v.world.Height(p[0], p[2]) {
			previous = distance
			continue
		}
		lo, hi := previous, distance
		for range 12 {
			mid := (lo + hi) * .5
			p = origin.Add(direction.Scale(mid))
			if p[1] > v.world.Height(p[0], p[2]) {
				lo = mid
			} else {
				hi = mid
			}
		}
		p = origin.Add(direction.Scale(hi))
		p[1] = v.world.Height(p[0], p[2])
		v.destination, v.travel, v.autoRun = p, true, false
		a.startTravelEffect()
		return
	}
}
