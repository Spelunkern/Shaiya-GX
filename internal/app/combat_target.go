package app

import (
	"math"

	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
)

// pickGroundLabel uses the exact label anchor and font rectangle from the last
// rendered frame. Rapture still validates ownership, distance and inventory.
func (a *App) pickGroundLabel(in *platform.Input, w, h int) uint32 {
	if !in.Clicked || a.scene3d == nil || a.font == nil || a.sprites == nil || a.wv == nil {
		return 0
	}
	vp := a.scene3d.View.Mul(a.scene3d.Proj)
	bestDepth, bestID := float32(math.MaxFloat32), uint32(0)
	for id, it := range a.state.GroundItems {
		pos := groundItemLabelPosition(a.wv.world, it)
		if pos.Sub(a.wv.pos).Len() > 50 {
			continue
		}
		x, y, depth, ok := projectLabel(pos, vp, w, h)
		if !ok {
			continue
		}
		width := a.font.Width(a.groundItemName(it))
		if in.MouseX < x-width/2-1 || in.MouseX > x+(width+1)/2+1 || in.MouseY < y-a.font.Height-1 || in.MouseY > y+1 {
			continue
		}
		if depth < bestDepth || depth == bestDepth && (bestID == 0 || id < bestID) {
			bestDepth, bestID = depth, id
		}
	}
	if bestID != 0 && a.pickNameVisibleAt(in.MouseX, in.MouseY, w, h, bestDepth) {
		return bestID
	}
	return 0
}

// pickNPC uses the displayed position while eligibility uses the confirmed
// position. Panel capture and ground-item handling occur before this test.
func (a *App) pickNPC(in *platform.Input, w, h int) uint32 {
	if !in.Clicked || a.scene3d == nil || a.wv == nil {
		return 0
	}
	vp := a.scene3d.View.Mul(a.scene3d.Proj)
	origin, direction, rayOK := clickRay(in.MouseX, in.MouseY, w, h, vp)
	_, centre, _ := clickRay(w/2, h/2, w, h, vp) // the camera heading
	best, bestDepth, id := float32(40*40), float32(math.MaxFloat32), uint32(0)
	bestDistance := float32(math.MaxFloat32)
	for key, e := range a.state.Entities {
		if e.Kind != game.KindNpc {
			continue
		}
		actor := a.wv.npcs[key]
		if actor == nil || actor.actor == nil || actor.entity != e {
			continue
		}
		// 0x43baf0: the pick region is a camera-facing rectangle at the NPC's
		// feet, 2*half-width wide and as tall as its plate (MON Height + 0.3);
		// there is no distance cut-off beyond the NPC being in the list.
		if rayOK {
			distance, hit := a.npcBillboardHit(e, actor.actor, origin, direction, centre)
			if !hit {
				continue
			}
			sample := origin.Add(direction.Scale(max(distance-1, 0)))
			_, _, depth, projected := projectLabel(sample, vp, w, h)
			if !projected || a.sprites != nil && !a.pickNameVisibleAt(in.MouseX, in.MouseY, w, h, depth) {
				continue
			}
			if distance < bestDistance || distance == bestDistance && (id == 0 || key < id) {
				bestDistance, id = distance, key
			}
			continue
		}
		pos := a.visualEntityPosition(e).Add(render.V3{0, 1, 0})
		x, y, depth, ok := projectLabel(pos, vp, w, h)
		if !ok {
			continue
		}
		px, py := float32(x-in.MouseX), float32(y-in.MouseY)
		d := px*px + py*py
		if bestDistance == math.MaxFloat32 && (d < best || d == best && (depth < bestDepth || depth == bestDepth && (id == 0 || key < id))) {
			best, bestDepth, id = d, depth, key
		}
	}
	if id != 0 && bestDistance < math.MaxFloat32 && rayOK {
		// Do not open an NPC behind a player or monster under the pointer.
		for key, e := range a.state.Entities {
			if e.Dead || e.ID == a.state.SelfID {
				continue
			}
			var distance float32
			var hit bool
			switch e.Kind {
			case game.KindMob:
				if m := a.wv.mobs[key]; m != nil && m.entity == e && m.actor != nil {
					distance, hit = a.monsterRayHit(m.actor, origin, direction)
				}
			case game.KindUser:
				if p := a.wv.actors[key]; p != nil && p.entity == e && p.character != nil {
					if clo, chi, ok := p.character.CoarseBounds(); ok {
						if _, coarseHit := rayBox(origin, direction, clo, chi); !coarseHit {
							continue
						}
					}
					if lo, hi, valid := p.character.WorldBounds(); valid {
						distance, hit = rayBox(origin, direction, lo, hi)
					}
				}
			}
			if hit && distance < bestDistance {
				return 0
			}
		}
	}
	return id
}

