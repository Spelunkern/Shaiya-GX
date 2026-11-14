//go:build integration

package app

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
	"github.com/go-gl/gl/v4.5-core/gl"
)

func TestPhoenixChatAndLoginLogoGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL and original assets")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Width: 1024, Height: 768, Hidden: true, Debug: true})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
	if err = render.Init(true); err != nil {
		t.Fatal(err)
	}
	arc, err := archive.Open(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	a, err := New(Config{}, arc)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	dir := os.Getenv("SHAIYA_TEST_ARTIFACTS")
	if dir == "" {
		dir = t.TempDir()
	}
	save := func(name string) {
		t.Helper()
		if err := render.SavePNG(filepath.Join(dir, name+".png"), 1024, 768); err != nil {
			t.Fatal(err)
		}
	}
	draw := func(screen screen) []byte {
		a.screen = screen
		a.frameDT = 0
		gl.Viewport(0, 0, 1024, 768)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		a.Draw(1024, 768)
		pixels := make([]byte, 512*80*4)
		gl.ReadPixels(256, 768-128, 512, 80, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&pixels[0]))
		return pixels
	}
	a.state = game.New()
	draw(screenLogin)
	login := draw(screenLogin)
	save("phoenix-chat-login-logo")
	servers := draw(screenServers)
	save("phoenix-chat-server-logo")
	if !bytes.Equal(login, servers) {
		t.Fatal("logo scenery changed while selecting a server")
	}
	a.state.Phase = game.PhaseFaction
	characters := draw(screenCharacters)
	if bytes.Equal(servers, characters) {
		t.Fatal("login logo remained after changing scenery")
	}
	if err = a.Preview("world-inventory"); err != nil {
		t.Fatal(err)
	}
	a.inventory.open = false
	a.state.Chat = nil
	for _, line := range []game.ChatLine{{Op: proto.ChatNormal, Sender: "Alice", Text: "Normal message"}, {Op: proto.ChatParty, Sender: "Alice", Text: "Party message"}, {Op: proto.ChatGuild, Sender: "Alice", Text: "Guild message"}, {Op: proto.ChatWhisper, Sender: "Bob", Text: "Whisper message"}, {Op: proto.ChatWorld, Sender: "Alice", Text: "Trade message"}, {Op: proto.ChatMap, Sender: "Bob", Text: "Area message"}} {
		a.state.AddChat(line)
	}
	a.systemMessage(0, 1000, map[string]string{"t": "Wolf", "v": "123"})
	a.systemMessage(1, 1001, map[string]string{"t": "Wolf", "v": "35"})
	a.systemMessage(3, 50, map[string]string{"b": "240"})
	a.systemMessage(3, 53, map[string]string{"b": "50"})
	a.chatOpen = true
	a.chatInput.Text = "Phoenix chat"
	draw(screenWorld)
	save("phoenix-chat-channels")
	a.chatUI.ModeOpen = true
	draw(screenWorld)
	save("phoenix-chat-selector")
	a.chatUI.ModeOpen = false
	a.chatUI.Channel = 5
	a.chatUI.Recipient = "Bob"
	a.chatInput.Text = "!Bob Single-row whisper"
	draw(screenWorld)
	save("phoenix-chat-whisper")
	a.chatOpen = false
	a.chatUI.Channel = 0
	draw(screenWorld)
	save("phoenix-chat-idle")
	if code := gl.GetError(); code != gl.NO_ERROR {
		t.Fatalf("GL error %#x", code)
	}
}

