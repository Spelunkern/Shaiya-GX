package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

var (
	white = [4]float32{1, 1, 1, 1}
	gray  = [4]float32{.8, .8, .8, 1}
)

func (a *App) updateCharacters(in *platform.Input, w, h int, dt float64) {
	if a.state == nil {
		return
	}
	a.clock += dt
	if a.account.op != 0 && a.clock >= a.account.deadline {
		a.backToLoginID(12015)
		return
	}
	if a.updateAccountMessage(in, w, h) {
		return
	}
	if a.state.Phase == game.PhaseCharSelect && !a.creating {
		if a.menuOpen {
			// The options window is not modal here: pointer input outside it
			// still reaches the roster buttons.
			if a.gameMenuOwnsInput(in, w, h) {
				a.updateGameMenu(in, w, h)
				return
			}
		} else if hasChar(in, 'o') || in.HasKey(platform.KeyEscape) {
			a.openGameMenu()
			return
		}
	}
	if a.world == nil && !a.creating { // offscreen preview
		if a.chars.Update(in, w, h, dt) == ui.CharOptions {
			a.openGameMenu()
		}
		return
	}
	switch a.state.Phase {
	case game.PhaseCharSelect:
		a.screen = screenCharacters
		if a.creating {
			if a.create.Busy && a.clock >= a.createDeadline {
				a.backToLoginID(12016)
				return
			}
			switch a.create.Update(in, w, h) {
			case ui.CharCreate:
				if a.world == nil {
					return
				}
				a.create.Busy = true
				a.createDeadline = a.clock + 30
				if err := a.world.Send(proto.CreateCharacterPacket(a.create.Draft)); err != nil {
					a.backToLoginID(12017)
				}
			case ui.CharExit:
				a.creating = false
			}
			return
		}
		a.chars.Light = a.state.Faction.Faction == 0
		a.refreshCharacterRoster()
		a.chars.Busy = a.account.op != 0
		action := a.chars.Update(in, w, h, dt)
		if action != ui.CharNone {
			a.playUI()
		}
		switch action {
		case ui.CharOptions:
			if !a.menuOpen {
				a.openGameMenu()
			}
		case ui.CharStart:
			if ch := a.chars.Chars[a.chars.Selected]; ch != nil {
				a.sendAccountAction(proto.SelectCharacter, ch.ID)
			}
		case ui.CharDelete:
			ch := a.chars.Chars[a.chars.Selected]
			if ch == nil {
				break
			}
			a.confirmCharacterRemoval(*ch)
		case ui.CharCreate:
			a.menuOpen = false
			for slot := uint8(0); slot < 5; slot++ {
				if _, used := a.state.Slots[slot]; !used {
					a.create.Open(slot, a.state.Faction.Faction)
					a.creating = true
					break
				}
			}
		case ui.CharExit:
			a.backToLogin("")
		}
	case game.PhaseFaction:
		if a.factionUI.Busy && a.clock >= a.factionDeadline {
			a.backToLoginID(12018)
			return
		}
		choice := a.factionUI.Update(in, w, h)
		if choice == ui.FactionExit {
			a.backToLogin("")
			return
		}
		if choice >= 0 {
			if err := a.world.Send(proto.FactionPacket(uint8(choice))); err != nil {
				a.backToLoginID(12019)
				return
			}
			a.factionUI.Busy, a.factionDeadline = true, a.clock+10
		}

	case game.PhaseInWorld:
		a.checkNPCWindowRange()
		if a.state == nil {
			return
		}
		if a.screen != screenWorld || a.wv == nil || a.wv.mapID != a.state.MapID {
			a.screen = screenWorld
			a.enterWorld()
		}
		// 0x43bfa0: DirectInput 0x57 (F11) flips DAT_007198c4, which gates
		// the UI draw (0x4354b0, 0x41c5c0) and UI input (0x48caf0).
		if in.HasKey(platform.KeyF11) {
			a.uiHidden = !a.uiHidden
		}
		if a.worldUIHidden() && !a.menuOpen {
			controls := in.WithBindings(a.clientSettings().Bindings)
			if a.wv.uiLeft != 0 {
				controls.MouseDown = false
			}
			if a.wv.uiRight {
				controls.RightPressed, controls.RightDown = false, false
			}
			a.wv.worldLeftPress(&controls, in, a.wv.uiLeft == 0 && !a.wv.uiRight)
			if in.HasKey(platform.KeyShowIDs) {
				a.showCatalogIDs = !a.showCatalogIDs
			}
			if !in.MouseDown {
				a.wv.uiLeft = 0
			}
			if !in.RightDown {
				a.wv.uiRight = false
			}
			a.updateCombat(&controls, w, h)
			if a.state != nil && a.wv != nil && a.world != nil {
				a.updateWorld(&controls, dt)
			}
			return
		}
		if a.state.Self.Dead {
			a.stopTeleportCast()
			if a.summonMessage != nil {
				a.summonMessage.Visible = false
			}
			a.appearanceEdit = nil
			// The dead can still chat (the original keeps the chat line live).
			// Typed text must not reach the death dialog, whose R key revives.
			deathIn := in
			if a.shop.quantity == nil && !a.personalMarket.modal() {
				if a.updateChat(in) {
					stripped := chatWorldInput(*in)
					deathIn = &stripped
				} else {
					pointer := *in
					pointer.Clicked, pointer.Pressed = false, false
					if a.updateChatPointer(in, w, h) {
						deathIn = &pointer
					}
				}
			}
			a.updateDeath(deathIn, w, h)
			if a.state != nil && a.wv != nil && a.world != nil {
				dead := a.deadCameraInput(deathIn)
				a.updateWorld(&dead, dt)
			}
			return
		}
		a.wv.death = deathView{}
		if a.mallConfirm != nil && a.mallConfirm.Visible {
			// The purchase prompt never pauses the game: movement, combat and
			// hotkeys keep running. It owns the pointer over itself, Enter and
			// Escape, and every key while a gift recipient is typed.
			over := a.mallConfirm.Contains(in.MouseX, in.MouseY, w, h)
			typing := a.mall.modal == ui.MallGift
			a.updateMallConfirm(in, w, h)
			if a.state == nil || a.wv == nil || a.world == nil {
				return
			}
			masked := *in
			if typing {
				masked = chatWorldInput(masked)
			} else {
				masked.Keys = nil
				for _, k := range in.Keys {
					if k != platform.KeyEscape && k != platform.KeyEnter {
						masked.Keys = append(masked.Keys, k)
					}
				}
			}
			if over {
				masked.Pressed, masked.Clicked, masked.MouseDown = false, false, false
				masked.RightPressed, masked.RightDown, masked.MiddlePressed = false, false, false
				masked.Wheel = 0
			}
			in = &masked
		}
		if !a.checkPendingActions() {
			return
		}
		if a.updateTeleportCast(in) || a.updateSummonCancel(in) {
			return
		}
		if a.appearanceEdit != nil {
			// The window never pauses the game: movement, camera and combat
			// keep running. It only owns the pointer over itself.
			over := a.appearanceEdit.confirming || a.appearanceUI.Contains(in.MouseX, in.MouseY, w, h) && !in.MouseLook
			a.updateAppearanceEditor(in, w, h)
			if a.state == nil || a.wv == nil || a.world == nil {
				return
			}
			if a.appearanceEdit != nil && (over || a.windowDrag.position == &a.appearanceUI.Position) {
				if a.updateWindowDrag(in, w, h) {
					over = true
				}
			}
			masked := *in
			masked.Keys = nil
			for _, k := range in.Keys {
				if k != platform.KeyEscape {
					masked.Keys = append(masked.Keys, k)
				}
			}
			if over {
				masked.Pressed, masked.Clicked, masked.MouseDown = false, false, false
				masked.RightPressed, masked.RightDown, masked.MiddlePressed = false, false, false
				masked.Wheel = 0
			}
			in = &masked
		}
		if a.updateGuildVote(in, w, h) {
			if a.wv != nil && a.world != nil && a.state != nil {
				a.updateWorld(&platform.Input{MouseX: in.MouseX, MouseY: in.MouseY, SuppressAutoRun: true}, dt)
			}
			return
		}
		if a.updateGuildCreate(in, w, h) {
			if a.wv != nil && a.world != nil && a.state != nil {
				a.updateWorld(&platform.Input{MouseX: in.MouseX, MouseY: in.MouseY, SuppressAutoRun: true}, dt)
			}
			return
		}
		if a.duel.quantity != nil || a.duel.moneyOpen {
			a.updateDuelModal(in)
			if a.wv != nil && a.world != nil && a.state != nil {
				a.updateWorld(&platform.Input{MouseX: in.MouseX, MouseY: in.MouseY, SuppressAutoRun: true}, dt)
			}
			return
		}
		if a.updateDuelInvite(in, w, h) {
			if a.wv != nil && a.world != nil && a.state != nil {
				a.updateWorld(&platform.Input{MouseX: in.MouseX, MouseY: in.MouseY, SuppressAutoRun: true}, dt)
			}
			return
		}
		if a.updateGuildDuelInvite(in, w, h) {
			if a.wv != nil && a.world != nil && a.state != nil {
				a.updateWorld(&platform.Input{MouseX: in.MouseX, MouseY: in.MouseY, SuppressAutoRun: true}, dt)
			}
			return
		}
		if a.updateUnionConfirm(in, w, h) {
			if a.wv != nil && a.world != nil && a.state != nil {
				a.updateWorld(&platform.Input{MouseX: in.MouseX, MouseY: in.MouseY, SuppressAutoRun: true}, dt)
			}
			return
		}
		if a.updateGuildConfirm(in, w, h) {
			if a.wv != nil && a.world != nil && a.state != nil {
				a.updateWorld(&platform.Input{MouseX: in.MouseX, MouseY: in.MouseY, SuppressAutoRun: true}, dt)
			}
			return
		}
		if a.updateGuildInvite(in, w, h) {
			if a.wv != nil && a.world != nil && a.state != nil {
				a.updateWorld(&platform.Input{MouseX: in.MouseX, MouseY: in.MouseY, SuppressAutoRun: true}, dt)
			}
			return
		}
		if a.exchange.quantity != nil || a.exchange.moneyOpen {
			a.updateExchangeModal(in)
			if a.wv != nil && a.world != nil && a.state != nil {
				a.updateWorld(&platform.Input{MouseX: in.MouseX, MouseY: in.MouseY, SuppressAutoRun: true}, dt)
			}
			return
		}
		if a.updateExchangeInvite(in, w, h) {
			if a.wv != nil && a.world != nil && a.state != nil {
				a.updateWorld(&platform.Input{MouseX: in.MouseX, MouseY: in.MouseY, SuppressAutoRun: true}, dt)
			}
			return
		}
		if a.questConfirm != nil && a.questConfirm.Visible {
			a.updateQuestLog(in, w, h)
			if a.state != nil && a.wv != nil && a.world != nil {
				a.updateWorld(&platform.Input{MouseX: in.MouseX, MouseY: in.MouseY, SuppressAutoRun: true}, dt)
			}
			return
		}
		if a.blacksmith.quantity || a.gemConfirm != nil && a.gemConfirm.Visible || a.gateConfirm != nil && a.gateConfirm.Visible {
			a.worldControls(in, w, h)
			if a.state != nil && a.wv != nil && a.world != nil {
				a.updateWorld(&platform.Input{MouseX: in.MouseX, MouseY: in.MouseY, SuppressAutoRun: true}, dt)
			}
			return
		}
		if a.updateVehicleInvite(in, w, h) {
			a.updateWorld(&platform.Input{MouseX: in.MouseX, MouseY: in.MouseY, SuppressAutoRun: true}, dt)
			return
		}
		if a.shop.quantity == nil && (a.updateSummonPrompt(in, w, h) || a.updatePartyInvite(in, w, h)) {
			a.updateWorld(&platform.Input{MouseX: in.MouseX, MouseY: in.MouseY, SuppressAutoRun: true}, dt)
			return
		}
		if a.menuOpen && (a.menuPending || a.menuConfirm != nil && a.menuConfirm.Visible || a.gameMenu.DisplayPopupOpen() || a.gameMenu.BindingCapture()) {
			controls := a.worldControls(in, w, h)
			if a.wv != nil && a.world != nil && a.state != nil {
				a.updateWorld(&controls, dt)
			}
			return
		}
		if a.socialInviteUI != nil && a.socialInviteUI.Visible || a.guild.open && a.guildUI != nil && a.guildUI.Page != 0 && a.socialUI != nil && a.socialUI.Focus && !in.Pressed {
			controls := a.worldControls(in, w, h)
			controls.Down = ""
			controls.SuppressAutoRun = true
			if a.wv != nil && a.world != nil && a.state != nil {
				a.updateWorld(&controls, dt)
			}
			return
		}
		allowsChat := a.shop.quantity == nil && !a.drop.active && !a.personalMarket.modal()
		if allowsChat && a.updateChatItemDrag(in, w, h) {
			a.updateWorld(&platform.Input{MouseX: in.MouseX, MouseY: in.MouseY}, dt)
			return
		}
		if allowsChat && a.chatOpen && a.chatUI != nil && a.chatUI.PopupOpen() && in.HasKey(platform.KeyEscape) &&
			a.updateChatPointer(in, w, h) {
			a.updateWorld(&platform.Input{MouseX: in.MouseX, MouseY: in.MouseY}, dt)
			return
		}
		if allowsChat && a.updateChat(in) {
			if a.wv == nil || a.world == nil || a.state == nil {
				return
			}
			pointer := chatWorldInput(*in)
			in = &pointer
		} else if allowsChat && (a.auction.npc == 0 || a.auction.focus < 0) {
			gameplay := in.WithBindings(a.clientSettings().Bindings)
			in = &gameplay
		}
		controls := a.worldControls(in, w, h)
		if a.state == nil {
			return
		}
		a.updateCombat(&controls, w, h)
		if a.state == nil {
			return
		}
		if a.state.Self.Dead {
			dead := a.deadCameraInput(in)
			a.updateWorld(&dead, dt)
		} else {
			a.updateWorld(&controls, dt)
		}
		if a.state == nil {
			return
		}
	}
}

