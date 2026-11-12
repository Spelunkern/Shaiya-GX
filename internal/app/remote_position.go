package app

import (
	"math"

	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

// remotePosition smooths only presentation. Entity coordinates remain the
// server snapshot for gameplay; large jumps and explicit respawns snap.
type remotePosition struct {
	initialized                  bool
	pos, from, to                render.V3
	moveSerial, resetSerial      uint64
	start, duration, lastArrival float64
}

func (r *remotePosition) update(e *game.Entity, now float64) render.V3 {
	target := render.V3{e.X, e.Y, e.Z}
	if !r.initialized {
		r.initialized = true
		r.pos, r.from, r.to = target, target, target
		r.moveSerial, r.resetSerial = e.MoveSerial, e.PositionResetSerial
		r.lastArrival = now
		return r.pos
	}
	if r.duration > 0 {
		t := float32(min(max((now-r.start)/r.duration, 0), 1))
		r.pos = r.from.Add(r.to.Sub(r.from).Scale(t))
	}
	if e.PositionResetSerial != r.resetSerial {
		r.pos, r.from, r.to = target, target, target
		r.duration = 0
		r.resetSerial, r.moveSerial = e.PositionResetSerial, e.MoveSerial
		r.lastArrival = now
		return r.pos
	}
	if e.MoveSerial != r.moveSerial {
		delta := target.Sub(r.pos)
		if delta.Dot(delta) > 400 || math.Abs(float64(delta[1])) > 8 {
			r.pos, r.from, r.to = target, target, target
			r.duration = 0
		} else {
			r.from, r.to = r.pos, target
			r.start = now
			r.duration = min(max(now-r.lastArrival, .1), .5)
		}
		r.moveSerial = e.MoveSerial
		r.lastArrival = now
	}
	return r.pos
}

// visualEntityPosition keeps picking and effects attached to the rendered actor.
func (a *App) visualEntityPosition(e *game.Entity) render.V3 {
	if a.wv != nil {
		if e.Kind == game.KindUser {
			if actor := a.wv.actors[e.ID]; actor != nil && actor.entity == e && actor.position.initialized {
				return actor.position.pos
			}
		} else if e.Kind == game.KindMob {
			if actor := a.wv.mobs[e.ID]; actor != nil && actor.entity == e && actor.position.initialized {
				return actor.pos
			}
		} else if e.Kind == game.KindNpc {
			if actor := a.wv.npcs[e.ID]; actor != nil && actor.entity == e && actor.position.initialized {
				return actor.position.pos
			}
		}
	}
	return render.V3{e.X, e.Y, e.Z}
}
