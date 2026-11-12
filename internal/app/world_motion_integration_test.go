//go:build integration

package app

import (
	"math"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/testworld"
	"github.com/go-gl/gl/v4.5-core/gl"
)

func TestWorldMotionAgainstRapture(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires local GL context, assets and disposable account")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Title: "movement probe", Width: 64, Height: 64, Hidden: true, Debug: true})
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
	st, w, _, _ := testworld.Enter(t)
	a, err := New(Config{}, arc)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	a.state, a.world, a.screen = st, w, screenWorld
	a.pumpWorld()
	a.enterWorld()
	v := a.wv
	if v.world == nil || !v.world.HasTerrain() {
		t.Fatal("fixture is not in a field")
	}
	// Choose a non-flat path from real map samples. Reposition only this
	// disposable fixture, as the combat probes do for their mob spawn.
	var start render.V3
	best := float32(0)
	for x := float32(600); x <= 1400; x += 200 {
		for z := float32(600); z <= 1400; z += 200 {
			y := v.world.Height(x, z)
			dry := true
			for d := float32(0); d <= 60; d++ {
				if v.world.Height(x+d, z) <= 0 {
					dry = false
					break
				}
			}
			if !dry {
				continue
			}
			delta := float32(math.Abs(float64(v.world.Height(x+60, z) - y)))
			if delta > best {
				best, start = delta, render.V3{x, y, z}
			}
		}
	}
	if best < 1 {
		t.Fatal("no elevation change found for the movement regression")
	}
	v.step(render.V3{-100000, 0, 100000})
	if v.pos[0] != 0 || v.pos[2] != float32(v.world.WLD.MapSize) {
		t.Fatal("movement escaped field bounds")
	}
	v.place(start)
	v.facing = render.V3{1, 0, 0}
	v.resetCamera()
	offset := v.eye.Sub(v.pos)
	for i := 0; i < 120; i++ {
		a.updateWorld(&platform.Input{Down: "W"}, .1)
		if a.state == nil {
			t.Fatal("movement disconnected")
		}
		if v.eye.Sub(v.pos).Sub(offset).Len() > .01 {
			t.Fatal("camera offset changed while crossing terrain")
		}
	}
	a.updateWorld(&platform.Input{}, .01)
	end := v.pos
	if math.Abs(float64(end[0]-start[0]-60)) > .01 || math.Abs(float64(end[1]-start[1])) < 1 {
		t.Fatal("movement did not follow the selected terrain path", start, end)
	}
	id := st.SelfID
	send := func(p []byte) {
		t.Helper()
		if err := w.Send(p); err != nil {
			t.Fatal(err)
		}
	}
	wait := func(cond func() bool) {
		t.Helper()
		deadline := time.Now().Add(10 * time.Second)
		for !cond() {
			a.pumpWorld()
			if a.state == nil || time.Now().After(deadline) {
				t.Fatal("movement persistence confirmation failed")
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
	send(proto.LogoutPacket())
	wait(func() bool { return st.Phase == game.PhaseCharSelect })
	send(proto.SelectCharacterPacket(id))
	wait(func() bool { return st.Phase == game.PhaseInWorld })
	if got := (render.V3{st.Self.X, st.Self.Y, st.Self.Z}); got.Sub(end).Len() > .01 {
		t.Fatal("Rapture did not persist predicted terrain movement", got, end)
	}
	// Swim below the surface, then re-enter and check that the server retained
	// the actual Y instead of replacing it with terrain or flotation height.
	a.enterWorld()
	v = a.wv
	var water render.V3
	found := false
	for x := float32(10); x < float32(v.world.WLD.MapSize)-20 && !found; x += 10 {
		for z := float32(10); z < float32(v.world.WLD.MapSize)-20; z += 10 {
			floor, deep := v.world.WaterFloor(x, z)
			if deep && floor < -6 && v.world.Height(x+10, z) < -6 {
				water, found = render.V3{x, -3, z}, true
				break
			}
		}
	}
	if !found {
		t.Fatal("fixture map has no deep water for swimming regression")
	}
	v.place(water)
	v.facing = render.V3{1, 0, 0}
	v.resetCamera()
	send(v.movePacket(0))
	for i := 0; i < 10; i++ {
		a.updateWorld(&platform.Input{Down: "W"}, .1)
		if a.state == nil {
			t.Fatal("swimming disconnected")
		}
	}
	a.updateWorld(&platform.Input{}, .01)
	end = v.pos
	if end[0] <= water[0] || math.Abs(float64(end[1]-water[1])) > .01 {
		t.Fatalf("swimming did not preserve submerged altitude: %v → %v", water, end)
	}
	send(proto.LogoutPacket())
	wait(func() bool { return st.Phase == game.PhaseCharSelect })
	send(proto.SelectCharacterPacket(id))
	wait(func() bool { return st.Phase == game.PhaseInWorld })
	if got := (render.V3{st.Self.X, st.Self.Y, st.Self.Z}); got.Sub(end).Len() > .01 {
		t.Fatal("Rapture did not persist submerged movement", got, end)
	}
	if code := gl.GetError(); code != gl.NO_ERROR {
		t.Fatalf("movement GL error: %#x", code)
	}
	t.Logf("terrain and submerged movements persisted in Rapture; final swim position %v", end)
}

func TestWorldCloseAgainstRapture(t *testing.T) {
	st, w, _, _ := testworld.Enter(t)
	end := render.V3{st.Self.X + 1.25, st.Self.Y, st.Self.Z}
	a := inputApp()
	a.state, a.world = st, w
	a.wv = &worldView{mapID: st.MapID, pos: end, facing: render.V3{0, 0, 1}, moving: true}
	a.Close()
	next, world, _, _ := testworld.Enter(t)
	defer world.Close()
	if got := (render.V3{next.Self.X, next.Self.Y, next.Self.Z}); got != end {
		t.Fatal("closing the app lost the last unsent movement", got, end)
	}
	t.Log("window-close path sent the final movement, saved position and released the account before re-entry")
}
