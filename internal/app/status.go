package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

type statusView struct {
	open, pending bool
	deadline      float64
}

func (a *App) toggleStatus() {
	a.status.open = !a.status.open
	if a.statusUI != nil {
		a.statusUI.Clear()
	}
	if a.status.open {
		a.subActionsOpen = false
		a.inventory.open = false
		a.skills.open = false
		a.skills.assign = nil
	}
}

func (a *App) statusState() ui.StatusState {
	c, _ := a.selfSlot()
	return ui.StatusState{Class: ui.ClassName(a.msg, a.state.Faction.Faction == 0, c.Class), Mode: a.msg.Get(470 + int(c.Mode)),
		Base: proto.StatPoints{c.Str, c.Dex, c.Rec, c.Int, c.Wis, c.Luc}, Points: a.state.Details.StatPoint, Details: a.state.Details,
		HP: a.state.Pools.HP, MP: a.state.Pools.MP, SP: a.state.Pools.SP, Stats: a.state.Stats, Known: a.state.StatsKnown, Busy: a.status.pending || a.state.Self.Dead}
}

func (a *App) updateStatus(in *platform.Input, w, h int) bool {
	for _, r := range in.Chars {
		if r == 't' || r == 'T' {
			a.toggleStatus()
			return true
		}
	}
	if !a.status.open || a.statusUI == nil {
		return false
	}
	x, y := a.statusUI.Origin(w, h)
	width := ui.StatusWidth
	if a.statusUI.Expanded {
		width += 132
	}
	captured := panelPointer(in, x, y, width, ui.StatusHeight)
	apply, close := a.statusUI.Update(in, w, h, a.statusState())
	if close || in.HasKey(platform.KeyEscape) {
		a.status.open = false
		a.statusUI.Clear()
		return true
	}
	if apply && !a.status.pending {
		d := a.statusUI.Draft
		if d.Total() > 0 && d.Total() <= uint32(a.state.Details.StatPoint) && !a.state.Self.Dead {
			if err := a.world.Send(proto.StatPointsPacket(d)); err != nil {
				a.backToLogin("Unable to assign attributes.")
				return true
			}
			a.status.pending, a.status.deadline = true, a.clock+10
		}
	}
	return captured
}

func (a *App) drawStatus(w, h int) {
	if a.status.open && a.statusUI != nil {
		x, y := a.statusUI.Origin(w, h)
		width := ui.StatusWidth
		if a.statusUI.Expanded {
			width += 132
		}
		a.clearTooltipUnderPanel(x, y, width, ui.StatusHeight)
		a.statusUI.Draw(a.sprites, a.font, w, h, a.statusState())
	}
}
