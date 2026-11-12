//go:build integration

package app

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
	"github.com/go-gl/gl/v4.5-core/gl"
)

func TestNPCNameplateOriginalAssetsGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL and original assets")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Title: "NPC nameplate", Width: 1024, Height: 734, Hidden: true, Debug: true})
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
	defer a.Close()
	if err := a.Preview("world-stream"); err != nil {
		t.Fatal(err)
	}
	a.state.Guild.ID, a.state.Guild.Name = 7, "Guardians"
	right := render.V3{0, 1, 0}.Cross(a.wv.facing)
	pos := a.wv.pos.Add(a.wv.facing.Scale(4)).Add(right.Scale(3))
	pos[1] = a.wv.world.Height(pos[0], pos[2])
	if pos[1] > 1 {
		drop := proto.GroundItem{X: pos[0], Y: 0, Z: pos[2]}
		if got := groundItemLabelPosition(a.wv.world, drop); got[1] != pos[1]+.3 || drop.Y != 0 {
			t.Fatalf("ground drop presentation should follow terrain without changing server data: %v", got)
		}
		drop.Y = pos[1] + 5
		if got := groundItemLabelPosition(a.wv.world, drop); got[1] != drop.Y+.3 {
			t.Fatalf("elevated server drop position was lost: %v", got)
		}
	} else {
		t.Fatal("NPC fixture did not provide elevated terrain for the item label")
	}
	a.state.Entities[42] = &game.Entity{ID: 42, Kind: game.KindNpc, NpcType: 1, TypeID: 6, X: pos[0], Y: pos[1], Z: pos[2]}
	for i := 0; i < 240; i++ {
		gl.Viewport(0, 0, 1024, 734)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		a.Draw(1024, 734)
		if a.SceneReady() {
			break
		}
	}
	if !a.SceneReady() {
		t.Fatal("field did not finish streaming")
	}
	actor := a.wv.npcs[42]
	if actor == nil || actor.actor == nil || actor.actor.Name == "" {
		t.Fatal("original NPC was not rendered")
	}
	if _, ok := actor.actor.NameAnchor(); !ok {
		t.Fatal("NPC pose lacks a name anchor")
	}
	center := a.visualEntityPosition(a.state.Entities[42]).Add(render.V3{0, 1, 0})
	mx, my, ok := projectName(center, a.scene3d.View.Mul(a.scene3d.Proj), 1024, 734)
	if !ok || a.pickNPC(&platform.Input{Clicked: true, MouseX: mx, MouseY: my}, 1024, 734) != 42 {
		t.Fatal("displayed NPC cannot be selected by mouse")
	}
	blocker, err := scene.BuildCharacter(a.assets, scene.Appearance{Helmet: -1}, "000_normal")
	if err != nil {
		t.Fatal(err)
	}
	blockingPos := a.wv.eye.Add(center.Sub(a.wv.eye).Scale(.65)).Sub(render.V3{0, 1, 0})
	blocker.World = render.Basis(blockingPos, render.V3{0, 0, -1}, render.V3{0, 1, 0})
	a.state.Entities[100] = &game.Entity{ID: 100, Kind: game.KindUser, Shape: &proto.UserShape{ID: 100}}
	a.wv.actors[100] = &remoteActor{entity: a.state.Entities[100], character: blocker}
	if a.pickNPC(&platform.Input{Clicked: true, MouseX: mx, MouseY: my}, 1024, 734) != 0 {
		t.Fatal("NPC behind another actor was selected")
	}
	if a.pickMob(&platform.Input{Clicked: true, MouseX: mx, MouseY: my}, 1024, 734) != 100 {
		t.Fatal("front player was not selected over NPC")
	}
	delete(a.state.Entities, 100)
	delete(a.wv.actors, 100)
	monster, err := a.assets.BuildMonster(2009)
	if err != nil {
		t.Fatal(err)
	}
	monster.World = blocker.World
	lo, hi, valid := monster.WorldBounds()
	if !valid {
		t.Fatal("monster bounds unavailable")
	}
	delta := a.wv.eye.Add(center.Sub(a.wv.eye).Scale(.65)).Sub(lo.Add(hi).Scale(.5))
	monster.World[12] += delta[0]
	monster.World[13] += delta[1]
	monster.World[14] += delta[2]
	a.state.Entities[101] = &game.Entity{ID: 101, Kind: game.KindMob}
	a.wv.mobs[101] = &mobActor{entity: a.state.Entities[101], actor: monster}
	if a.pickMob(&platform.Input{Clicked: true, MouseX: mx, MouseY: my}, 1024, 734) != 101 {
		t.Fatal("front monster was not selected over NPC")
	}
	gl.DepthMask(true)
	gl.ClearDepth(0)
	gl.Clear(gl.DEPTH_BUFFER_BIT)
	a.sprites.CaptureNameDepth(1024, 734)
	gl.ClearDepth(1)
	if a.pickNPC(&platform.Input{Clicked: true, MouseX: mx, MouseY: my}, 1024, 734) != 0 {
		t.Fatal("NPC behind closer depth was selected")
	}
	if a.pickMob(&platform.Input{Clicked: true, MouseX: mx, MouseY: my}, 1024, 734) != 0 {
		t.Fatal("player behind closer depth was selected")
	}
	delete(a.state.Entities, 101)
	delete(a.wv.mobs, 101)
	move := proto.NewWriter(proto.MapNpcMove)
	move.U32(42)
	move.U8(0)
	move.F32(pos[0] + 4)
	move.F32(pos[1])
	move.F32(pos[2])
	if _, err := a.state.Apply(move.Bytes()); err != nil {
		t.Fatal(err)
	}
	a.clock += 1
	a.prepareNPCs()
	if actor.actor.World[12] != pos[0] {
		t.Fatal("NPC snapped to patrol destination")
	}
	a.clock += .25
	a.prepareNPCs()
	if actor.actor.World[12] != pos[0]+2 || a.state.Entities[42].X != pos[0]+4 {
		t.Fatal("NPC patrol interpolation did not preserve server position")
	}
	if got := a.visualEntityPosition(a.state.Entities[42]); got[0] != pos[0]+2 {
		t.Fatal("NPC effect did not follow model")
	}
	a.clock += .25
	a.prepareNPCs()
	if actor.actor.World[12] != pos[0]+4 {
		t.Fatal("NPC did not reach patrol destination")
	}
	if dir := os.Getenv("SHAIYA_TEST_ARTIFACTS"); dir != "" {
		if err := render.SavePNG(filepath.Join(dir, "npc-nameplate-preview.png"), 1024, 734); err != nil {
			t.Fatal(err)
		}
	}
	if code := gl.GetError(); code != gl.NO_ERROR {
		t.Fatalf("NPC nameplate GL error %#x", code)
	}
}

func TestNameDepthCaptureIsLazyGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL and original assets")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Title: "name depth laziness", Width: 64, Height: 64, Hidden: true})
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
	a.state = game.New()
	a.wv = &worldView{}
	a.items = map[sdata.ItemKey]*sdata.Item{}
	a.scene3d.View, a.scene3d.Proj = render.Identity(), render.Identity()
	// A known entity without a renderable shape should not allocate depth.
	a.state.Entities[1] = &game.Entity{ID: 1, Kind: game.KindUser}
	gl.Viewport(0, 0, 64, 64)
	a.sprites.Begin(64, 64)
	before := render.GPUBytes()
	a.drawPlayerNames(64, 64)
	a.sprites.End()
	if got := render.GPUBytes(); got != before {
		t.Fatalf("empty name pass allocated GPU storage: %d -> %d", before, got)
	}
	// A visible item label must cause the depth copy to be allocated.
	a.state.GroundItems[2] = proto.GroundItem{Type: 26, X: 0, Y: 0, Z: .5}
	gl.ClearDepth(1)
	gl.DepthMask(true)
	gl.Clear(gl.DEPTH_BUFFER_BIT)
	a.sprites.Begin(64, 64)
	a.drawPlayerNames(64, 64)
	a.sprites.End()
	if got := render.GPUBytes(); got < before+64*64*4 {
		t.Fatal("visible label did not allocate the depth texture", before, got)
	}
	x, y, _, ok := projectLabel(render.V3{0, .3, .5}, render.Identity(), 64, 64)
	if !ok {
		t.Fatal("test item was not visible")
	}
	click := &platform.Input{Clicked: true, MouseX: x, MouseY: y - a.font.Height/2}
	if id := a.pickGroundLabel(click, 64, 64); id != 2 {
		t.Fatal("visible item label was not picked", id)
	}
	gl.ClearDepth(.25)
	gl.Clear(gl.DEPTH_BUFFER_BIT)
	a.sprites.Begin(64, 64)
	a.drawPlayerNames(64, 64)
	a.sprites.End()
	if id := a.pickGroundLabel(click, 64, 64); id != 0 {
		t.Fatal("occluded item label was picked", id)
	}
	if code := gl.GetError(); code != gl.NO_ERROR {
		t.Fatalf("lazy name depth GL error %#x", code)
	}
}
