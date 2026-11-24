package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

func (a *App) toggleInventory() {
	a.playUISound("ui_window.ogg")
	a.inventory.open = !a.inventory.open
	if a.inventory.open {
		a.skills.assign = nil
	} else {
		a.inventory.cancelItemPointer()
	}
}

// EP5.4 merchant 0x43e1c2 and warehouse 0x440369 align both panels at the
// same Y and put their edges together. GX anchors the inventory on the left.
func (a *App) inventoryDockOrigin(w, h int) (x, y int, active bool) {
	if a.state == nil {
		return 0, 0, false
	}
	switch {
	case a.warehouseOpen():
		x, y = a.bankUI.Origin(w, h)
	case a.merchantOpen():
		x, y = a.marketUI.Origin(w, h)
		x, y = a.shop.inventoryPosition.Origin(w, h, ui.InventoryWidth, ui.InventoryHeight, x-ui.InventoryWidth, y)
		return x, y, true
	default:
		return 0, 0, false
	}
	return max(0, x-ui.InventoryWidth), min(y, max(0, h-ui.InventoryHeight)), true
}

func (a *App) toggleSkills(assign bool) {
	a.playUISound("ui_window.ogg")
	v := &a.skills
	v.open = !v.open || v.assignMode != assign
	v.assignMode = assign
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
	switch action {
	case ui.BottomNone, ui.BottomSkills, ui.BottomStatus, ui.BottomGuild, ui.BottomQuests:
		// the window toggles play ui_window themselves
	default:
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
	return panelPointer(in, x, y, 213, 114) || a.targetFrameShown() && panelPointer(in, tx, ty, 256, 64)
}

func (a *App) personalPanelMask() uint8 {
	var mask uint8
	for i, open := range []bool{a.inventory.open, a.skills.open, a.subActionsOpen, a.status.open} {
		if open {
			mask |= 1 << i
		}
	}
	return mask
}

// Keep independently opened frames visible beside existing ones where space
// permits. Player-positioned frames retain their saved positions.
func (a *App) placeNewPersonalPanels(previous uint8, w, h int) {
	mask := a.personalPanelMask()
	if mask&^previous == 0 {
		return
	}
	type frame struct {
		position      *ui.WindowPosition
		origin        func(int, int) (int, int)
		width, height int
	}
	frames := make([]frame, 4)
	if a.inventoryUI != nil {
		frames[0] = frame{&a.inventoryUI.Position, a.inventoryUI.Origin, ui.InventoryWidth, ui.InventoryHeight}
	}
	if a.skillBook != nil {
		frames[1] = frame{&a.skillBook.Position, a.skillBook.Origin, ui.SkillBookWidth, ui.SkillBookHeight}
	}
	if a.subActionsUI != nil {
		frames[2] = frame{&a.subActionsUI.Position, a.subActionsUI.Origin, ui.SubActionWidth, ui.SubActionHeight}
	}
	if a.statusUI != nil {
		width := ui.StatusWidth
		if a.statusUI.Expanded {
			width += 132
		}
		frames[3] = frame{&a.statusUI.Position, a.statusUI.Origin, width, ui.StatusHeight}
	}
	_, _, inventoryDocked := a.inventoryDockOrigin(w, h)
	for i, f := range frames {
		if mask&^previous&(1<<i) == 0 || f.position == nil || f.position.Moved || i == 0 && inventoryDocked {
			continue
		}
		x, y := f.origin(w, h)
		candidates := []int{x}
		for j, other := range frames {
			if j == i || mask&(1<<j) == 0 || other.position == nil {
				continue
			}
			ox, _ := other.origin(w, h)
			candidates = append(candidates, ox-f.width-8, ox+other.width+8)
		}
		for _, cx := range candidates {
			if cx < 0 || cx+f.width > w {
				continue
			}
			free := true
			for j, other := range frames {
				if j == i || mask&(1<<j) == 0 || other.position == nil {
					continue
				}
				ox, oy := other.origin(w, h)
				if cx < ox+other.width && cx+f.width > ox && y < oy+other.height && y+f.height > oy {
					free = false
					break
				}
			}
			if free {
				if cx != x {
					f.position.Move(cx, y, w, h, f.width, f.height)
				}
				break
			}
		}
	}
}
