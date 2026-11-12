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
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
	"github.com/go-gl/gl/v4.5-core/gl"
)

func TestVehicleAgainstRapture(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_VEHICLE") != "1" || os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires disposable mount and GL")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Title: "mount probe", Width: 1024, Height: 734, Hidden: true, Debug: true})
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
	st, world, _, wait := testworld.Enter(t)
	other, otherWorld, _, waitOther := testworld.EnterAs(t, os.Getenv("SHAIYA_TEST_PARTNER"), os.Getenv("SHAIYA_TEST_PARTNER_PASSWORD"))
	wait(func() bool { return st.Entities[other.SelfID] != nil })
	waitOther(func() bool { return other.Entities[st.SelfID] != nil })
	a, err := New(Config{}, arc)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	a.state, a.world, a.screen = st, world, screenWorld
	a.enterWorld()
	item := st.Inventory[[2]uint8{0, 13}]
	if item.Type != 42 || item.Count != 1 || st.Self.Mounted {
		t.Fatal("invalid mount fixture")
	}
	a.inventory.open = true
	ix, iy := a.inventoryUI.Origin(1024, 734)
	sx, sy, _, _ := ui.InventoryCell(0, 13)
	a.worldControls(&platform.Input{MouseX: ix + sx + 2, MouseY: iy + sy + 2, Pressed: true, MouseDown: true}, 1024, 734)
	a.worldControls(&platform.Input{MouseX: ix + sx + 2, MouseY: iy + sy + 2, Clicked: true}, 1024, 734)
	a.worldControls(&platform.Input{Chars: []rune{'u'}}, 1024, 734)
	if !a.vehicle.pending || st.Self.Mounted {
		t.Fatal("mount was applied before server acknowledgement")
	}
	deadline := time.Now().Add(10 * time.Second)
	for a.vehicle.pending || !st.Self.Mounted || st.Self.MountShape != 2 {
		a.pumpWorld()
		if a.state == nil || time.Now().After(deadline) {
			t.Fatal("mount confirmation timed out")
		}
		time.Sleep(time.Millisecond)
	}
	waitOther(func() bool {
		e := other.Entities[st.SelfID]
		return e != nil && e.Shape != nil && e.Mounted && e.MountShape == 2
	})
	a.inventory.open = false
	for frame := 0; frame < 240; frame++ {
		a.Update(&platform.Input{}, 1024, 734, 1.0/60)
		gl.Viewport(0, 0, 1024, 734)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		a.Draw(1024, 734)
		if a.SceneReady() && a.wv.selfMount != nil {
			break
		}
	}
	if !a.SceneReady() || a.wv.selfMount == nil || !a.wv.self.Mounted {
		t.Fatal("mount model or rider pose missing")
	}
	seat := a.wv.selfMount.Seat()
	if seat[1]-a.wv.pos[1] < .2 || seat[1]-a.wv.pos[1] > 4 {
		t.Fatalf("implausible seat height: %+v rider=%+v", seat, a.wv.pos)
	}
	if code := gl.GetError(); code != gl.NO_ERROR {
		t.Fatalf("mount GL error %#x", code)
	}
	dir := os.Getenv("SHAIYA_TEST_ARTIFACTS")
	if dir == "" {
		dir = t.TempDir()
	}
	if err := render.SavePNG(filepath.Join(dir, "vehicle-preview.png"), 1024, 734); err != nil {
		t.Fatal(err)
	}
	beforeMove := a.wv.pos
	a.Update(&platform.Input{Down: "W"}, 1024, 734, .1)
	if a.wv.pos == beforeMove || !a.wv.moving {
		t.Fatal("mounted character could not move")
	}
	a.Draw(1024, 734)
	a.Update(&platform.Input{}, 1024, 734, .01)
	observer, err := New(Config{}, arc)
	if err != nil {
		t.Fatal(err)
	}
	observer.state, observer.world, observer.screen = other, otherWorld, screenWorld
	observer.enterWorld()
	for frame := 0; frame < 240; frame++ {
		gl.Viewport(0, 0, 1024, 734)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		observer.Draw(1024, 734)
		if observer.SceneReady() && observer.wv.actors[st.SelfID] != nil && observer.wv.actors[st.SelfID].mount != nil {
			break
		}
	}
	if observer.wv.actors[st.SelfID] == nil || observer.wv.actors[st.SelfID].mount == nil {
		t.Fatal("remote player's mount model missing")
	}
	observer.world = nil
	observer.Close()
	if !a.vehicleCommand("/mount") || !a.vehicle.pending || !st.Self.Mounted {
		t.Fatal("dismount changed state optimistically")
	}
	deadline = time.Now().Add(10 * time.Second)
	for a.vehicle.pending || st.Self.Mounted || st.Self.MountShape != 0 {
		a.pumpWorld()
		if a.state == nil || time.Now().After(deadline) {
			t.Fatal("dismount confirmation timed out")
		}
		time.Sleep(time.Millisecond)
	}
	waitOther(func() bool { e := other.Entities[st.SelfID]; return e != nil && !e.Mounted })
	if err := world.Shutdown(10 * time.Second); err != nil {
		t.Fatal(err)
	}
	a.world = nil
	next, _, _, _ := testworld.Enter(t)
	if next.Inventory[[2]uint8{0, 13}].Type != 42 || next.Self.Mounted {
		t.Fatal("mount equipment or dismounted state not restored")
	}
	t.Log("equipped mount, self/remote shapes, animated model and dismount confirmed")
}
