package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
)

type attackSound struct {
	actionClock
	source, entity uint32
	file           string
	pos            render.V3
}

func (a *App) queueAttackSounds(r proto.AttackResult, weapon uint8, actor *scene.Character) {
	if a.sound == nil || len(a.wv.attackSounds)+2 > 64 || (r.Result != 0 && r.Result != 1 && r.Result != 3) {
		return
	}
	swing, hit := scene.WeaponSounds(weapon, r.Result == 1, a.attackVariant)
	queue := func(file string, entity uint32, time float32) {
		pos, ok := a.effectPosition(entity)
		if file != "" && ok {
			a.wv.attackSounds = append(a.wv.attackSounds, attackSound{actionClock{actor, actor.ActionID(), time}, r.Source, entity, file, pos})
		}
	}
	queue(swing, r.Source, scene.WeaponSwingTime(weapon))
	if contact := actor.ContactTime(); contact > 0 && r.Result <= 1 && r.HP > 0 {
		queue(hit, r.Target, contact)
	}
}

func (a *App) updateAttackSounds() {
	if a.wv == nil {
		return
	}
	kept := a.wv.attackSounds[:0]
	for _, cue := range a.wv.attackSounds {
		due, alive := cue.ready(a.playerActor(cue.source))
		if !alive {
			continue
		}
		if !due {
			kept = append(kept, cue)
			continue
		}
		if pos, ok := a.effectPosition(cue.entity); ok {
			cue.pos = pos
		}
		a.playSound(cue.file, cue.pos[0], cue.pos[2])
	}
	clear(a.wv.attackSounds[len(kept):])
	a.wv.attackSounds = kept
}
