package app

import (
	"math"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
)

func TestMobDoubleClickRequiresUninterruptedPair(t *testing.T) {
	var c mobClick
	for _, click := range []struct {
		id   uint32
		x, y int
		at   float64
		want bool
	}{
		{1, 100, 100, 1, false}, {1, 103, 102, 1.2, true},
		{1, 103, 102, 1.3, false}, // Third click starts a new pair.
		{2, 103, 102, 1.4, false}, // Another target breaks the pair.
		{2, 110, 102, 1.5, false}, // Pointer moved more than six pixels.
		{2, 110, 102, 2, false},   // Too late.
		{0, 110, 102, 2.1, false}, {2, 110, 102, 2.2, false},
		{2, 110, 102, 1, false}, // Clock rollback cannot trigger an attack.
	} {
		if got := c.double(click.id, click.x, click.y, click.at); got != click.want {
			t.Fatalf("click %+v: double=%v", click, got)
		}
	}
}

func TestPickupRejectionsReachSystemChat(t *testing.T) {
	a := inputApp()
	for code, want := range map[byte]string{
		0: "That item is reserved for another player.",
		1: "Your inventory is full.",
		2: "You do not need that quest item.",
		3: "That item is no longer available.",
		4: "That item is too far away.",
		9: "Unable to pick up that item.",
	} {
		p := proto.NewWriter(proto.AddItem)
		p.U8(0)
		p.U8(code)
		a.confirmedPickup(p.Bytes())
		if got := a.state.Chat[len(a.state.Chat)-1].Text; got != want {
			t.Errorf("reason %d: %q, want %q", code, got, want)
		}
	}
	before := len(a.state.Chat)
	a.confirmedPickup(proto.PickupPacket(42))
	if len(a.state.Chat) != before {
		t.Fatal("success-shaped packet produced a rejection")
	}
}

func TestMobPickingUsesLiveMobsAndVisibleProjection(t *testing.T) {
	a := inputApp()
	a.scene3d = &render.Scene3D{
		View: render.LookAtLH(render.V3{0, 1, -5}, render.V3{0, 1, 0}, render.V3{0, 1, 0}),
		Proj: render.PerspectiveFovLH(math.Pi/4, 1, .1, 100),
	}
	a.state.Entities = map[uint32]*game.Entity{
		1: {ID: 1, Kind: game.KindMob},
		2: {ID: 2, Kind: game.KindMob},
		3: {ID: 3, Kind: game.KindMob, Dead: true},
		4: {ID: 4, Kind: game.KindUser},
		5: {ID: 5, Kind: game.KindMob, Z: -10},
	}
	click := platform.Input{Clicked: true, MouseX: 256, MouseY: 256}
	if target := a.pickMob(&click, 512, 512); target != 1 {
		t.Fatal("overlapping mobs did not pick deterministically", target)
	}
	delete(a.state.Entities, 1)
	delete(a.state.Entities, 2)
	if target := a.pickMob(&click, 512, 512); target != 0 {
		t.Fatal("picked a dead/non-mob/behind-camera entity", target)
	}
	a.scene3d = nil
	if target := a.pickMob(&platform.Input{}, 512, 512); target != 5 {
		t.Fatal("keyboard nearest target incorrectly requires a renderer", target)
	}
}

func TestHUDClickBlocksWorldAndBreaksAttackPair(t *testing.T) {
	for _, point := range [][2]int{{40, 50}, {300, 30}, {717, 715}} {
		a := inputApp()
		a.state.Target = 1
		a.wv.mobClick.double(1, 512, 400, 1)
		controls := a.worldControls(&platform.Input{MouseX: point[0], MouseY: point[1], Pressed: true, MouseDown: true, Down: "W"}, 1024, 768)
		if controls.Pressed || controls.MouseDown || controls.Down != "W" || a.wv.mobClick.id != 0 {
			t.Fatal("HUD press reached world/stopped movement/retained double click", point, controls)
		}
		controls = a.worldControls(&platform.Input{MouseX: 512, MouseY: 400, Clicked: true}, 1024, 768)
		if controls.Clicked {
			t.Fatal("HUD press released over world selected a mob", point)
		}
	}
}

func TestPlayerCanBeSelectedWithoutMobRequest(t *testing.T) {
	a := inputApp()
	a.scene3d = &render.Scene3D{View: render.LookAtLH(render.V3{0, 1, -5}, render.V3{0, 1, 0}, render.V3{0, 1, 0}), Proj: render.PerspectiveFovLH(math.Pi/4, 1, .1, 100)}
	a.state.Entities[42] = &game.Entity{ID: 42, Kind: game.KindUser, Shape: &proto.UserShape{Name: "Ally"}}
	click := &platform.Input{Clicked: true, MouseX: 256, MouseY: 256}
	a.updateCombat(click, 512, 512)
	if a.state.Target != 42 {
		t.Fatalf("player not selected: %d", a.state.Target)
	}
	if got := a.pickMob(&platform.Input{}, 512, 512); got != 0 {
		t.Fatal("tab selected a player", got)
	}
}

