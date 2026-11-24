package app

import (
	"fmt"
	"math"
	"strings"

	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

type buffBadge struct {
	buff     proto.ActiveBuff
	def      sdata.Skill
	until    float64
	duration uint32
	timed    bool
}

type buffsView struct {
	rows []buffBadge
}

func (b buffBadge) seconds(now float64) uint32 {
	return uint32(max(0, math.Ceil(b.until-now)))
}

// Only authoritative application/snapshot packets start display timers. A
// refresh keeps its instance slot; elapsed timers do not mutate game.State.
func (a *App) confirmedBuffs(op proto.Opcode) {
	if op != proto.CharacterActiveBuffs && op != proto.BuffAdd && op != proto.BuffRemove {
		return
	}
	a.loadSkills()
	old := make(map[uint32]buffBadge, len(a.buffs.rows))
	for _, b := range a.buffs.rows {
		old[b.buff.Instance] = b
	}
	rows := make([]buffBadge, 0, len(a.state.Buffs))
	for _, b := range a.state.Buffs {
		badge, exists := old[b.Instance]
		if !exists || op == proto.CharacterActiveBuffs || op == proto.BuffAdd && b.Instance == a.state.LastBuff.Instance {
			def, _ := a.skillDefinition(b.Skill, b.Level)
			seconds := def.KeepSeconds()
			if b.HasRemaining {
				seconds = b.Remaining
			}
			badge = buffBadge{buff: b, def: def, until: a.clock + float64(seconds), timed: b.HasRemaining || seconds > 0, duration: max(seconds, def.KeepSeconds())}
		}
		rows = append(rows, badge)
	}
	a.buffs.rows = rows
	if op == proto.BuffAdd && a.state.LastBuff.Skill == warehouseCallSkill {
		a.openRemoteWarehouse() // using the rune opens the warehouse at once
	}
}

func (a *App) buffCancelPacket(index int) []byte {
	if index < 0 || index >= len(a.buffs.rows) || (a.buffs.rows[index].def.Effect != 4 || a.buffs.rows[index].buff.Level == 100 || a.buffs.rows[index].buff.Skill == warehouseCallSkill) {
		return nil // Warehouse Call is never cancelled by hand
	}
	// Rapture friendly.go cancelBuff accepts only own TypeEffect=4 and SkillID.
	return proto.CancelBuffPacket(a.buffs.rows[index].buff.Skill)
}

func (a *App) updateBuffs(in *platform.Input, w, h int) bool {
	if i := a.targetEffectHit(w, h, in.MouseX, in.MouseY); i >= 0 {
		x, y := a.targetEffectSlot(w, h, i)
		return panelPointer(in, x, y, 16, 16)
	}
	i := a.buffHit(w, h, in.MouseX, in.MouseY)
	if i < 0 {
		return false
	}
	if in.Clicked && a.buffs.rows[i].buff.Skill == warehouseCallSkill {
		a.openRemoteWarehouse() // the buff reopens the window
	}
	if in.RightPressed && a.world != nil {
		if p := a.buffCancelPacket(i); p != nil {
			if err := a.world.Send(p); err != nil {
				a.backToLoginID(12014)
			}
		}
	}
	x, y := a.buffSlot(w, h, i)
	return panelPointer(in, x, y, 32, 32)
}

func (a *App) drawBuffs(w, h int) {
	for i, b := range a.buffs.rows {
		x, y := a.buffSlot(w, h, i)
		if y >= h {
			break
		}
		if file, uv, ok := ui.SkillIcon(b.def.Icon); ok {
			if tex := a.assets.Texture("interface/" + file); tex != nil {
				a.sprites.Draw(tex, float32(x), float32(y), 32, 32, uv.U0, uv.V0, uv.U1, uv.V1, white)
			}
		} else {
			a.sprites.Draw(a.white, float32(x), float32(y), 32, 32, 0, 0, 1, 1, gray)
		}
		ui.DrawSkillLevel(a.sprites, a.font, x, y, b.buff.Level)
		if buffGroup(b) == 1 {
			a.sprites.Draw(a.white, float32(x), float32(y), 32, 32, 0, 0, 1, 1, [4]float32{1, 1, 1, debuffFlash(a.clock)})
		}
		if b.timed && b.duration > 0 {
			a.sprites.Draw(a.white, float32(x), float32(y+32), 32, 5, 0, 0, 1, 1, [4]float32{0, 0, 0, 1})
			ratio := min(float32(1), float32(max(0, b.until-a.clock))/float32(b.duration))
			a.sprites.Draw(a.white, float32(x+1), float32(y+33), 30*ratio, 3, 0, 0, 1, 1, [4]float32{1, 1, 0, 1})
		}
	}
}

func (a *App) drawBuffTooltip(w, h int) {
	a.drawTargetEffectTooltip(w, h)
	i := a.buffHit(w, h, a.lastMouseX, a.lastMouseY)
	if i < 0 {
		return
	}
	b := a.buffs.rows[i]
	name := b.def.Name
	if name == "" {
		name = a.msg.Format(12295, map[string]string{"v": fmt.Sprint(b.buff.Skill), "v2": fmt.Sprint(b.buff.Level)})
	}
	a.setTooltip(tooltipContent{
		tooltip:            []itemDetailLine{{name, [4]float32{1, 1, 0, 1}}},
		tooltipDescription: strings.ReplaceAll(b.def.Description, `\n`, "\n"),
	})

	if b.buff.Skill == warehouseCallSkill {
		a.tooltipDescription = warehouseCallDescription
	}

	if b.timed {
		if remaining := ui.BuffRemainingText(uint32(max(0, b.until-a.clock))); remaining != "" {
			if a.tooltipDescription != "" {
				a.tooltipDescription += "\n"
			}
			a.tooltipDescription += remaining
		}
	}
}

func buffGroup(b buffBadge) int {
	if b.def.Effect == 3 || b.buff.Level == 100 {
		return 1
	}
	return 0
}
func (a *App) buffCounts() (counts [2]int) {
	for _, b := range a.buffs.rows {
		counts[buffGroup(b)]++
	}
	return
}
func (a *App) buffSlot(w, h, index int) (int, int) {
	counts := a.buffCounts()
	group := buffGroup(a.buffs.rows[index])
	slot := 0
	for i := 0; i < index; i++ {
		if buffGroup(a.buffs.rows[i]) == group {
			slot++
		}
	}
	return a.buffLayout.Slot(w, h, group, slot, counts[group])
}
func (a *App) buffHit(w, h, mx, my int) int {
	for i := range a.buffs.rows {
		x, y := a.buffSlot(w, h, i)
		if mx >= x && mx < x+32 && my >= y && my < y+37 {
			return i
		}
	}
	return -1
}

// Native 0x4c0267..0x4c02c4 folds phase at .5 before multiplying by 255:
// a quantized 0 -> 127 -> 0 white pulse, not an opaque sawtooth flash.
func debuffFlash(now float64) float32 {
	phase := now - math.Floor(now)
	return float32(math.Floor((.5-math.Abs(.5-phase))*255)) / 255
}
