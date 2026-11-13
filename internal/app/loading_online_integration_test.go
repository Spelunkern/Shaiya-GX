//go:build integration

package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/testworld"
	"os"
	"runtime"
	"testing"
	"time"
)

func TestLoadingAgainstRapture(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" || os.Getenv("SHAIYA_TEST_DISPOSABLE") != "1" {
		t.Skip("isolated N2 fixture")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Width: 640, Height: 360, Hidden: true})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
	if err := render.Init(false); err != nil {
		t.Fatal(err)
	}
	arc, err := archive.Open(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	st, conn, _, _ := testworld.Enter(t)
	a, err := New(Config{}, arc)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	a.state, a.world, a.screen = st, conn, screenCharacters
	a.Update(&platform.Input{}, 640, 360, .016)
	if !a.loading.active || a.loading.started {
		t.Fatal("online load did not wait for its first visible frame")
	}
	deadline := time.Now().Add(15 * time.Second)
	frames := 0
	for a.loading.active {
		a.Draw(640, 360)
		a.Update(&platform.Input{}, 640, 360, .016)
		frames++
		if a.state == nil || time.Now().After(deadline) {
			t.Fatal("online loading failed")
		}
		time.Sleep(time.Millisecond)
	}
	if !a.SceneReady() || a.screen != screenWorld || a.wv.mapID != st.MapID {
		t.Fatal("online transition did not reach ready world")
	}
	t.Logf("isolated login/character/map session survived %d loading frames", frames)
}