func (a *App) drawCharacters(w, h int) {
	if a.state == nil {
		return
	}
	y := 40
	line := func(s string, c [4]float32) {
		a.font.Draw(a.sprites, s, 40, y, c, 0)
		y += 18
	}
	switch a.state.Phase {
	case game.PhaseFaction:
		a.factionUI.Draw(a.sprites, a.font, w, h)
	case game.PhaseCharSelect, game.PhaseLoading:
		if a.creating {
			a.create.Draw(a.sprites, a.font, w, h)
			return
		}
		a.chars.Draw(a.sprites, a.font, w, h)
		if a.state.Phase == game.PhaseLoading {
			line("Loading...", white)
		}
	case game.PhaseInWorld:
		a.setTooltip(tooltipContent{})
		if a.worldUIHidden() {
			break
		}
		a.drawPlayerNames(w, h)
		a.drawCombat(w, h)
		a.drawChatBubbles(w, h)
		a.drawPersonalMarketSigns(w, h)
		a.drawMapLabels(w, h)
		a.drawMinimap(w, h, false)
		a.drawBuffs(w, h)
		a.drawBuffTooltip(w, h)
		a.drawParty(w, h)
		a.drawChat(w, h)
		a.drawQuickPotions(w, h)
		a.drawEvolutionBar(w, h)
		a.drawRevolver(w, h)
		a.drawShop(w, h)
		a.drawExchange(w, h)
		a.drawDuel(w, h)
		a.drawGuildDuel(w, h)
		a.drawGuildRanking(w, h)
		a.drawInventory(w, h)
		a.drawSkills(w, h)
		if a.subActionsOpen && a.subActionsUI != nil {
			walking, autoRun := false, false
			if a.wv != nil {
				walking, autoRun = a.wv.walking, a.wv.autoRun
			}
			a.subActionsUI.Draw(a.sprites, w, h, walking, autoRun)
		}
		a.drawStatus(w, h)
		a.drawQuestLog(w, h)
		a.drawGuild(w, h)
		a.drawPersonalMarket(w, h)
		a.drawMall(w, h)
		a.drawPlayerContext(w, h)
		a.chatUI.DrawOverlay(a.sprites, a.chatFont(), a.white, h)
		a.drawTeleportStone(w, h)
		if a.summonMessage != nil {
			a.summonMessage.Draw(a.sprites, a.font, w, h)
		}
		if a.gateConfirm != nil {
			a.gateConfirm.Draw(a.sprites, a.font, w, h)
		}
		if a.gemConfirm != nil {
			a.gemConfirm.Draw(a.sprites, a.font, w, h)
		}
		a.drawDropDialog(w, h)
		if a.questConfirm != nil {
			a.questConfirm.Draw(a.sprites, a.font, w, h)
		}
		a.drawShopDrag()
		a.drawMinimap(w, h, true)
		if a.appearanceEdit != nil {
			a.appearanceUI.Draw(a.sprites, a.font, w, h)
			a.drawAppearancePreview(w, h)
			if a.appearanceConfirm != nil {
				a.appearanceConfirm.Draw(a.sprites, a.font, w, h)
			}
		}
		if a.duel.quantity != nil {
			a.quantityUI.Draw(a.sprites, a.font)
		}
		if a.duel.moneyOpen && a.exchangeMoney != nil {
			a.exchangeMoney.Draw(a.sprites, a.font)
		}
		if a.exchange.quantity != nil {
			a.quantityUI.Draw(a.sprites, a.font)
		}
		if a.exchange.moneyOpen && a.exchangeMoney != nil {
			a.exchangeMoney.Draw(a.sprites, a.font)
		}
		if a.blacksmith.quantity {
			a.quantityUI.Draw(a.sprites, a.font)
		}
		if a.shop.quantity != nil {
			a.quantityUI.Draw(a.sprites, a.font)
		}
		a.drawPersonalMarketModal(w)
		a.drawRecreationDetails(w, h)
		a.drawScreenNotices(w, h)
		a.drawSharedTooltip(w, h)
		a.drawWarehouseDrag()
		a.drawInventoryDrag()
		a.drawQuickPotionDrag()
		a.drawShortcutDrag()
		a.drawRevolverDrag()
		a.drawLocalActionDrag()
		if a.state.Self.Dead {
			a.drawDeath(w, h)
		}
	}
	if a.menuOpen && a.gameMenu != nil {
		a.gameMenu.Draw(a.sprites, a.font, w, h, a.white)
		a.menuConfirm.Draw(a.sprites, a.font, w, h)
	}

}