func TestPlayerPickingAndEffectsFollowDisplayedPosition(t *testing.T) {
	a := inputApp()
	a.scene3d = &render.Scene3D{View: render.LookAtLH(render.V3{0, 1, -5}, render.V3{0, 1, 0}, render.V3{0, 1, 0}), Proj: render.PerspectiveFovLH(math.Pi/4, 1, .1, 100)}
	a.state.Entities[42] = &game.Entity{ID: 42, Kind: game.KindUser, X: 20, Shape: &proto.UserShape{Name: "Ally"}}
	a.wv.actors = map[uint32]*remoteActor{42: {entity: a.state.Entities[42], position: remotePosition{initialized: true, pos: render.V3{0, 0, 0}}}}
	if got := a.pickMob(&platform.Input{Clicked: true, MouseX: 256, MouseY: 256}, 512, 512); got != 42 {
		t.Fatalf("displayed player not selected: %d", got)
	}
	if got, ok := a.effectPosition(42); !ok || got != (render.V3{0, 0, 0}) {
		t.Fatalf("effect position = %v, %v", got, ok)
	}
}

func TestMonsterPickingAndEffectsFollowDisplayedPosition(t *testing.T) {
	a := inputApp()
	a.scene3d = &render.Scene3D{View: render.LookAtLH(render.V3{0, 1, -5}, render.V3{0, 1, 0}, render.V3{0, 1, 0}), Proj: render.PerspectiveFovLH(math.Pi/4, 1, .1, 100)}
	a.state.Entities[42] = &game.Entity{ID: 42, Kind: game.KindMob, X: 20}
	a.wv.mobs = map[uint32]*mobActor{42: {entity: a.state.Entities[42], pos: render.V3{0, 0, 0}, position: remotePosition{initialized: true}}}
	if got := a.pickMob(&platform.Input{Clicked: true, MouseX: 256, MouseY: 256}, 512, 512); got != 42 {
		t.Fatalf("displayed monster not selected: %d", got)
	}
	if got, ok := a.effectPosition(42); !ok || got != (render.V3{0, 0, 0}) {
		t.Fatalf("effect position = %v, %v", got, ok)
	}
}

func TestNPCPickingUsesVisiblePositionAndConfirmedRange(t *testing.T) {
	a := inputApp()
	a.scene3d = &render.Scene3D{View: render.LookAtLH(render.V3{0, 1, -5}, render.V3{0, 1, 0}, render.V3{0, 1, 0}), Proj: render.PerspectiveFovLH(math.Pi/4, 1, .1, 100)}
	e := &game.Entity{ID: 42, Kind: game.KindNpc, X: 4}
	a.state.Entities[42] = e
	a.wv.npcs = map[uint32]*npcActor{42: {entity: e, actor: &scene.CreatureActor{}, position: remotePosition{initialized: true, pos: render.V3{0, 0, 0}}}}
	in := &platform.Input{Clicked: true, MouseX: 256, MouseY: 256}
	if got := a.pickNPC(in, 512, 512); got != 42 {
		t.Fatalf("visible NPC not picked: %d", got)
	}
	e.X = 10
	if got := a.pickNPC(in, 512, 512); got != 0 {
		t.Fatalf("distant NPC picked: %d", got)
	}
	e.X = 4
	in.Clicked = false
	if got := a.pickNPC(in, 512, 512); got != 0 {
		t.Fatal("hover activated NPC")
	}
}

func TestRayBoxHandlesInsideParallelAndDepth(t *testing.T) {
	lo, hi := render.V3{-1, -1, 4}, render.V3{1, 1, 6}
	if d, ok := rayBox(render.V3{}, render.V3{0, 0, 1}, lo, hi); !ok || d != 4 {
		t.Fatalf("front hit: %v %v", d, ok)
	}
	if _, ok := rayBox(render.V3{2, 0, 0}, render.V3{0, 0, 1}, lo, hi); ok {
		t.Fatal("parallel ray hit outside box")
	}
	if d, ok := rayBox(render.V3{0, 0, 5}, render.V3{0, 0, 1}, lo, hi); !ok || d != 0 {
		t.Fatal("inside origin missed")
	}
	if _, ok := rayBox(render.V3{}, render.V3{0, 0, -1}, lo, hi); ok {
		t.Fatal("box behind ray hit")
	}
}

func TestNativeTargetKeyCyclesByDistanceAndWraps(t *testing.T) {
	a := inputApp()
	a.state.Entities = map[uint32]*game.Entity{1: {ID: 1, Kind: game.KindMob, X: 2}, 2: {ID: 2, Kind: game.KindMob, X: 4}, 3: {ID: 3, Kind: game.KindMob, X: 6, Dead: true}, 4: {ID: 4, Kind: game.KindMob, X: 8}}
	for _, want := range []uint32{1, 2, 4, 1} {
		got := a.pickMob(&platform.Input{Keys: []platform.Key{platform.KeyTarget}}, 1024, 768)
		if got != want {
			t.Fatalf("target=%d want %d", got, want)
		}
		a.state.Target = got
	}
}
