package app

import (
	"fmt"
	"slices"
	"strings"

	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

// updateChat captures typing before world controls. Channel syntax is a
// delivery adaptation; encoding and opcodes use the verified 0x599a60 writer.
func (a *App) updateChat(in *platform.Input) bool {
	if !a.chatOpen {
		if !in.HasKey(platform.KeyEnter) {
			return false
		}
		a.chatOpen = true
		if a.wv != nil {
			a.wv.mobClick = mobClick{}
		}
		a.chatInput = ui.TextField{Max: 120}
		a.chatHistoryIndex = len(a.chatHistory)
		a.chatDraft = ""
		return true
	}
	if in.HasKey(platform.KeyEscape) {
		a.chatOpen = false
		return true
	}
	a.chatInput.Edit(in)
	if in.HasKey(platform.KeyUp) || in.HasKey(platform.KeyDown) {
		if a.chatHistoryIndex == len(a.chatHistory) {
			a.chatDraft = a.chatInput.Text
		}
		direction := -1
		if in.HasKey(platform.KeyDown) {
			direction = 1
		}
		a.chatHistoryIndex = max(0, min(len(a.chatHistory), a.chatHistoryIndex+direction))
		a.chatInput.Text = a.chatDraft
		if a.chatHistoryIndex < len(a.chatHistory) {
			a.chatInput.Text = a.chatHistory[a.chatHistoryIndex]
		}
	}
	if in.HasKey(platform.KeyEnter) {
		text := strings.TrimSpace(a.chatInput.Text)
		if text != "" {
			a.chatHistory = append(a.chatHistory, text)
			if len(a.chatHistory) > 32 {
				a.chatHistory = a.chatHistory[len(a.chatHistory)-32:]
			}
		}
		if a.gestureCommand(text) || a.vehicleCommand(text) || a.partyCommand(text) || a.mallCommand(text) || a.questCommand(text) || a.exchangeCommand(text) || a.duelCommand(text) || a.guildCommand(text) || a.guildDuelCommand(text) {
			a.chatOpen = false
			return true
		}
		if packet, handled := whisperCommand(text); handled {
			if packet == nil {
				a.questMessage("Use /w name message.")
			} else if err := a.world.Send(packet); err != nil {
				a.backToLogin("Unable to send whisper.")
			}
			a.chatOpen = false
			return true
		}
		if text != "" {
			channel := proto.ChannelNormal
			if a.chatUI != nil {
				channel = a.chatUI.Channel
			}
			for _, prefix := range []struct {
				s string
				c uint8
			}{{"/party ", proto.ChannelParty}, {"/guild ", proto.ChannelGuild}, {"/trade ", proto.ChannelTrade}, {"/shout ", proto.ChannelShout}} {
				if strings.HasPrefix(text, prefix.s) {
					channel = prefix.c
					text = strings.TrimSpace(strings.TrimPrefix(text, prefix.s))
					break
				}
			}
			if text != "" {
				if err := a.world.Send(proto.ChatPacket(channel, false, text)); err != nil {
					a.backToLogin("Unable to send chat message.")
				}
			}
		}
		a.chatOpen = false
	}
	return true
}

func whisperCommand(text string) ([]byte, bool) {
	command, rest, _ := strings.Cut(strings.TrimSpace(text), " ")
	if !strings.EqualFold(command, "/w") && !strings.EqualFold(command, "/whisper") {
		return nil, false
	}
	name, message, _ := strings.Cut(strings.TrimSpace(rest), " ")
	return proto.WhisperPacket(name, strings.TrimSpace(message)), true
}

func (a *App) drawChat(w, h int) {
	f := a.tooltipFont
	if f == nil {
		f = a.font
	}
	a.chatUI.SetEntries(f, a.formattedChat())
	a.chatUI.Draw(a.sprites, f, a.white, h, a.chatOpen, a.chatInput.Text)
}

func (a *App) formattedChat() []ui.ChatEntry {
	if slices.Equal(a.chatCache, a.state.Chat) {
		return a.chatEntries
	}
	clear(a.chatCache)
	a.chatCache = append(a.chatCache[:0], a.state.Chat...)
	clear(a.chatEntries)
	a.chatEntries = a.chatEntries[:0]
	for _, line := range a.state.Chat {
		a.chatEntries = append(a.chatEntries, chatEntry(line))
	}
	return a.chatEntries
}

func chatEntry(c game.ChatLine) ui.ChatEntry {
	name := c.Sender
	if c.Op == proto.ChatWhisper && name != "" {
		if c.Direction == 1 {
			name = "To " + name
		} else {
			name = "From " + name
		}
	}
	if name == "" {
		name = fmt.Sprintf("Player %d", c.SenderID)
		if c.SenderID == 0 {
			name = "System"
		}
	}
	channel := ""
	switch c.Op {
	case proto.ChatWorld, proto.ChatWorldAdmin:
		channel = "Trade"
	case proto.ChatGuild:
		channel = "Guild"
	case proto.ChatParty:
		channel = "Party"
	case proto.ChatShout, proto.ChatWorldShout:
		channel = "Shout"
	case proto.ChatMap:
		channel = "Map"
	case proto.ChatWhisper:
		channel = "Whisper"
	}
	prefix := ""
	if channel != "" {
		prefix = "[" + channel + "] "
	}
	system := c.SenderID == 0 && c.Sender == ""
	tint := white
	// Phoenix chat_hud.gd CHANNEL_COLORS; channel opcodes remain native EP5.
	switch c.Op {
	case proto.ChatParty:
		tint = [4]float32{165.0 / 255, 240.0 / 255, 120.0 / 255, 1}
	case proto.ChatGuild:
		tint = [4]float32{239.0 / 255, 164.0 / 255, 213.0 / 255, 1}
	case proto.ChatWhisper:
		tint = [4]float32{214.0 / 255, 91.0 / 255, 117.0 / 255, 1}
	case proto.ChatWorld, proto.ChatWorldAdmin:
		tint = [4]float32{215.0 / 255, 180.0 / 255, 141.0 / 255, 1}
	case proto.ChatMap, proto.ChatShout, proto.ChatWorldShout:
		tint = [4]float32{112.0 / 255, 185.0 / 255, 245.0 / 255, 1}
	}
	if system {
		tint = [4]float32{243.0 / 255, 199.0 / 255, 120.0 / 255, 1}
	}
	return ui.ChatEntry{Text: fmt.Sprintf("%s[%s] %s", prefix, name, c.Text), System: system, Color: tint}
}

func (a *App) updateChatPointer(in *platform.Input, _, h int) bool {
	if a.chatUI == nil {
		return false
	}
	a.chatUI.Typing = a.chatOpen
	captured := a.chatUI.Update(in, h)
	if a.chatUI.Resized && a.gameMenu != nil {
		a.gameMenu.SetChatRows(a.chatUI.Rows)
		a.saveClientSettings()
	}
	return captured
}
