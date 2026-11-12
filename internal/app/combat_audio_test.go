package app

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/scene"
)

func TestAttackAudioFollowsActionAndCancelsStaleActor(t *testing.T) {
	actor := &scene.Character{}
	cue := attackSound{actionClock: actionClock{actor, actor.ActionID(), .4}}
	actor.Time = .39
	if due, alive := cue.ready(actor); due || !alive {
		t.Fatal("weapon cue fired before its contact")
	}
	actor.Time = .4
	if due, alive := cue.ready(actor); !due || !alive {
		t.Fatal("weapon cue missed its contact")
	}
	for _, other := range []*scene.Character{nil, {Time: 1}} {
		if due, alive := cue.ready(other); due || alive {
			t.Fatal("removed/replaced actor kept its old cue")
		}
	}
	actor.ReleaseAction()
	if due, alive := cue.ready(actor); due || alive {
		t.Fatal("interrupted action kept its old cue")
	}
}
