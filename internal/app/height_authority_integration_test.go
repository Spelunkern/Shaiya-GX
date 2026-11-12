//go:build integration

package app

import (
	"os"
	"runtime"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

func TestPlayerHeightsRemainServerConfirmedGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL and original assets")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Title: "player height", Width: 256, Height: 256, Hidden: true})
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
	a, err := New(Config{}, arc)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	if err := a.Preview("world-stream"); err != nil {
		t.Fatal(err)
	}
	if a.wv.pos[1] != a.state.Self.Y {
		t.Fatal("initial local height ignored server position")
	}
	remote := &game.Entity{Kind: game.KindUser, ID: 2, X: a.wv.pos[0], Z: a.wv.pos[2], Shape: &proto.UserShape{ID: 2, Height: 2}}
	a.state.Entities[2] = remote
	for _, y := range []float32{a.wv.pos[1] + 15, -1.5, -8} {
		remote.Y = y
		a.drawWorldView(256, 256)
		actor := a.wv.actors[2]
		if actor == nil || actor.character == nil || actor.character.World[13] != y {
			t.Fatalf("server height %v not rendered", y)
		}
	}
	var found bool
	for x := float32(0); x <= float32(a.wv.world.WLD.MapSize) && !found; x += 16 {
		for z := float32(0); z <= float32(a.wv.world.WLD.MapSize); z += 16 {
			if a.wv.world.SwimmingAt(x, -1.5, z) {
				remote.X, remote.Y, remote.Z, found = x, -1.5, z, true
				break
			}
		}
	}
	if !found {
		t.Fatal("no deep water in map 1")
	}
	remote.Motion = 0
	if got := remoteLocomotion(a.wv.world, remote); got != "006_swnormal" {
		t.Fatalf("idle swimmer clip = %s", got)
	}
	remote.Motion = 1
	if got := remoteLocomotion(a.wv.world, remote); got != "007_swim" {
		t.Fatalf("moving swimmer clip = %s", got)
	}
	remote.Y = 1
	if got := remoteLocomotion(a.wv.world, remote); got != "002_run" {
		t.Fatalf("above water clip = %s", got)
	}
}