// EP5.4's monster picker (0x43B8C0) ray-tests a vertical quad facing the
// camera horizontally. Its width is Monster.SData Size (0x432E36) and
// its height follows the scaled MON height plus .2 (0x4318E4).
func (a *App) monsterRayHit(m *scene.CreatureActor, origin, direction render.V3) (float32, bool) {
	base := render.V3{m.World[12], m.World[13], m.World[14]}
	view := a.scene3d.View
	right := render.V3{view[0], 0, view[8]}.Norm()
	return rayVerticalQuad(origin, direction, base, right, float32(m.Definition.Size), m.MonsterPlateAnchor()[1]-base[1])
}

func rayVerticalQuad(origin, direction, base, right render.V3, width, height float32) (float32, bool) {
	normal := render.V3{-right[2], 0, right[0]}
	dot := direction.Dot(normal)
	if math.Abs(float64(dot)) < 1e-7 || width <= 0 || height <= 0 {
		return 0, false
	}
	distance := base.Sub(origin).Dot(normal) / dot
	hit := origin.Add(direction.Scale(distance)).Sub(base)
	horizontal := hit.Dot(right)
	return distance, distance >= 0 && horizontal >= -width*.5 && horizontal <= width*.5 && hit[1] >= 0 && hit[1] <= height
}

func clickRay(x, y, w, h int, vp render.M4) (render.V3, render.V3, bool) {
	if w <= 0 || h <= 0 || x < 0 || y < 0 || x >= w || y >= h {
		return render.V3{}, render.V3{}, false
	}
	inv := vp.Inverse()
	sx, sy := (float32(x)+.5)*2/float32(w)-1, 1-(float32(y)+.5)*2/float32(h)
	unproject := func(z float32) (render.V3, bool) {
		cw := sx*inv[3] + sy*inv[7] + z*inv[11] + inv[15]
		if cw == 0 || math.IsNaN(float64(cw)) || math.IsInf(float64(cw), 0) {
			return render.V3{}, false
		}
		p := render.V3{(sx*inv[0] + sy*inv[4] + z*inv[8] + inv[12]) / cw, (sx*inv[1] + sy*inv[5] + z*inv[9] + inv[13]) / cw, (sx*inv[2] + sy*inv[6] + z*inv[10] + inv[14]) / cw}
		for _, v := range p {
			if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
				return render.V3{}, false
			}
		}
		return p, true
	}
	near, ok := unproject(0)
	far, farOK := unproject(1)
	if !ok || !farOK || far.Sub(near).Len() == 0 {
		return render.V3{}, render.V3{}, false
	}
	return near, far.Sub(near).Norm(), true
}

func rayBox(origin, direction, lo, hi render.V3) (float32, bool) {
	near, far := float32(0), float32(math.MaxFloat32)
	for axis := range 3 {
		if direction[axis] == 0 {
			if origin[axis] < lo[axis] || origin[axis] > hi[axis] {
				return 0, false
			}
			continue
		}
		a, b := (lo[axis]-origin[axis])/direction[axis], (hi[axis]-origin[axis])/direction[axis]
		if a > b {
			a, b = b, a
		}
		near, far = max(near, a), min(far, b)
		if near > far {
			return 0, false
		}
	}
	return near, true
}

