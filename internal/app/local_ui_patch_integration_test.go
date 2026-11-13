//go:build integration

package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"testing"
)

func TestLocalHUDUnicodeMapAndCastGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("needs assets and GPU")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	win, e := platform.Open(platform.Config{Width: 1280, Height: 800, Hidden: true})
	if e != nil {
		t.Fatal(e)
	}
	defer win.Close()
	if e = render.Init(true); e != nil {
		t.Fatal(e)
	}
	arc, e := archive.Open(os.Getenv("SHAIYA_DATA"))
	if e != nil {
		t.Fatal(e)
	}
	defer arc.Close()
	a, e := New(Config{}, arc)
	if e != nil {
		t.Fatal(e)
	}
	defer a.Close()
	if e = a.Preview("world-stream"); e != nil {
		t.Fatal(e)
	}
	a.state.Slots[0] = proto.CharacterSlot{ID: a.state.SelfID, Name: "Guerrero Ñ ⚔️", Level: 60, Class: 0, Race: 0}
	a.state.Pools = proto.Pools{HP: 4500, MP: 600, SP: 500}
	a.state.Details.MaxHP, a.state.Details.MaxMP, a.state.Details.MaxSP = 4500, 600, 500
	a.state.AddChat(game.ChatLine{Op: proto.ChatNormal, SenderID: a.state.SelfID, Text: "你好 مرحبا नमस्ते 👨‍👩‍👧‍👦 🇦🇷 1️⃣"})
	a.hud.Position.Move(24, 34, 1280, 800, 213, 114)
	a.hud.TargetPosition.Move(420, 38, 1280, 800, 256, 64)
	a.selectSelf()
	drag := func(x, y, dx, dy int) {
		t.Helper()
		if !a.updateWindowDrag(&platform.Input{MouseX: x, MouseY: y, Pressed: true, MouseDown: true}, 1280, 800) {
			t.Fatal("HUD header did not capture drag")
		}
		if !a.updateWindowDrag(&platform.Input{MouseX: x + dx, MouseY: y + dy, Clicked: true}, 1280, 800) {
			t.Fatal("HUD drag release lost")
		}
	}
	drag(24+110, 34+12, 12, 8)
	if x, y := a.hud.Origin(1280, 800); x != 36 || y != 42 {
		t.Fatal("portrait did not move", x, y)
	}
	drag(420+110, 38+12, 12, 8)
	if x, y := a.hud.TargetOrigin(1280, 800); x != 432 || y != 46 {
		t.Fatal("target did not move", x, y)
	}
	a.cfg.RememberLayout = true
	a.saveUILayout()
	own, target := a.hud.Position, a.hud.TargetPosition
	a.hud.Position.Moved = false
	a.hud.TargetPosition.Moved = false
	a.loadUILayout()
	if a.hud.Position != own || a.hud.TargetPosition != target {
		t.Fatal("HUD persistence failed")
	}
	for i := 0; i < 100; i++ {
		a.Draw(1280, 800)
	}
	if !regexp.MustCompile(`^[0-9]{2}:[0-9]{2}$`).MatchString(a.minimap.Caption) {
		t.Fatal("clock contains map name", a.minimap.Caption)
	}
	dir := os.Getenv("SHAIYA_TEST_ARTIFACTS")
	if dir == "" {
		dir = t.TempDir()
	}
	os.MkdirAll(dir, 0755)
	if e = render.SavePNG(filepath.Join(dir, "hud-unicode.png"), 1280, 800); e != nil {
		t.Fatal(e)
	}
	a.wv.mapView.full = true
	a.Draw(1280, 800)
	if e = render.SavePNG(filepath.Join(dir, "map-title-clock.png"), 1280, 800); e != nil {
		t.Fatal(e)
	}
	a.wv.mapView.full = false
	a.loadSkills()
	active := false
	for _, d := range a.skillDefs {
		if d.ReadyTime > 0 && a.wv.self.PlayCast(scene.SkillCastAction(d.Animation, d.Effect), 3) {
			active = true
			break
		}
	}
	if !active {
		t.Fatal("could not start a real preparation animation")
	}
	a.wv.self.Update(.8)
	if left, total, active := a.wv.self.CastTiming(); !active || left <= 0 || total != 3 {
		t.Fatalf("cast clock lost: %v %v %v", left, total, active)
	}
	a.Draw(1280, 800)
	if e = render.SavePNG(filepath.Join(dir, "castbar-unicode.png"), 1280, 800); e != nil {
		t.Fatal(e)
	}
	a.wv.self.CancelCast()
	if _, _, active = a.wv.self.CastTiming(); active {
		t.Fatal("cancelled cast kept its bar")
	}
}
