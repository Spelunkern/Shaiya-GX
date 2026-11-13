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

func (a *App) updateHUDPointer(in *platform.Input, _, _ int) bool {
	// Native status window 0x509f10; target window 0x50a2e0.
	if in.Clicked && in.MouseX >= 25 && in.MouseX < 80 && in.MouseY >= 34 && in.MouseY < 87 {
		a.selectSelf()
	}
	return panelPointer(in, 8, 8, 213, 114) || a.state.Target != 0 && panelPointer(in, 256, 0, 256, 64)
}
