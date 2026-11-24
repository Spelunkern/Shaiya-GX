package app

import (
	"fmt"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
	"strings"
)

type targetEffectsView struct {
	requestedID   uint32
	requestedKind uint8
	serial        uint64
	rows          []buffBadge
}

func (a *App) updateTargetEffects() {
	if a.world == nil {
		return
	}
	v := &a.targetEffects
	id := a.state.Target
	kind := uint8(0)
	if id == a.state.SelfID && id != 0 {
		kind = proto.TargetEffectUser
	} else if e := a.state.Entities[id]; e != nil && !e.Dead {
		if e.Kind == game.KindMob {
			kind = proto.TargetEffectMob
		} else if e.Kind == game.KindUser {
			kind = proto.TargetEffectUser
		}
	}
	if kind == 0 {
		id = 0
	}
	if id != v.requestedID || kind != v.requestedKind {
		if err := a.world.Send(proto.TargetEffectsQuery(kind, id)); err != nil {
			a.backToLoginID(12196)
			return
		}
		v.requestedID, v.requestedKind = id, kind
		v.rows = nil
		v.serial = 0
	}
	if id == 0 || a.state.TargetEffects.ID != id || a.state.TargetEffects.Kind != kind || v.serial == a.state.TargetEffectsSerial {
		return
	}
	v.serial = a.state.TargetEffectsSerial
	v.rows = v.rows[:0]
	for _, e := range a.state.TargetEffects.Effects {
		d, known := a.skillDefinition(e.Skill, e.Level)
		if !known {
			continue
		}
		seconds := e.Remaining
		if seconds == 0 && !e.HasRemaining {
			seconds = d.KeepSeconds()
		}
		v.rows = append(v.rows, buffBadge{buff: proto.ActiveBuff{Skill: e.Skill, Level: e.Level}, def: d, until: a.clock + float64(seconds), timed: seconds > 0, duration: max(seconds, d.KeepSeconds())})
	}
}
func (a *App) targetEffectSlot(w, h, index int) (int, int) {
	x, y := a.hud.TargetOrigin(w, h)
	return x + 40 + (index%8)*18, y + 62 + (index/8)*18
}
func (a *App) targetEffectHit(w, h, mx, my int) int {
	if a.state.Target != a.targetEffects.requestedID {
		return -1
	}
	for i := range a.targetEffects.rows {
		x, y := a.targetEffectSlot(w, h, i)
		if y < h && mx >= x && mx < x+16 && my >= y && my < y+16 {
			return i
		}
	}
	return -1
}
func (a *App) drawTargetEffects(w, h int) {
	if a.state.Target != a.targetEffects.requestedID {
		return
	}
	for i, b := range a.targetEffects.rows {
		x, y := a.targetEffectSlot(w, h, i)
		if y+16 > h {
			break
		}
		if file, uv, ok := ui.SkillIcon(b.def.Icon); ok {
			if tex := a.assets.Texture("interface/" + file); tex != nil {
				a.sprites.Draw(tex, float32(x), float32(y), 16, 16, uv.U0, uv.V0, uv.U1, uv.V1, white)
			}
		}
		if buffGroup(b) == 1 {
			a.sprites.Draw(a.white, float32(x), float32(y), 16, 16, 0, 0, 1, 1, [4]float32{1, 1, 1, debuffFlash(a.clock)})
		}
	}
}
func (a *App) drawTargetEffectTooltip(w, h int) {
	i := a.targetEffectHit(w, h, a.lastMouseX, a.lastMouseY)
	if i < 0 {
		return
	}
	b := a.targetEffects.rows[i]
	name := b.def.Name
	if name == "" {
		name = a.msg.Format(12321, map[string]string{"v": fmt.Sprint(b.buff.Skill)})
	}
	a.setTooltip(tooltipContent{
		tooltip:            []itemDetailLine{{name, [4]float32{1, 1, 0, 1}}, {ui.BuffDuration(b.seconds(a.clock), b.timed), white}},
		tooltipDescription: strings.ReplaceAll(b.def.Description, `\n`, "\n"),
	})
}
