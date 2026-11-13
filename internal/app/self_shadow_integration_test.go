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
	"runtime"
	"testing"
)

func TestOnlyOwnCharacterCreatesShadowGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("GL and assets")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Width: 256, Height: 256, Hidden: true})
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
	v := a.wv
	// Isolate caster scheduling from expensive scenery. Both test actors use
	// real character geometry, and off-camera visibility must not create shadows.
	other, err := scene.BuildCharacter(a.assets, scene.Appearance{Helmet: -1}, "000_normal")
	if err != nil {
		t.Fatal(err)
	}
	other.World = render.Basis(v.pos, render.V3{0, 0, 1}, render.V3{0, 1, 0})
	a.state.Entities[42] = &game.Entity{ID: 42, Kind: game.KindUser, Shape: &proto.UserShape{ID: 42}}
	v.actors = map[uint32]*remoteActor{42: {character: other}}
	a.state.Entities[43] = &game.Entity{ID: 43, Kind: game.KindNpc}
	v.npcs = map[uint32]*npcActor{43: {actor: &scene.CreatureActor{Character: other}}}
	a.state.Entities[44] = &game.Entity{ID: 44, Kind: game.KindMob}
	v.mobs = map[uint32]*mobActor{44: {actor: &scene.CreatureActor{Character: other}}}
	a.scene3d.SetCameraView(v.eye, v.target(), render.V3{0, 1, 0})
	a.scene3d.Proj = render.PerspectiveFovLH(.8, 1, .1, 400)
	a.scene3d.Begin()
	a.drawRemotePlayers()
	a.drawNPCs()
	a.drawMonsters()
	if v.shadowAttempted || v.shadow != nil {
		t.Fatal("a remote/NPC/mob allocated a shadow target")
	}
	a.prepareSelfForDraw()
	a.drawSelf()
	if !v.shadowAttempted || v.shadow == nil {
		t.Fatal("own character shadow was removed")
	}
}
