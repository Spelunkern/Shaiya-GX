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

func TestNativeChatBubblesWorldMapAndExternalFontsGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("GL and resources")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	win, err := platform.Open(platform.Config{Width: 1280, Height: 800, Hidden: true, Debug: true})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
	if err = render.Init(true); err != nil {
		t.Fatal(err)
	}
	arc, err := archive.OpenDirectory(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	a, err := New(Config{}, arc)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	if err = a.Preview("world-stream"); err != nil {
		t.Fatal(err)
	}
	for i := 0; !a.SceneReady(); i++ {
		if i > 500 {
			t.Fatal("world did not load")
		}
		a.Draw(1280, 800)
	}
	out := os.Getenv("SHAIYA_CAPTURE_DIR")
	if out == "" {
		out = t.TempDir()
	}
	if err = os.MkdirAll(out, 0700); err != nil {
		t.Fatal(err)
	}
	capture := func(name string) {
		t.Helper()
		a.Draw(1280, 800)
		if err = render.SavePNG(filepath.Join(out, name+".png"), 1280, 800); err != nil {
			t.Fatal(err)
		}
	}
	slot := a.state.Slots[0]
	slot.Name = "Jugador"
	a.state.Slots[0] = slot
	a.clock = 10
	for _, op := range []proto.Opcode{proto.ChatNormal, proto.ChatParty, proto.ChatWhisper, proto.ChatGuild, proto.ChatWorld, proto.ChatShout} {
		a.state.AddChat(game.ChatLine{Op: op, Sender: "Alice", Text: "Color original del canal"})
	}
	a.state.AddChat(game.ChatLine{Op: proto.ChatNormal, SenderID: a.state.SelfID, Text: "Hola 😀 👨‍👩‍👧‍👦 🇦🇷"})
	a.confirmedChatBubble(proto.ChatNormal)
	a.state.AddChat(game.ChatLine{Op: proto.ChatNormal, Sender: "Unicode", Text: "你好 日本語 한글 — مرحبا — नमस्ते"})
	capture("native-chat-bubble-clock")
	if len(a.chatBubbles) != 1 {
		t.Fatal("ordinary bubble missing")
	}
	v := a.mapState()
	v.full = true
	a.loadWorldMap(v)
	a.updateMinimap(&platform.Input{}, 1280, 800)
	x, y, scale := a.minimap.FullRect(1280, 800)
	click := func(mx, my int) {
		t.Helper()
		a.worldControls(&platform.Input{MouseX: mx, MouseY: my, Pressed: true, MouseDown: true}, 1280, 800)
		a.worldControls(&platform.Input{MouseX: mx, MouseY: my, Clicked: true}, 1280, 800)
	}
	click(x+int(500*scale), y+int(10*scale))
	if !v.atlas.overview {
		t.Fatal("W was captured by window dragging instead of opening World Map")
	}
	point := func(mapID int) (int, int) {
		t.Helper()
		for _, r := range v.atlas.regions {
			if r.Map == mapID {
				return x + int((16+float32(r.Left+r.Right)/2)*scale), y + int((43+float32(r.Top+r.Bottom)/2)*scale)
			}
		}
		t.Fatal("region missing")
		return 0, 0
	}
	mx, my := point(19)
	a.updateMinimap(&platform.Input{MouseX: mx, MouseY: my}, 1280, 800)
	capture("world-map-hover")
	click(mx, my)
	if v.atlas.preview != 19 {
		t.Fatal("friendly map preview missing", v.atlas.preview)
	}
	capture("world-map-preview")
	a.updateMinimap(&platform.Input{MouseX: mx, MouseY: my, RightPressed: true}, 1280, 800)
	mx, my = point(2)
	click(mx, my)
	if !v.atlas.warning {
		t.Fatal("enemy preview warning missing")
	}
	capture("world-map-enemy-warning")
	a.clock = 16
	v.full = false
	capture("bubble-expired")
	if len(a.chatBubbles) != 0 {
		t.Fatal("expired bubble retained")
	}
	// Emoji colours must survive a red chat tint. Compare the same glyph drawn
	// in two tints; a monochrome or tinted bitmap would produce different pixels.
	font, err := ui.LoadFont(26)
	if err != nil {
		t.Fatal(err)
	}
	defer font.Delete()
	gl.Viewport(0, 0, 1280, 800)
	gl.ClearColor(0, 0, 0, 1)
	gl.Clear(gl.COLOR_BUFFER_BIT)
	a.sprites.Begin(1280, 800)
	font.Draw(a.sprites, "😀", 32, 32, [4]float32{1, 0, 0, 1}, 0)
	font.Draw(a.sprites, "😀", 160, 32, [4]float32{1, 1, 1, 1}, 0)
	a.sprites.End()
	left, right := make([]byte, 64*64*4), make([]byte, 64*64*4)
	gl.ReadPixels(24, 800-24-64, 64, 64, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(left))
	gl.ReadPixels(152, 800-24-64, 64, 64, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(right))
	colored := 0
	for i := 0; i < len(left); i += 4 {
		if left[i+1] > 60 && left[i] > 100 {
			colored++
		}
	}
	if !bytes.Equal(left, right) || colored < 50 {
		t.Fatal("emoji is missing or incorrectly tinted", colored)
	}
	if err = render.SavePNG(filepath.Join(out, "external-colour-emoji.png"), 1280, 800); err != nil {
		t.Fatal(err)
	}
	if code := gl.GetError(); code != gl.NO_ERROR {
		t.Fatalf("GL error %x", code)
	}
}