// Grave accent cycles hostile targets in increasing distance, wrapping to the nearest.
// Pointer selection uses the camera from the last rendered frame.
func (a *App) pickMob(in *platform.Input, w, h int) uint32 {
	best, target := float32(math.MaxFloat32), uint32(0)
	nextDistance, nextTarget := float32(math.MaxFloat32), uint32(0)
	currentDistance := float32(-1)
	if current := a.state.Entities[a.state.Target]; current != nil && !current.Dead {
		dx, dz := current.X-a.wv.pos[0], current.Z-a.wv.pos[2]
		currentDistance = dx*dx + dz*dz
	}
	var vp render.M4
	var origin, direction render.V3
	var rayOK bool
	bestRay := float32(math.MaxFloat32)
	bestMob := false
	if in.Clicked {
		if a.scene3d == nil {
			return 0
		}
		vp = a.scene3d.View.Mul(a.scene3d.Proj)
		origin, direction, rayOK = clickRay(in.MouseX, in.MouseY, w, h, vp)
	}
	for id, e := range a.state.Entities {
		player := e.Kind == game.KindUser && e.Shape != nil && (in.Clicked || !e.Dead && !e.Concealed && !a.followOnly(e))
		monster := e.Kind == game.KindMob && !e.Dead && (in.Clicked || !a.friendlyMob(e))
		if e.ID == a.state.SelfID || !player && !monster {
			continue
		}
		if in.Clicked {
			if m := a.wv.mobs[id]; e.Kind == game.KindMob && m != nil && m.entity != e {
				continue
			}
			if p := a.wv.actors[id]; e.Kind == game.KindUser && p != nil && p.entity != e {
				continue
			}
		}
		dx, dz := e.X-a.wv.pos[0], e.Z-a.wv.pos[2]
		d := dx*dx + dz*dz
		if !in.Clicked && id != a.state.Target && (d > currentDistance || d == currentDistance && id > a.state.Target) && (d < nextDistance || d == nextDistance && id < nextTarget) {
			nextDistance, nextTarget = d, id
		}
		if in.Clicked {
			if rayOK {
				var distance float32
				var hit, tested bool
				var occlusionDistance float32
				if e.Kind == game.KindMob {
					if m := a.wv.mobs[id]; m != nil && m.actor != nil {
						distance, hit = a.monsterRayHit(m.actor, origin, direction)
						tested = true
						occlusionDistance = distance
						if hit {
							hit = a.wv.world.SegmentVisible(origin, origin.Add(direction.Scale(distance)))
							// Scene depth contains the actor itself, whose front can
							// be closer than the native selection plane.
							if lo, hi, valid := m.actor.WorldBounds(); valid {
								if front, ok := rayBox(origin, direction, lo, hi); ok {
									occlusionDistance = min(front, distance)
								}
							}
						}
					}
				} else if p := a.wv.actors[id]; p != nil && p.character != nil {
					coarseMiss := false
					if clo, chi, ok := p.character.CoarseBounds(); ok {
						if _, coarseHit := rayBox(origin, direction, clo, chi); !coarseHit {
							coarseMiss = true
						}
					}
					if !coarseMiss {
						if lo, hi, valid := p.character.WorldBounds(); valid {
							distance, hit = rayBox(origin, direction, lo, hi)
							tested = hit
							occlusionDistance = distance
						}
					}
				}
				if hit {
					_, _, depth, projected := projectLabel(origin.Add(direction.Scale(occlusionDistance)), vp, w, h)
					if projected && (a.sprites == nil || a.pickNameVisibleAt(in.MouseX, in.MouseY, w, h, depth)) && (monster && !bestMob || monster == bestMob && (distance < bestRay || distance == bestRay && (target == 0 || id < target))) {
						bestRay, target, bestMob = distance, id, monster
					}
				}
				if tested {
					continue
				}
			}
			pos := a.visualEntityPosition(e)
			if e.Kind == game.KindMob && a.wv.world != nil && a.wv.world.HasTerrain() {
				pos[1] = a.wv.world.Height(pos[0], pos[2])
			}
			pos[1]++
			x, sy, depth, ok := projectLabel(pos, vp, w, h)
			if !ok {
				continue
			}
			dx, dy := float32(x-in.MouseX), float32(sy-in.MouseY)
			d = dx*dx + dy*dy
			if d > 60*60 {
				continue
			}
			if a.sprites != nil && !a.pickNameVisibleAt(in.MouseX, in.MouseY, w, h, depth) {
				continue
			}
		}
		if bestRay == math.MaxFloat32 && (d < best || d == best && id < target) {
			best, target = d, id
		}
	}
	if !in.Clicked && nextTarget != 0 {
		return nextTarget
	}
	return target
}

// npcBillboardHit follows 0x43baf0. The half-width is 0.3 (1.0 for the dead NPC
// type and model 55, whose height is fixed at 3.5) and the rectangle always
// faces the camera horizontally.
func (a *App) npcBillboardHit(e *game.Entity, actor *scene.CreatureActor, origin, direction, heading render.V3) (float32, bool) {
	half, height := float32(.3), actor.NPCPlateAnchor(e.NpcType, e.TypeID)[1]-actor.World[13]
	if e.NpcType == 12 {
		half = 1
	}
	if a.npcDefs[sdata.NPCKey{Type: e.NpcType, ID: e.TypeID}].Model == 55 {
		half, height = 1, 3.5
	}
	forward := heading
	forward[1] = 0
	if forward.Len() < 1e-4 {
		return 0, false
	}
	forward = forward.Norm()
	denominator := direction[0]*forward[0] + direction[2]*forward[2]
	if math.Abs(float64(denominator)) < 1e-6 {
		return 0, false
	}
	feet := a.visualEntityPosition(e)
	t := ((feet[0]-origin[0])*forward[0] + (feet[2]-origin[2])*forward[2]) / denominator
	if t <= 0 {
		return 0, false
	}
	hit := origin.Add(direction.Scale(t))
	right := render.V3{forward[2], 0, -forward[0]}
	lateral := (hit[0]-feet[0])*right[0] + (hit[2]-feet[2])*right[2]
	vertical := hit[1] - feet[1]
	if lateral < -half || lateral > half || vertical < 0 || vertical > height {
		return 0, false
	}
	return t, true
}
