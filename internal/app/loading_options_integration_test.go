//go:build integration

package app

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/session"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
	"github.com/go-gl/gl/v4.5-core/gl"
)

func TestLoadingAndCharacterOptionsGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("GL and assets")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Width: 1024, Height: 768, Hidden: true, Debug: true})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
	if err := render.Init(true); err != nil {
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
	defer func() { a.world = nil; a.Close() }()
	dir := os.Getenv("SHAIYA_TEST_ARTIFACTS")
	if dir == "" {
		dir = t.TempDir()
	}
	draw := func(name string) {
		gl.Viewport(0, 0, 1024, 768)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		a.Draw(1024, 768)
		if name != "" {
			if err := render.SavePNG(filepath.Join(dir, name), 1024, 768); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := a.Preview("chars"); err != nil {
		t.Fatal(err)
	}
	a.world = &session.World{}
	if !a.updateMapLoading() || !a.loading.selection {
		t.Fatal("selection scenery bypassed loading art")
	}
	draw("loading-selection.png")
	if a.updateMapLoading() || a.loading.active {
		t.Fatal("selection scenery did not complete")
	}
	a.world = nil
	draw("")
	option := a.chars.Option
	x, y := option.OriginX+option.OffX+20, option.OriginY+option.OffY+10
	a.Update(&platform.Input{MouseX: x, MouseY: y, Pressed: true, MouseDown: true}, 1024, 768, .016)
	a.Update(&platform.Input{MouseX: x, MouseY: y, Clicked: true}, 1024, 768, .016)
	if !a.menuOpen {
		t.Fatal("character Options button did not open game menu")
	}
	mx, my := a.gameMenu.Origin(1024, 768)
	for _, in := range []platform.Input{{MouseX: mx + 70, MouseY: my + 80, Pressed: true, MouseDown: true}, {MouseX: mx + 70, MouseY: my + 80, Clicked: true}} {
		a.Update(&in, 1024, 768, .016)
	}
	if a.gameMenu.Page != ui.MenuGraphics {
		t.Fatal("character graphics options route failed")
	}
	draw("character-graphics-options.png")
	a.Update(&platform.Input{Keys: []platform.Key{platform.KeyEscape}}, 1024, 768, .016)
	a.menuOpen = false
	a.world = &session.World{} // No packets or socket; only exercise the loading state machine.
	a.state.Phase, a.state.MapID = game.PhaseInWorld, 1
	a.state.SelfID = 1
	a.state.Self.X, a.state.Self.Y, a.state.Self.Z = 580, 78, 1762
	if !a.updateMapLoading() || a.loading.started || a.wv != nil {
		t.Fatal("map setup ran before presenting loading art")
	}
	draw("loading-before-map.png")
	if !a.loading.presented {
		t.Fatal("first loading frame not presented")
	}
	if !a.updateMapLoading() || !a.loading.started {
		t.Fatal("presented load did not start")
	}
	previous := float32(0)
	deadline := time.Now().Add(15 * time.Second)
	frames := 0
	for a.loading.active {
		draw("")
		if a.loading.progress < previous {
			t.Fatal("load progress went backwards")
		}
		previous = a.loading.progress
		frames++
		if frames == 3 {
			draw("loading-progress.png")
		}
		a.updateMapLoading()
		if time.Now().After(deadline) {
			t.Fatal("initial residency never completed")
		}
		time.Sleep(time.Millisecond)
	}
	if !a.SceneReady() {
		t.Fatal("overlay closed before map readiness")
	}
	draw("loading-completed.png")
	a.state.MapID = 19
	if !a.updateMapLoading() || a.loading.started {
		t.Fatal("map transfer bypassed its first overlay frame")
	}
	draw("loading-transfer.png")
	a.updateMapLoading()
	deadline = time.Now().Add(15 * time.Second)
	for a.loading.active {
		draw("")
		a.updateMapLoading()
		if time.Now().After(deadline) {
			t.Fatal("map transfer never finished")
		}
		time.Sleep(time.Millisecond)
	}
	if a.wv.mapID != 19 || !a.SceneReady() {
		t.Fatal("transfer did not reach the new map")
	}
	if code := gl.GetError(); code != gl.NO_ERROR {
		t.Fatalf("GL error %#x", code)
	}
	t.Logf("loading art presented before map setup, %d frames until initial residency ready", frames)
}
