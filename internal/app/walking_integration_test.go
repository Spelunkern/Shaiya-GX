//go:build integration

package app

import (
	"math"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/testworld"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

func TestWalkingAndAutoRunAgainstRapture(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_WALKING") != "1" || os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires two disposable accounts and GL")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Title: "walking probe", Width: 1024, Height: 734, Hidden: true})
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
	st, wa, _, waitA := testworld.Enter(t)
	other, wb, _, waitB := testworld.EnterAs(t, os.Getenv("SHAIYA_TEST_PARTNER"), os.Getenv("SHAIYA_TEST_PARTNER_PASSWORD"))
	waitA(func() bool { return st.Entities[other.SelfID] != nil })
	waitB(func() bool { e := other.Entities[st.SelfID]; return e != nil && e.Shape != nil && e.MoveSerial != 0 })
	a, err := New(Config{}, arc)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	a.state, a.world, a.screen = st, wa, screenWorld
	a.enterWorld()
	b, err := New(Config{}, arc)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	b.state, b.world, b.screen = other, wb, screenWorld
	b.enterWorld()
	b.clock = 1
	b.prepareRemotePlayers()
	if b.wv.actors[st.SelfID] == nil || b.wv.actors[st.SelfID].character == nil {
		t.Fatal("observer actor missing")
	}
	a.worldControls(&platform.Input{Chars: []rune{'b'}}, 1024, 734)
	click := func(index int) {
		x, y := a.subActionsUI.Origin(1024, 734)
		cx, cy := ui.SubActionCell(index)
		a.worldControls(&platform.Input{MouseX: x + cx + 2, MouseY: y + cy + 2, Pressed: true, MouseDown: true}, 1024, 734)
		a.worldControls(&platform.Input{MouseX: x + cx + 2, MouseY: y + cy + 2, Clicked: true}, 1024, 734)
	}
	waitMove := func(after uint64, mode uint8) {
		t.Helper()
		deadline := time.Now().Add(8 * time.Second)
		for other.Entities[st.SelfID].MoveSerial <= after || other.Entities[st.SelfID].Motion&0x3f != mode {
			b.pumpWorld()
			if b.state == nil || time.Now().After(deadline) {
				t.Fatal("remote movement timeout")
			}
			time.Sleep(time.Millisecond)
		}
	}
	click(1)
	if !a.wv.walking {
		t.Fatal("walk toggle failed")
	}
	prev := other.Entities[st.SelfID].MoveSerial
	visualStart := b.wv.actors[st.SelfID].position.pos
	start := a.wv.pos
	a.updateWorld(&platform.Input{Down: "W"}, .2)
	walk := float32(math.Hypot(float64(a.wv.pos[0]-start[0]), float64(a.wv.pos[2]-start[2])))
	waitMove(prev, 0)
	if other.Entities[st.SelfID].Motion != 0 {
		t.Fatal("walk mode not preserved in wire packet")
	}
	b.prepareRemotePlayers()
	visual := b.wv.actors[st.SelfID].position.pos
	server := other.Entities[st.SelfID]
	if math.Abs(float64(server.Y-visualStart[1])) > 8 {
		t.Fatalf("fixture changed height too far to test interpolation: start=%v serverY=%v", visualStart, server.Y)
	} else {
		if visual[0] != visualStart[0] || visual[2] != visualStart[2] {
			t.Fatal("remote actor jumped to the received position")
		}
		b.clock += .05
		b.prepareRemotePlayers()
		visual = b.wv.actors[st.SelfID].position.pos
		if visual[0] == visualStart[0] && visual[2] == visualStart[2] {
			t.Fatal("remote actor did not interpolate")
		}
		b.clock += .1
		b.prepareRemotePlayers()
		visual = b.wv.actors[st.SelfID].position.pos
		if visual[0] != server.X || visual[2] != server.Z {
			t.Fatal("remote actor did not reach the server position")
		}
	}
	if !b.wv.actors[st.SelfID].character.PlayingClip("001_walk") {
		c := b.wv.actors[st.SelfID]
		t.Fatalf("observer did not use walking animation: mode=%d serial=%d actor_serial=%d until=%v clock=%v pos=(%v,%v) cached=(%v,%v)", other.Entities[st.SelfID].Motion, other.Entities[st.SelfID].MoveSerial, c.moveSerial, c.walkUntil, b.clock, other.Entities[st.SelfID].X, other.Entities[st.SelfID].Z, c.moveX, c.moveZ)
	}
	click(2)
	if !a.wv.autoRun {
		t.Fatal("auto-run toggle failed")
	}
	start = a.wv.pos
	a.updateWorld(&platform.Input{}, .2)
	if a.wv.pos == start {
		t.Fatal("auto-run did not move")
	}
	start = a.wv.pos
	a.updateWorld(&platform.Input{SuppressAutoRun: true}, .2)
	if a.wv.pos != start || !a.wv.autoRun {
		t.Fatal("modal failed to pause auto-run")
	}
	a.updateWorld(&platform.Input{}, .2)
	if a.wv.pos == start {
		t.Fatal("auto-run did not resume after the modal")
	}
	a.updateWorld(&platform.Input{Keys: []platform.Key{platform.KeyNumLock}}, .01)
	if a.wv.autoRun {
		t.Fatal("Num Lock did not stop auto-run")
	}
	click(1)
	if a.wv.walking {
		t.Fatal("run toggle failed")
	}
	prev = other.Entities[st.SelfID].MoveSerial
	start = a.wv.pos
	a.updateWorld(&platform.Input{Down: "W"}, .2)
	run := float32(math.Hypot(float64(a.wv.pos[0]-start[0]), float64(a.wv.pos[2]-start[2])))
	waitMove(prev, 1)
	if other.Entities[st.SelfID].Motion != 1 {
		t.Fatal("run mode not preserved in wire packet")
	}
	b.prepareRemotePlayers()
	if !b.wv.actors[st.SelfID].character.PlayingClip("002_run") {
		t.Fatal("observer did not use run animation")
	}
	if math.Abs(float64(run-2*walk)) > .03 {
		t.Fatalf("run/walk distances %v/%v", run, walk)
	}
	t.Logf("walk %.3f and run %.3f units per probe; remote modes and modal auto-run pause confirmed", walk, run)
}
