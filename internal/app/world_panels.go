package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

func (a *App) toggleInventory() {
	a.inventory.open = !a.inventory.open
	if a.inventory.open {
		a.subActionsOpen = false
		a.status.open = false
		a.skills.open = false
		a.skills.assign = nil
	}
}

func (a *App) toggleSkills(assign bool) {
	v := &a.skills
	v.open = !v.open || v.assignMode != assign
	v.assignMode = assign
	if v.open {
		a.subActionsOpen = false
		a.status.open = false
		a.inventory.open = false
	}
	v.offset, v.assign = 0, nil
}

func (a *App) updateBottom(in *platform.Input, w, h int) bool {
	if a.bottom == nil {
		return false
	}
	a.bottom.QuestOpen, a.bottom.GuildOpen = a.questLog.open, a.guild.open
	action, captured := a.bottom.Update(in, w, h, a.inventory.open, a.skills.open, a.status.open, a.subActionsOpen)
	switch action {
	case ui.BottomInventory:
		a.toggleInventory()
	case ui.BottomSkills:
		a.toggleSkills(false)
	case ui.BottomSubActions:
		a.toggleSubActions()
	case ui.BottomStatus:
		a.toggleStatus()
	case ui.BottomGuild:
		a.toggleGuild()
	case ui.BottomQuests:
		a.toggleQuestLog()
	case ui.BottomMall:
		a.openMall()
	case ui.BottomOptions:
		a.openGameMenu()
	}
	if action != ui.BottomNone {
		a.playUI()
	}
	return captured
}

func (a *App) updateHUDPointer(in *platform.Input, w, h int) bool {
	x, y := 8, 8
	tx, ty := 256, 0
	if a.hud != nil {
		x, y = a.hud.Origin(w, h)
		tx, ty = a.hud.TargetOrigin(w, h)
	}
	// Native status window 0x509f10; target window 0x50a2e0.
	if in.Clicked && in.MouseX >= x+17 && in.MouseX < x+72 && in.MouseY >= y+26 && in.MouseY < y+79 {
		a.selectSelf()
	}
	return panelPointer(in, x, y, 213, 114) || a.state.Target != 0 && panelPointer(in, tx, ty, 256, 64)
}
