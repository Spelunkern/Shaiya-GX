package app

import (
	"fmt"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

// Panel capture is limited to pointer activity inside its visible rectangle.
// Idle hover does not consume hotkeys or held movement keys.
func panelPointer(in *platform.Input, x, y, w, h int) bool {
	return in.MouseX >= x && in.MouseX < x+w && in.MouseY >= y && in.MouseY < y+h &&
		(in.Pressed || in.Clicked || in.MouseDown || in.RightPressed || in.RightDown || in.Wheel != 0)
}

func (a *App) checkPendingActions() bool {
	var sitPending bool
	var sitDeadline float64
	if a.wv != nil {
		sitPending, sitDeadline = a.wv.sitPending, a.wv.sitDeadline
	}
	for _, p := range []struct {
		pending  bool
		deadline float64
		name     string
	}{
		{a.inventory.pending, a.inventory.deadline, "Inventory"},
		{a.vehicle.pending, a.vehicle.deadline, "Mount"},
		{a.exchange.mutation != 0, a.exchange.deadline, "Trade"},
		{a.duel.mutation != 0, a.duel.deadline, "Duel"},
		{a.guild.createPending, a.guild.createDeadline, "Guild creation"},
		{a.guild.lifecyclePending, a.guild.lifecycleDeadline, "Guild request"},
		{a.shop.pending, a.shop.deadline, "Shop"},
		{a.skills.pending, a.skills.deadline, "Skill"},
		{a.status.pending, a.status.deadline, "Attributes"},
		{sitPending, sitDeadline, "Sitting"},
	} {
		if p.pending && a.clock >= p.deadline {
			a.backToLogin(fmt.Sprintf("%s response timed out. Reconnect to refresh state.", p.name))
			return false
		}
	}
	return true
}

// Right-side panels are mutually exclusive in the delivery layout. A shop
// can stay open beside them. Captured events never reach a world target or
// logout, while the original held-key callback keeps movement available.
func (a *App) worldControls(in *platform.Input, w, h int) platform.Input {
	controls := *in
	v := a.wv
	if a.socialInviteUI != nil && a.socialInviteUI.Visible {
		a.updateSocialInvitation(in, w, h)
		return platform.Input{MouseX: in.MouseX, MouseY: in.MouseY, SuppressAutoRun: true}
	}
	if a.menuOpen {
		var captured bool
		controls, captured = a.gameMenuControls(in, w, h)
		if captured {
			return controls
		}
		in = &controls
	}
	if a.appearanceEdit != nil {
		a.updateAppearanceEditor(in, w, h)
		return platform.Input{MouseX: in.MouseX, MouseY: in.MouseY, SuppressAutoRun: true}
	}
	if a.gateConfirm != nil && a.gateConfirm.Visible {
		a.updateGateConfirm(in, w, h)
		return platform.Input{MouseX: in.MouseX, MouseY: in.MouseY}
	}
	if a.blacksmith.quantity {
		a.updateEnchantQuantity(in)
		return platform.Input{MouseX: in.MouseX, MouseY: in.MouseY}
	}
	if a.gemConfirm != nil && a.gemConfirm.Visible {
		a.updateGemConfirm(in, w, h)
		return platform.Input{MouseX: in.MouseX, MouseY: in.MouseY}
	}
	if a.duel.quantity != nil || a.duel.moneyOpen {
		a.updateDuelModal(in)
		return platform.Input{MouseX: in.MouseX, MouseY: in.MouseY}
	}
	if a.exchange.quantity != nil || a.exchange.moneyOpen {
		a.updateExchangeModal(in)
		return platform.Input{MouseX: in.MouseX, MouseY: in.MouseY}
	}
	if a.shop.quantity != nil {
		v.mobClick = mobClick{}
		if in.Pressed {
			v.uiLeft = 5
		}
		if in.Clicked || !in.MouseDown {
			v.uiLeft = 0
		}
		a.updateTradeQuantity(in)
		return platform.Input{MouseX: in.MouseX, MouseY: in.MouseY}
	}
	// A right drag belongs to its press owner until release. Virtual cursor
	// movement or a panel opened by a hotkey cannot transfer it to the UI.
	cameraDrag := v.mouselook && in.RightDown && !in.RightPressed && !v.uiRight
	if cameraDrag {
		pointer := *in
		pointer.RightDown = false
		pointer.Wheel = 0
		in = &pointer
	}
	captured := false
	if a.chatUI != nil && a.chatUI.ModeOpen {
		captured = a.updateChatPointer(in, w, h)
		if captured && in.Pressed {
			v.uiLeft = -3
		}
	}
	if !captured {
		captured = a.updateQuickPotionDrop(in, w, h)
		if !captured {
			captured = a.updateShortcutDrag(in, w, h)
		}
	}
	if a.state == nil || a.wv == nil {
		return platform.Input{SuppressAutoRun: true}
	}
	underMall := false
	if a.mall.open && a.mallUI != nil {
		x, y := a.mallUI.Origin(w, h)
		underMall = pointInside(in, x, y, ui.MallWidth, ui.MallHeight)
		if a.mallUI.GiftOpen {
			x, y = a.mallUI.GiftOrigin(w, h)
			underMall = underMall || pointInside(in, x, y, ui.GiftBoxWidth, ui.GiftBoxHeight)
		}
	}
	if !underMall && !captured {
		captured = a.updatePlayerContext(in, w, h)
	}
	if !captured {
		captured = a.updateDuelDrag(in, w, h)
	}
	if !captured {
		captured = a.updateExchangeDrag(in, w, h)
	}
	if !captured {
		captured = a.updateLocalActionDrag(in, w, h)
	}
	if !captured {
		captured = a.updateWindowDrag(in, w, h)
	}
	if !captured {
		captured = a.updateBlacksmithDrag(in, w, h)
	}
	if !captured {
		captured = a.updateWarehouseDrag(in, w, h)
	}
	if !captured {
		captured = a.updateCommerceDrag(in, w, h)
	}
	if a.wv.mapView.full && !captured {
		captured = a.updateMinimap(in, w, h)
		if captured && in.Pressed {
			v.uiLeft = -1 // The topmost full map owns release, even outside its frame.
		}
	}

	for index, update := range []func(*platform.Input, int, int) bool{a.updateMall, a.updateGuild, a.updateQuickPotions, a.updateBarEditing, a.updateSubActions, a.updateSkills, a.updateDuel, a.updateExchange, a.updateQuestLog, a.updateInventory, a.updateShop, a.updateBottom, a.updateHUDPointer, a.updateChatPointer, a.updateStatus, a.updateBuffs, a.updateMinimap, a.updatePartyPointer} {
		if captured {
			break
		}
		input := *in
		if in.Clicked && !in.Pressed && v.uiLeft != index+1 {
			// Keep the release as a hit-test event, but only its original
			// owner may activate a control. This also blocks click-through.
			input.Clicked = false
			input.MouseDown = true
		}
		if update(&input, w, h) {
			captured = true
			if in.Pressed {
				v.uiLeft = index + 1
			}
			break
		}
	}
	if in.Pressed && !captured {
		v.uiLeft = 0
	}
	if in.RightPressed {
		v.uiRight = captured
	}
	pointerOwned := v.uiLeft != 0 || v.uiRight
	if in.Clicked || !in.MouseDown {
		v.uiLeft = 0
	}
	if !in.RightDown {
		v.uiRight = false
	}
	if captured {
		controls.Chars = nil
		controls.Actions = nil
		controls.Keys = nil
		controls.Released = ""
	}
	if captured || pointerOwned {
		if in.Pressed || in.Clicked {
			v.mobClick = mobClick{}
		}
		controls.Pressed = false
		controls.Clicked = false
		controls.MouseDown = false
		controls.RightPressed = false
		controls.RightDown = false
		controls.Wheel = 0
		controls.MiddlePressed = false
	}
	if a.inventory.dragging && (!in.MouseDown || !a.inventory.open) {
		// The release can be captured by another panel before inventory runs.
		a.inventory.dragging, a.inventory.selected = false, false
	}
	if cameraDrag {
		controls.RightDown = true
	}
	if controls.HasKey(platform.KeyEscape) || hasChar(&controls, 'o') {
		a.openGameMenu()
		controls.Keys, controls.Chars = nil, nil
	}
	return controls
}
