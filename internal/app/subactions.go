package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

func (a *App) toggleSubActions() {
	a.subActionsOpen = !a.subActionsOpen
	if a.subActionsOpen {
		a.inventory.open = false
		a.skills.open = false
		a.skills.assign = nil
		a.status.open = false
	}
}
func (a *App) updateSubActions(in *platform.Input, w, h int) bool {
	for _, r := range in.Chars {
		if r == 'b' || r == 'B' {
			a.toggleSubActions()
			return true
		}
	}
	if !a.subActionsOpen || a.subActionsUI == nil {
		return false
	}
	if in.HasKey(platform.KeyEscape) {
		a.subActionsOpen = false
		return true
	}
	selected, captured := a.subActionsUI.Update(in, w, h)
	if in.Pressed && a.wv != nil && a.state != nil && !a.state.Self.Dead {
		if cell := a.subActionsUI.CellAt(in.MouseX, in.MouseY, w, h); cell >= 0 {
			a.wv.localActionDrag = subActionDrag{tracking: true, source: cell, x: in.MouseX, y: in.MouseY}
		}
	}
	if selected == ui.SubActionClose {
		a.subActionsOpen = false
		return true
	}
	if selected < 0 || a.state == nil || a.wv == nil || a.world == nil {
		return captured
	}
	a.activateSubAction(selected)
	return captured
}

func (a *App) activateSubAction(selected int) {
	if a.state == nil || a.state.Self.Dead || a.wv == nil || a.world == nil {
		return
	}
	switch selected {
	case 0:
		a.updateWorld(&platform.Input{Chars: []rune{'c'}}, 0)
	case 1:
		a.wv.walking = !a.wv.walking
	case 2:
		a.wv.autoRun = !a.wv.autoRun
	case 3:
		a.pickupNearest()
	case 4:
		a.autoAttackNearest()
	default:
		if selected >= 5 && selected <= 14 {
			a.sendGesture(uint8(selected + 111))
		}
	}
	if a.state != nil {
		a.playUI()
	}
}
