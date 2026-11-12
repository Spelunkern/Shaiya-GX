package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
	"math"
)

func (a *App) drawGroundItems() {
	v, s := a.wv, a.scene3d
	for id := range v.groundStart {
		if _, exists := a.state.GroundItems[id]; !exists {
			delete(v.groundStart, id)
		}
	}
	for i := range v.groundBatches {
		v.groundBatches[i] = v.groundBatches[i][:0]
		v.groundColors[i] = v.groundColors[i][:0]
	}
	if len(a.state.GroundItems) == 0 {
		return
	}
	if v.groundStart == nil {
		v.groundStart = make(map[uint32]float64)
	}
	frustum := render.ViewFrustum(s.View, s.Proj)
	for id, item := range a.state.GroundItems {
		start, exists := v.groundStart[id]
		if !exists {
			start = a.clock
			v.groundStart[id] = start
		}
		kind := scene.GroundItemModel(item.Type)
		model := a.assets.GroundModel(kind)
		if model == nil {
			continue
		}
		pos := groundItemPosition(v.world, item)
		if !render.ClipSphereVisible(s.ClipPlane, pos, model.Radius) || !frustum.SphereVisible(pos, model.Radius) {
			continue
		}
		world := render.Identity()
		world[12], world[13], world[14] = pos[0], pos[1], pos[2]
		v.groundBatches[kind] = append(v.groundBatches[kind], world)
		brightness := groundBrightness(a.clock-start, item.Type, item.TypeID)
		v.groundColors[kind] = append(v.groundColors[kind], [4]float32{brightness, brightness, brightness, 1})
	}
	for kind, batch := range v.groundBatches {
		if len(batch) == 0 {
			continue
		}
		part := a.assets.GroundModel(kind).Part
		s.DrawGroundItems(part.Mesh, part.Tex, batch, v.groundColors[kind])
	}
}

// 0x428470 changes texture factor at 2 units/s, clamped to .3–1.
func groundBrightness(age float64, typ, id uint8) float32 {
	if typ == 44 && id == 246 {
		return 1
	}
	phase := math.Mod(max(age, 0), .7)
	if phase > .35 {
		phase = .7 - phase
	}
	return float32(.3 + phase*2)
}
