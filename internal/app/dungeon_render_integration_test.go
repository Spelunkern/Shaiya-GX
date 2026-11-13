//go:build integration

package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestDungeonHasNoFogInGameGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("GL and dungeon assets")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	win, err := platform.Open(platform.Config{Width: 1280, Height: 800, Hidden: true})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
	if err = render.Init(false); err != nil {
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
	a.state = game.New()
	a.state.MapID, a.state.SelfID = 3, 1
	a.state.Phase = game.PhaseInWorld
	a.state.Self.X, a.state.Self.Y, a.state.Self.Z = 199.53047, 19.09025, 400.599
	a.state.Self.IsGM = true
	a.state.Slots[0] = proto.CharacterSlot{ID: 1, Race: 0, Class: 0, Height: 2, Name: "Dungeon"}
	a.screen = screenWorld
	a.enterWorld()
	a.wv.facing = render.V3{0, 0, -1}
	a.wv.resetCamera()
	a.Draw(1280, 800)
	if a.scene3d.FogStart != 0 || a.scene3d.FogEnd != 0 {
		t.Fatal("game forces dungeon fog")
	}
	if a.scene3d.Occlusion == nil {
		t.Fatal("game dungeon occlusion mask missing")
	}
	out := os.Getenv("SHAIYA_CAPTURE_DIR")
	if out == "" {
		out = t.TempDir()
	}
	if err = os.MkdirAll(out, 0700); err != nil {
		t.Fatal(err)
	}
	if err = render.SavePNG(filepath.Join(out, "dungeon-game.png"), 1280, 800); err != nil {
		t.Fatal(err)
	}
}
