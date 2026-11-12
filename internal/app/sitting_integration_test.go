//go:build integration

package app

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/testworld"
	"github.com/go-gl/gl/v4.5-core/gl"
)

func TestSittingAgainstRapture(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_SITTING") != "1" || os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires disposable accounts and GL")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	window, err := platform.Open(platform.Config{Title: "sitting probe", Width: 1024, Height: 734, Hidden: true, Debug: true})
	if err != nil {
		t.Fatal(err)
	}
	defer window.Close()
	if err := render.Init(true); err != nil {
		t.Fatal(err)
	}
	arc, err := archive.Open(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	st, world, _, _ := testworld.Enter(t)
	a, err := New(Config{}, arc)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	a.state, a.world, a.screen = st, world, screenWorld
	a.enterWorld()
	pump := func(cond func() bool) {
		t.Helper()
		deadline := time.Now().Add(10 * time.Second)
		for !cond() {
			a.pumpWorld()
			if a.state == nil || time.Now().After(deadline) {
				t.Fatal("sitting confirmation timed out")
			}
			time.Sleep(time.Millisecond)
		}
	}
	a.updateWorld(&platform.Input{Chars: []rune{'c'}}, 1.0/60)
	if !a.wv.sitPending || st.Self.Sitting {
		t.Fatal("sit changed before server confirmation")
	}
	pump(func() bool { return st.Self.Sitting && !a.wv.sitPending })
	other, _, _, waitOther := testworld.EnterAs(t, os.Getenv("SHAIYA_TEST_PARTNER"), os.Getenv("SHAIYA_TEST_PARTNER_PASSWORD"))
	waitOther(func() bool { e := other.Entities[st.SelfID]; return e != nil && e.Shape != nil && e.Sitting })
	for i := 0; i < 150; i++ {
		gl.Viewport(0, 0, 1024, 734)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		a.updateWorld(&platform.Input{}, 1.0/60)
		a.Draw(1024, 734)
	}
	if !a.SceneReady() || !st.Self.Sitting {
		t.Fatal("sitting scene failed to load")
	}
	dir := os.Getenv("SHAIYA_TEST_ARTIFACTS")
	if dir == "" {
		dir = t.TempDir()
	}
	if err := render.SavePNG(filepath.Join(dir, "n2-sitting.png"), 1024, 734); err != nil {
		t.Fatal(err)
	}
	pos := a.wv.pos
	a.updateWorld(&platform.Input{Down: "W"}, .1)
	if !a.wv.sitPending || a.wv.pos != pos || !st.Self.Sitting {
		t.Fatal("movement before standing confirmation")
	}
	pump(func() bool { return !st.Self.Sitting && !a.wv.sitPending })
	waitOther(func() bool { e := other.Entities[st.SelfID]; return e != nil && !e.Sitting })
	if a.wv.self == nil || !a.wv.self.PlayingAction(11) {
		t.Fatal("standing transition did not start")
	}
	a.updateWorld(&platform.Input{Down: "W"}, .01)
	if a.wv.pos != pos {
		t.Fatal("movement started before the standing animation ended")
	}
	for i := 0; i < 600 && a.wv.pos == pos; i++ {
		a.updateWorld(&platform.Input{Down: "W"}, 1.0/60)
	}
	if a.wv.pos == pos {
		t.Fatal("movement did not resume after standing")
	}
	if err := gl.GetError(); err != gl.NO_ERROR {
		t.Fatalf("sitting GL error %#x", err)
	}
	t.Log("sit/stand confirmed on both clients, including late observer")
}
