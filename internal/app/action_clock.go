package app

import "github.com/Spelunkern/Shaiya-GX/internal/scene"

// actionClock ties a confirmed visual/audio cue to one visible action. A
// replacement actor or action invalidates it, even if its clock reaches time.
type actionClock struct {
	actor  *scene.Character
	action uint64
	time   float32
}

func (c actionClock) ready(actor *scene.Character) (due, alive bool) {
	alive = actor != nil && actor == c.actor && actor.ActionID() == c.action
	return alive && actor.Time >= c.time, alive
}
