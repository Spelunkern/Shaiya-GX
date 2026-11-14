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
		a.backToLogin("Character request timed out. Reconnect to refresh the roster.")
		return
	}
	if a.updateAccountMessage(in, w, h) {
		return
	}
	if a.state.Phase == game.PhaseCharSelect && !a.creating {
		if a.menuOpen {
			a.updateGameMenu(in, w, h)
			return
		}
		if hasChar(in, 'o') || in.HasKey(platform.KeyEscape) {
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
				a.backToLogin("Character creation timed out. Reconnect to refresh the roster.")
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
					a.backToLogin("Unable to send character creation request.")
				}
			case ui.CharExit:
				a.creating = false
			}
			return
		}
		a.chars.Light = a.state.Faction.Faction == 0
		a.chars.SetChars(a.state.Slots)
		a.chars.Busy = a.account.op != 0
		action := a.chars.Update(in, w, h, dt)
		if action != ui.CharNone {
			a.playUI()
		}
		switch action {
		case ui.CharOptions:
			a.openGameMenu()
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
			for slot := uint8(0); slot < 5; slot++ {
				if _, used := a.state.Slots[slot]; !used {
					a.create.Open(slot, a.state.Faction.Faction, a.state.Faction.MaxMode)
					a.creating = true
					break
				}
			}
		case ui.CharExit:
			a.backToLogin("")
		}
	case game.PhaseFaction:
		if a.factionUI.Busy && a.clock >= a.factionDeadline {
			a.backToLogin("Faction response timed out.")
			return
		}
		choice := a.factionUI.Update(in, w, h)
		if choice == ui.FactionExit {
			a.backToLogin("")
			return
		}
		if choice >= 0 {
			if err := a.world.Send(proto.FactionPacket(uint8(choice))); err != nil {
				a.backToLogin("Unable to select faction.")
				return
			}
			a.factionUI.Busy, a.factionDeadline = true, a.clock+10
		}

	case game.PhaseInWorld:
		if a.screen != screenWorld || a.wv == nil || a.wv.mapID != a.state.MapID {
			a.screen = screenWorld
			a.enterWorld()
		}
		if a.state.Self.Dead {
			a.appearanceEdit = nil
			a.updateDeath(in, w, h)
			if a.state != nil && a.wv != nil && a.world != nil {
				a.updateWorld(&platform.Input{MouseX: in.MouseX, MouseY: in.MouseY, SuppressAutoRun: true}, dt)
			}
			return
		}
		a.wv.death = deathView{}
		if a.mallConfirm != nil && a.mallConfirm.Visible {
			a.updateMallConfirm(in, w, h)
			if a.state != nil && a.wv != nil && a.world != nil {
				a.updateWorld(&platform.Input{MouseX: in.MouseX, MouseY: in.MouseY, SuppressAutoRun: true}, dt)
			}
			return
		}
		if !a.checkPendingActions() {
			return
		}
		if a.appearanceEdit != nil {
			a.updateAppearanceEditor(in, w, h)
			if a.state != nil && a.wv != nil && a.world != nil {
				a.updateWorld(&platform.Input{MouseX: in.MouseX, MouseY: in.MouseY, SuppressAutoRun: true}, dt)
			}
			return
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
		if a.shop.quantity == nil && a.updatePartyInvite(in, w, h) {
			a.updateWorld(&platform.Input{MouseX: in.MouseX, MouseY: in.MouseY, SuppressAutoRun: true}, dt)
			return
		}
		if a.menuOpen && (a.menuPending || a.menuConfirm != nil && a.menuConfirm.Visible || a.gameMenu.DisplayPopupOpen()) {
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
		if a.chatOpen && a.shop.quantity == nil {
			a.updateChatPointer(in, w, h)
		}
		if a.shop.quantity == nil && a.updateChat(in) {
			// Empty controls still let an active movement send its stop packet.
			if a.wv != nil && a.world != nil && a.state != nil {
				a.updateWorld(&platform.Input{MouseX: in.MouseX, MouseY: in.MouseY, SuppressAutoRun: true}, dt)
			}
			return
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
			a.updateWorld(&platform.Input{MouseX: in.MouseX, MouseY: in.MouseY, SuppressAutoRun: true}, dt)
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
		a.tooltip = nil
		a.tooltipMuted = false
		a.drawPlayerNames(w, h)
		a.drawCombat(w, h)
		a.drawChatBubbles(w, h)
		a.drawMapLabels(w, h)
		a.drawMinimap(w, h, false)
		a.drawBuffs(w, h)
		a.drawBuffTooltip(w, h)
		a.drawParty(w, h)
		a.drawChat(w, h)
		a.drawQuickPotions(w, h)
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
		a.drawMall(w, h)
		a.drawPlayerContext(w, h)
		a.chatUI.DrawOverlay(a.sprites, a.chatFont(), a.white, h)
		if a.gateConfirm != nil {
			a.gateConfirm.Draw(a.sprites, a.font, w, h)
		}
		if a.gemConfirm != nil {
			a.gemConfirm.Draw(a.sprites, a.font, w, h)
		}
		if a.questConfirm != nil {
			a.questConfirm.Draw(a.sprites, a.font, w, h)
		}
		a.drawShopDrag()
		a.drawMinimap(w, h, true)
		if a.appearanceEdit != nil {
			a.appearanceUI.Draw(a.sprites, a.font, w, h)
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
		a.drawRecreationDetails(w, h)
		a.drawSharedTooltip(w, h)
		a.drawShortcutDrag()
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
