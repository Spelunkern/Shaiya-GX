package app

import (
	"fmt"
	"github.com/Spelunkern/Shaiya-GX/internal/data/text"
	"github.com/Spelunkern/Shaiya-GX/internal/gm"
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
		if _, _, known := gm.Parse(text); known {
			if err := a.world.Send(proto.ChatPacket(proto.ChannelNormal, false, text)); err != nil {
				a.backToLogin("Unable to send GM command.")
			}
			a.chatOpen = false
			return true
		}
		selectedChannel, channelText, channelCommand := phoenixChannelCommand(text)
		if channelCommand {
			text = channelText
		}
		if !channelCommand && (a.gestureCommand(text) || a.vehicleCommand(text) || a.partyCommand(text) || a.mallCommand(text) || a.questCommand(text) || a.exchangeCommand(text) || a.duelCommand(text) || a.guildCommand(text) || a.guildDuelCommand(text)) {
			a.chatOpen = false
			return true
		}
		if packet, handled := whisperCommand(text); handled && !channelCommand {
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
			if channelCommand {
				channel = selectedChannel
			}
			if text != "" {
				if !channelCommand {
					switch text[0] {
					case '@':
						channel = proto.ChannelParty
						text = strings.TrimSpace(text[1:])
					case '#':
						channel = proto.ChannelGuild
						text = strings.TrimSpace(text[1:])
					case '$':
						channel = proto.ChannelTrade
						text = strings.TrimSpace(text[1:])
					}
				}
				if text == "" {
					a.chatOpen = false
					return true
				}
				wireChannel := channel

				packet := proto.ChatPacket(wireChannel, false, text)
				if channel == 5 {
					recipient := strings.TrimSpace(a.chatUI.Recipient)
					if recipient == "" {
						recipient, text, _ = strings.Cut(text, " ")
					}
					packet = proto.WhisperPacket(recipient, strings.TrimSpace(text))
					if packet == nil {
						a.localizedNotice(1, 232, nil, "Use !name message.")
						return true
					}
				}
				if err := a.world.Send(packet); err != nil {
					a.backToLogin("Unable to send chat message.")
				}
			}
		}
		a.chatOpen = false
	}
	return true
}

func whisperCommand(text string) ([]byte, bool) {
	if strings.HasPrefix(text, "!") {
		name, message, _ := strings.Cut(strings.TrimSpace(text[1:]), " ")
		return proto.WhisperPacket(name, strings.TrimSpace(message)), true
	}
	command, rest, _ := strings.Cut(strings.TrimSpace(text), " ")
	if !strings.EqualFold(command, "/w") && !strings.EqualFold(command, "/whisper") && !strings.EqualFold(command, "/t") && !strings.EqualFold(command, "/tell") {
		return nil, false
	}
	name, message, _ := strings.Cut(strings.TrimSpace(rest), " ")
	return proto.WhisperPacket(name, strings.TrimSpace(message)), true
}

func (a *App) drawChat(w, h int) {
	f := a.chatFont()
	for i, id := range [...]int{5086, 5089, 5088, 5087, 5388, 5377} {
		if label := a.msg.Get(id); label != "" {
			a.chatUI.ChannelNames[i] = label
		}
	}
	a.chatUI.SetEntries(f, a.formattedChat())
	shown := ""
	if a.chatOpen {
		shown = a.chatInput.Text
	}
	a.chatUI.Draw(a.sprites, f, a.white, h, a.chatOpen, shown)
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
		a.chatEntries = append(a.chatEntries, chatEntry(a.msg, line))
	}
	return a.chatEntries
}

// Native prefixes use the runtime message table, never a translated copy.
func chatEntry(messages text.SysMsg, c game.ChatLine) ui.ChatEntry {
	name := c.Sender
	if name == "" && c.SenderID != 0 {
		name = fmt.Sprintf("Player %d", c.SenderID)
	}
	style := c.Style
	body := c.Text
	if c.Op == proto.ChatError && c.MessageID != 0 && messages.Get(c.MessageID) != "" {
		body = messages.Format(c.MessageID, map[string]string{"p": name, "b": c.Text})
	} else if c.Op != 0 && c.Op != proto.ChatError {
		id := 0
		style = 9
		switch c.Op {
		case proto.ChatParty:
			id, style = 353, 10
		case proto.ChatWhisper:
			id, style = 354, 11
		case proto.ChatGuild:
			id, style = 370, 12
		case proto.ChatWorld:
			id, style = 355, 13
		case proto.ChatShout, proto.ChatWorldShout, proto.ChatMap:
			style = 14
		case proto.ChatNormalAdmin:
			style = 16
		case proto.ChatWorldAdmin:
			id, style = 356, 15
		}
		if id != 0 && messages.Get(id) != "" {
			body = messages.Format(id, map[string]string{"p": name, "b": c.Text})
		} else {
			body = name + " : " + c.Text
		}
	}
	rgb := ui.ChatRGB(style)

	return ui.ChatEntry{Text: body, System: game.ChatSystemPane(style), Color: ui.RGBColor(rgb), Style: style, Markup: c.Op == 0}
}

func (a *App) updateChatPointer(in *platform.Input, _, h int) bool {
	if a.chatUI == nil {
		return false
	}
	a.chatUI.Typing = a.chatOpen
	captured := a.chatUI.Update(in, h)
	if a.chatUI.FocusRequested && !a.chatOpen {
		a.updateChat(&platform.Input{Keys: []platform.Key{platform.KeyEnter}})
	}
	if (a.chatUI.Resized || a.chatUI.ChannelChanged) && a.gameMenu != nil {
		a.gameMenu.SetChatRows(a.chatUI.Rows)
		a.gameMenu.SetChatChannel(a.chatUI.Channel)
		a.saveClientSettings()
	}
	return captured
}

func (a *App) chatFont() *ui.Font {
	if a.tooltipFont != nil {
		return a.tooltipFont
	}
	return a.font
}

// Phoenix aliases take precedence over delivery-only diagnostic commands.
func phoenixChannelCommand(value string) (uint8, string, bool) {
	command, body, _ := strings.Cut(value, " ")
	channels := map[string]uint8{"/s": 0, "/say": 0, "/p": 3, "/party": 3, "/g": 2, "/guild": 2, "/1": 4, "/area": 4, "/shout": 4, "/2": 1, "/trade": 1}
	channel, ok := channels[strings.ToLower(command)]
	return channel, strings.TrimSpace(body), ok
}
