package app

import (
	"log"
	"math"
	"path"
	"strings"

	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/data/world"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
)

func (a *App) audioError(file string, err error) {
	if err != nil && !a.soundErrors[file] {
		log.Printf("audio %s: %v", file, err)
		a.soundErrors[file] = true
	}
}
func (a *App) playSound(name string, x, z float32) {
	if a.sound == nil || name == "" || strings.EqualFold(name, "LOAD") {
		return
	}
	gain := 1.0
	if a.wv != nil {
		dx, dz := float64(x-a.wv.pos[0]), float64(z-a.wv.pos[2])
		gain = math.Max(0, 1-math.Hypot(dx, dz)/40)
	}
	if gain == 0 {
		return
	}
	file := "Sound/" + path.Base(strings.ReplaceAll(name, `\`, "/"))
	a.audioError(file, a.sound.Play(file, gain))
}
func mapMusic(w *world.WLD, x, z float32) string {
	if w == nil {
		return ""
	}
	for _, zone := range w.MusicZones {
		if x < zone.Box.Min[0] || x > zone.Box.Max[0] || z < zone.Box.Min[2] || z > zone.Box.Max[2] {
			continue
		}
		if zone.Asset >= 0 && int(zone.Asset) < len(w.MusicAssets) {
			return w.MusicAssets[zone.Asset]
		}
	}
	return ""
}
func (a *App) updateAudio() {
	if a.sound == nil {
		return
	}
	a.updateAttackSounds()
	a.audioError("device", a.sound.Update())
	music := ""
	switch a.screen {
	case screenLogin, screenServers:
		music = "bgm_login001.wav"
	case screenWorld:
		if a.wv != nil && a.wv.world != nil {
			music = mapMusic(a.wv.world.WLD, a.wv.pos[0], a.wv.pos[2])
		}
	case screenCharacters:
		key := "select_a"
		if a.state != nil && a.state.Faction.Faction == 1 {
			key = "select_b"
		}
		if a.creating {
			key = "creation"
		}
		if w := a.worlds[key]; w != nil {
			x, z := selectActorPos[0], selectActorPos[2]
			if a.creating {
				x, z = 68, 66
			}
			music = mapMusic(w.WLD, x, z)
		}
	}
	if music != "" {
		music = "Sound/Music/" + path.Base(strings.ReplaceAll(music, `\`, "/"))
	}
	a.audioError(music, a.sound.Music(music))
}
func (a *App) confirmedSound(p []byte) {
	if a.sound == nil || a.wv == nil {
		return
	}
	switch proto.Op(p) {
	case proto.SkillCastStarted:
		r := a.state.LastCast
		a.playSkillSound(r.Source, r.ID, r.Level, true)
	case proto.SkillUse, proto.SkillUseNoTarget, proto.SkillUserDrain, proto.SkillMobDrain:
		r := a.state.LastSkill
		if r.Result <= 3 && !(r.Result == 3 && r.Target == 0) {
			a.playSkillSound(r.Source, r.ID, r.Level, false)
		}
	case proto.MobAttackCharacter:
		id := a.state.LastAttack.Source
		if m := a.wv.mobs[id]; m != nil && m.actor != nil {
			a.playSound(m.actor.Sound(0), m.pos[0], m.pos[2])
		}
	case proto.MobDeath:
		id := proto.NewReader(p).U32()
		if m := a.wv.mobs[id]; m != nil && m.actor != nil {
			a.playSound(m.actor.Sound(3), m.pos[0], m.pos[2])
		}
	}
}

func (a *App) playSkillSound(source uint32, id uint16, level uint8, preparing bool) {
	pos, visible := a.effectPosition(source)
	if !visible || level == 0 || level == 100 {
		return
	}
	a.loadSkills()
	d, ok := a.skillDefs[sdata.SkillKey{ID: id, Level: level}]
	if !ok || d.Effect == 0 {
		return
	}
	action := scene.SkillCastAction(d.Animation, d.Effect)
	if !preparing {
		// Area announcements/results need their own deduplication and dispatch.
		if d.Target < 1 || d.Target > 3 {
			return
		}
		action = scene.SkillAction(d.Animation, d.Attack, d.Effect, 0, 0)
	}
	a.playSound(scene.SkillActionSound(action), pos[0], pos[2])
}

func (a *App) playUI() {
	if a.sound != nil {
		a.audioError("Sound/ui_button001.wav", a.sound.Play("Sound/ui_button001.wav", 0.4))
	}
}