func verifyNativeChat(t *testing.T, a, b *App, pump func(func() bool)) {
	t.Helper()
	submit := func(message string) {
		t.Helper()
		if !a.chatOpen {
			a.updateChat(&platform.Input{Keys: []platform.Key{platform.KeyEnter}})
		}
		a.updateChat(&platform.Input{Chars: []rune(message)})
		a.updateChat(&platform.Input{Keys: []platform.Key{platform.KeyEnter}})
	}
	contains := func(s *game.State, op proto.Opcode, message string) bool {
		for _, line := range s.Chat {
			if line.Op == op && line.Text == message {
				return true
			}
		}
		return false
	}
	a.chatUI.Channel = proto.ChannelNormal
	submit("GX native normal message")
	pump(func() bool {
		return contains(a.state, proto.ChatNormal, "GX native normal message") && contains(b.state, proto.ChatNormal, "GX native normal message")
	})
	own, _ := b.selfSlot()
	a.state.Target = b.state.SelfID
	a.worldControls(&platform.Input{MouseX: 300, MouseY: 20, RightPressed: true, RightDown: true}, 1024, 734)
	if !a.playerContext.open {
		t.Fatal("player context did not open for whisper")
	}
	row := len(a.contextActions(b.state.SelfID)) - 1
	x, y := a.playerContext.x+10, a.playerContext.y+ui.PlayerMenuTextInset+row*contextRowHeight+5
	a.worldControls(&platform.Input{MouseX: x, MouseY: y, Pressed: true, MouseDown: true}, 1024, 734)
	a.worldControls(&platform.Input{MouseX: x, MouseY: y, Clicked: true}, 1024, 734)
	if !a.chatOpen || a.chatUI.Channel != 5 || a.chatUI.Recipient != own.Name {
		t.Fatal("context did not prepare whisper recipient")
	}
	submit("GX native whisper message")
	pump(func() bool {
		return contains(a.state, proto.ChatWhisper, "GX native whisper message") && contains(b.state, proto.ChatWhisper, "GX native whisper message")
	})
	for _, line := range a.state.Chat {
		if line.Op == proto.ChatWhisper && line.Text == "GX native whisper message" {
			entry := chatEntry(a.msg, line)
			if entry.Color != ui.RGBColor(0xf3a0a0) || entry.Text != a.msg.Format(354, map[string]string{"p": own.Name, "b": line.Text}) {
				t.Fatal("whisper did not use SysMsg template and native RGB", entry)
			}
		}
	}
	a.chatUI.Recipient = ""
	submit(own.Name + " GX Phoenix single-row whisper")
	pump(func() bool { return contains(b.state, proto.ChatWhisper, "GX Phoenix single-row whisper") })
	a.chatUI.Channel = proto.ChannelNormal
	submit("!" + own.Name + " GX native prefix whisper")
	pump(func() bool { return contains(b.state, proto.ChatWhisper, "GX native prefix whisper") })
	a.chatUI.Channel = 5
	a.chatUI.Recipient = "MissingGXCharacter"
	submit("GX missing recipient")
	pump(func() bool {
		for _, line := range a.state.Chat {
			if line.MessageID == 347 && line.Op == proto.ChatError {
				return true
			}
		}
		return false
	})
	a.chatUI.Channel = proto.ChannelNormal
	// Real party request from the popup, then a declined response. Membership
	// must stay server-owned throughout the interaction.
	inviteSerial, replySerial := b.state.Party.InviteSerial, a.state.Party.ReplySerial
	a.worldControls(&platform.Input{MouseX: 300, MouseY: 20, RightPressed: true, RightDown: true}, 1024, 734)
	if !a.playerContext.open || a.contextActions(b.state.SelfID)[0].label != "Invite to party" {
		t.Fatal("party row missing")
	}
	x, y = a.playerContext.x+10, a.playerContext.y+ui.PlayerMenuTextInset+5
	a.worldControls(&platform.Input{MouseX: x, MouseY: y, Pressed: true, MouseDown: true}, 1024, 734)
	a.worldControls(&platform.Input{MouseX: x, MouseY: y, Clicked: true}, 1024, 734)
	if len(a.state.Party.Members) != 0 {
		t.Fatal("menu predicted party membership")
	}
	pump(func() bool { return b.state.Party.InviteSerial > inviteSerial })
	if b.state.Party.InviteID != a.state.SelfID {
		t.Fatal("context invited wrong player")
	}
	if err := b.world.Send(proto.PartyAnswerPacket(false, b.state.SelfID)); err != nil {
		t.Fatal(err)
	}
	pump(func() bool { return a.state.Party.ReplySerial > replySerial })
	if len(a.state.Party.Members) != 0 || len(b.state.Party.Members) != 0 {
		t.Fatal("declined popup invite changed membership")
	}
	t.Log("native chat: normal echo and context-menu whisper delivered to both clients; recipient/prefix and missing recipient SysMsg 347 verified")
	t.Log("native player menu: party invitation and declined response confirmed by Rapture")
}
