//go:build integration

package app

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/testworld"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
	"github.com/go-gl/gl/v4.5-core/gl"
)

func TestGesturesAgainstRapture(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_GESTURES") != "1" || os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires disposable accounts and GL")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Title: "gesture probe", Width: 1024, Height: 734, Hidden: true, Debug: true})
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
	st, wa, sendA, waitA := testworld.Enter(t)
	other, wb, _, waitB := testworld.EnterAs(t, os.Getenv("SHAIYA_TEST_PARTNER"), os.Getenv("SHAIYA_TEST_PARTNER_PASSWORD"))
	waitA(func() bool { return st.Entities[other.SelfID] != nil })
	waitA(func() bool { return st.SkillBarReady })
	waitB(func() bool { e := other.Entities[st.SelfID]; return e != nil && e.Shape != nil })
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
	for i := 0; i < 240 && !b.SceneReady(); i++ {
		gl.Viewport(0, 0, 1024, 734)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		b.Draw(1024, 734)
	}
	if !b.SceneReady() || b.wv.actors[st.SelfID] == nil || b.wv.actors[st.SelfID].character == nil {
		t.Fatal("observer actor unavailable")
	}
	// Open the original sub-action panel using the fifth bottom button.
	buttonX, buttonY := 646+4*35+4, 734-64+10+4
	a.worldControls(&platform.Input{MouseX: buttonX, MouseY: buttonY, Pressed: true, MouseDown: true}, 1024, 734)
	a.worldControls(&platform.Input{MouseX: buttonX, MouseY: buttonY, Clicked: true}, 1024, 734)
	if !a.subActionsOpen {
		t.Fatal("sub-action bottom button did not open the panel")
	}
	// Drag Greet to slot 5, then reload its local assignment.
	px, py := a.subActionsUI.Origin(1024, 734)
	cx, cy := ui.SubActionCell(10)
	ax, ay := ui.ActionBarOrigin(1024, 734)
	px, py = px+cx+2, py+cy+2
	ax, ay = ax+30+4*40+2, ay+12
	a.worldControls(&platform.Input{MouseX: px, MouseY: py, Pressed: true, MouseDown: true}, 1024, 734)
	a.worldControls(&platform.Input{MouseX: ax, MouseY: ay, MouseDown: true}, 1024, 734)
	a.worldControls(&platform.Input{MouseX: ax, MouseY: ay, Clicked: true}, 1024, 734)
	if action, ok := a.localAction(4); !ok || action != 10 {
		t.Fatalf("greeting shortcut not assigned: %d %v", action, ok)
	}
	a.wv.localActions = [ui.TotalActionSlots]uint8{}
	a.loadLocalActions()
	if action, ok := a.localAction(4); !ok || action != 10 {
		t.Fatalf("greeting shortcut not restored: %d %v", action, ok)
	}
	var previousAction uint64
	for seq := uint64(1); seq <= 2; seq++ {
		if seq == 1 {
			x, y := a.subActionsUI.Origin(1024, 734)
			cx, cy := ui.SubActionCell(10) // Greet
			a.worldControls(&platform.Input{MouseX: x + cx + 2, MouseY: y + cy + 2, Pressed: true, MouseDown: true}, 1024, 734)
			a.worldControls(&platform.Input{MouseX: x + cx + 2, MouseY: y + cy + 2, Clicked: true}, 1024, 734)
		} else {
			a.worldControls(&platform.Input{Chars: []rune{'5'}}, 1024, 734)
		}
		if st.Self.GestureSerial != seq-1 {
			t.Fatal("gesture applied before server echo")
		}
		deadline := time.Now().Add(10 * time.Second)
		for st.Self.GestureSerial < seq || other.Entities[st.SelfID].GestureSerial < seq {
			a.pumpWorld()
			b.pumpWorld()
			if a.state == nil || b.state == nil || time.Now().After(deadline) {
				t.Fatal("gesture echo timed out")
			}
			time.Sleep(time.Millisecond)
		}
		for _, state := range []*game.State{st, other} {
			if len(state.Chat) < int(seq) || !strings.Contains(state.Chat[len(state.Chat)-1].Text, st.Slots[0].Name) || !strings.Contains(state.Chat[len(state.Chat)-1].Text, "greet") {
				t.Fatal("localized gesture message missing", state.Chat)
			}
		}
		a.updateWorld(&platform.Input{}, 1.0/60)
		b.prepareRemotePlayers()
		if !a.wv.self.PlayingAction(120) || !b.wv.actors[st.SelfID].character.PlayingAction(120) {
			t.Fatal("confirmed greeting did not use native action 120")
		}
		if action := b.wv.actors[st.SelfID].character.ActionID(); action <= previousAction {
			t.Fatal("repeated gesture did not restart")
		} else {
			previousAction = action
		}
	}
	b.frameDT = 1.0 / 60
	for i := 0; i < 20; i++ {
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		b.Draw(1024, 734)
	}
	if err := gl.GetError(); err != gl.NO_ERROR {
		t.Fatalf("gesture GL error %#x", err)
	}
	dir := os.Getenv("SHAIYA_TEST_ARTIFACTS")
	if dir == "" {
		dir = t.TempDir()
	}
	if err := render.SavePNG(filepath.Join(dir, "n2-gesture.png"), 1024, 734); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 240 && !a.SceneReady(); i++ {
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		a.Draw(1024, 734)
	}
	if !a.SceneReady() {
		t.Fatal("gesture panel scene did not load")
	}
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	a.Draw(1024, 734)
	if err := render.SavePNG(filepath.Join(dir, "n2-subactions.png"), 1024, 734); err != nil {
		t.Fatal(err)
	}
	px, py = a.subActionsUI.Origin(1024, 734)
	a.worldControls(&platform.Input{MouseX: px + 251, MouseY: py + 3, Pressed: true, MouseDown: true}, 1024, 734)
	a.worldControls(&platform.Input{MouseX: px + 251, MouseY: py + 3, Clicked: true}, 1024, 734)
	if a.subActionsOpen {
		t.Fatal("native close button did not close the panel")
	}
	a.worldControls(&platform.Input{Chars: []rune{'b'}}, 1024, 734)
	if !a.subActionsOpen {
		t.Fatal("B did not reopen the panel")
	}
	px, py = a.subActionsUI.Origin(1024, 734)
	a.worldControls(&platform.Input{MouseX: px + 50, MouseY: py + 15, Pressed: true, MouseDown: true}, 1024, 734)
	a.worldControls(&platform.Input{MouseX: 500, MouseY: 115, MouseDown: true}, 1024, 734)
	a.worldControls(&platform.Input{MouseX: 500, MouseY: 115, Clicked: true}, 1024, 734)
	if !a.subActionsUI.Position.Moved {
		t.Fatal("gesture panel did not move")
	}
	if !b.wv.actors[st.SelfID].character.PlayingAction(120) {
		t.Fatal("gesture ended before the movement probe")
	}
	sendA(proto.MovePacket(1, 0, st.Self.X, st.Self.Y, st.Self.Z))
	deadline := time.Now().Add(10 * time.Second)
	for other.Entities[st.SelfID].Motion&0x3f == 0 {
		b.pumpWorld()
		if b.state == nil || time.Now().After(deadline) {
			t.Fatal("remote movement did not arrive")
		}
		time.Sleep(time.Millisecond)
	}
	if b.wv.actors[st.SelfID].character.PlayingAction(120) {
		t.Fatal("remote movement did not interrupt the gesture")
	}
	t.Log("two greetings echoed to sender and observer, using native action 120")
}
