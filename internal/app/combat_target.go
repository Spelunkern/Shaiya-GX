package app

import (
	"math"

	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
)

// Delivery double-click gesture: same mob, <=350ms and <=6 screen pixels.
// Consuming the pair prevents a third click from repeating the request.
type mobClick struct {
	id   uint32
	at   float64
	x, y int
}

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
	if bestID != 0 && a.sprites.NameVisibleAt(in.MouseX, in.MouseY, w, h, bestDepth) {
		return bestID
	}
	return 0
}

func (c *mobClick) double(id uint32, x, y int, now float64) bool {
	dx, dy, elapsed := x-c.x, y-c.y, now-c.at
	double := id != 0 && c.id == id && elapsed >= 0 && elapsed <= .35 && dx*dx+dy*dy <= 36
	*c = mobClick{id, now, x, y}
	if double || id == 0 {
		*c = mobClick{}
	}
	return double
}

// pickNPC uses the displayed position while eligibility uses the confirmed
// position. Panel capture and ground-item handling occur before this test.
func (a *App) pickNPC(in *platform.Input, w, h int) uint32 {
	if !in.Clicked || a.scene3d == nil || a.wv == nil {
		return 0
	}
	vp := a.scene3d.View.Mul(a.scene3d.Proj)
	origin, direction, rayOK := clickRay(in.MouseX, in.MouseY, w, h, vp)
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
		dx, dz := e.X-a.wv.pos[0], e.Z-a.wv.pos[2]
		reach := float32(100)
		if e.NpcType == 2 {
			reach = 50 * 50
		}
		if dx*dx+dz*dz >= reach {
			continue
		}
		if lo, hi, valid := actor.actor.WorldBounds(); valid && rayOK {
			distance, hit := rayBox(origin, direction, lo, hi)
			if !hit {
				continue
			}
			_, _, depth, projected := projectLabel(origin.Add(direction.Scale(distance)), vp, w, h)
			if !projected || a.sprites != nil && !a.sprites.NameVisibleAt(in.MouseX, in.MouseY, w, h, depth) {
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
			var actor *scene.Character
			switch e.Kind {
			case game.KindMob:
				if m := a.wv.mobs[key]; m != nil && m.entity == e && m.actor != nil {
					actor = m.actor.Character
				}
			case game.KindUser:
				if p := a.wv.actors[key]; p != nil && p.entity == e {
					actor = p.character
				}
			}
			if actor == nil {
				continue
			}
			if lo, hi, valid := actor.WorldBounds(); valid {
				if d, hit := rayBox(origin, direction, lo, hi); hit && d < bestDistance {
					return 0
				}
			}
		}
	}
	return id
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

// Grave accent cycles monsters in increasing distance, wrapping to the nearest; pointer selection also supports players.
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
	if in.Clicked {
		if a.scene3d == nil {
			return 0
		}
		vp = a.scene3d.View.Mul(a.scene3d.Proj)
		origin, direction, rayOK = clickRay(in.MouseX, in.MouseY, w, h, vp)
	}
	for id, e := range a.state.Entities {
		if e.Dead || e.ID == a.state.SelfID || (e.Kind != game.KindMob && !(in.Clicked && e.Kind == game.KindUser && e.Shape != nil)) {
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
		if !in.Clicked && id != a.state.Target && d > currentDistance && (d < nextDistance || d == nextDistance && id < nextTarget) {
			nextDistance, nextTarget = d, id
		}
		if in.Clicked {
			var actor *scene.Character
			if e.Kind == game.KindMob {
				if m := a.wv.mobs[id]; m != nil && m.actor != nil {
					actor = m.actor.Character
				}
			} else if p := a.wv.actors[id]; p != nil {
				actor = p.character
			}
			if actor != nil && rayOK {
				if lo, hi, valid := actor.WorldBounds(); valid {
					if distance, hit := rayBox(origin, direction, lo, hi); hit {
						_, _, depth, projected := projectLabel(origin.Add(direction.Scale(distance)), vp, w, h)
						if projected && (a.sprites == nil || a.sprites.NameVisibleAt(in.MouseX, in.MouseY, w, h, depth)) && (distance < bestRay || distance == bestRay && (target == 0 || id < target)) {
							bestRay, target = distance, id
						}
						continue
					}
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
			if a.sprites != nil && !a.sprites.NameVisibleAt(in.MouseX, in.MouseY, w, h, depth) {
				continue
			}
			dx, dy := float32(x-in.MouseX), float32(sy-in.MouseY)
			d = dx*dx + dy*dy
			if d > 60*60 {
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
