//go:build integration

package app

import (
	"os"
	"path/filepath"
	"runtime"
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

func TestSelfSkillEffectAgainstRapture(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires local GL context, original assets and disposable fighter")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Title: "skill effect probe", Width: 1024, Height: 734, Hidden: true, Debug: true})
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
	st, world, send, wait := testworld.Enter(t)
	const skillID = 31 // Concentration: self buff, no projectile; level-four fighter.
	number := uint8(0)
	for _, k := range st.Skills {
		if k.ID == skillID {
			number = k.Number
		}
	}
	if number == 0 {
		send(proto.LearnSkillPacket(skillID, 1))
		wait(func() bool { return st.LastLearn.ID == skillID })
		if st.LastLearn.Result != 0 {
			t.Fatal("self-effect fixture could not learn Concentration", st.LastLearn)
		}
		number = st.LastLearn.Number
	}
	a, err := New(Config{}, arc)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	a.state, a.world, a.screen = st, world, screenWorld
	a.enterWorld()
	if a.wv.world == nil {
		t.Fatal("fixture map did not load")
	}
	draw := func() {
		gl.Viewport(0, 0, 1024, 734)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		a.Draw(1024, 734)
		if err := gl.GetError(); err != gl.NO_ERROR {
			t.Fatalf("skill effect GL error: %#x", err)
		}
	}
	for i := 0; i < 240; i++ {
		draw()
		if a.SceneReady() {
			break
		}
	}
	if !a.SceneReady() {
		t.Fatal("fixture map stream did not become ready")
	}
	send(proto.UseSkillPacket(number, st.SelfID, true))
	deadline := time.Now().Add(5 * time.Second)
	for st.LastSkill.ID != skillID {
		a.pumpWorld()
		if a.state == nil || time.Now().After(deadline) {
			t.Fatal("self-effect result missing", a.login.Message)
		}
		time.Sleep(time.Millisecond)
	}
	if st.LastSkill.Result > 1 || len(a.wv.impacts) != 1 || len(a.wv.projectiles) != 0 || a.wv.impacts[0].target != st.SelfID {
		t.Fatal("confirmed self buff did not create its target effect", st.LastSkill, len(a.wv.impacts), len(a.wv.projectiles))
	}
	pools := st.Pools
	a.frameDT = 1.0 / 60
	for i := 0; i < 12; i++ {
		draw()
	}
	if st.Pools != pools || len(a.wv.impacts) == 0 || a.wv.impacts[0].instance.Done() {
		t.Fatal("effect was invisible or mutated gameplay pools")
	}
	dir := os.Getenv("SHAIYA_TEST_ARTIFACTS")
	if dir == "" {
		dir = t.TempDir()
	}
	if err := render.SavePNG(filepath.Join(dir, "n2-self-effect.png"), 1024, 734); err != nil {
		t.Fatal(err)
	}
	t.Log("Concentration confirmed by Rapture; self effect rendered without a projectile or pool mutation")
	pumpUntil := func(ready func() bool) {
		t.Helper()
		until := time.Now().Add(5 * time.Second)
		for !ready() {
			a.pumpWorld()
			if a.state == nil || time.Now().After(until) {
				t.Fatal("buff HUD server acknowledgement missing")
			}
			time.Sleep(time.Millisecond)
		}
	}
	pumpUntil(func() bool { return len(a.buffs.rows) != 0 })
	var instance uint32
	for _, row := range a.buffs.rows {
		if row.buff.Skill == skillID {
			instance = row.buff.Instance
			if row.def.Effect != 4 || !row.timed || row.seconds(a.clock) == 0 {
				t.Fatal("self buff had no definition or display duration", row)
			}
		}
	}
	if instance == 0 {
		t.Fatal("Concentration missing from HUD")
	}
	// Re-enter to exercise 0x010A remaining duration and world-view teardown.
	characterID := st.SelfID
	send(proto.LogoutPacket())
	pumpUntil(func() bool { return st.Phase == game.PhaseCharSelect })
	send(proto.SelectCharacterPacket(characterID))
	pumpUntil(func() bool { return st.Phase == game.PhaseInWorld && st.InventoryReady && len(a.buffs.rows) != 0 })
	a.enterWorld()
	index := -1
	for i, row := range a.buffs.rows {
		if row.buff.Skill == skillID {
			index, instance = i, row.buff.Instance
			if !row.buff.HasRemaining || row.seconds(a.clock) > row.def.KeepSeconds() {
				t.Fatal("re-entry ignored authoritative remaining seconds", row)
			}
		}
	}
	if index < 0 {
		t.Fatal("world teardown discarded restored buffs")
	}
	x, y := ui.BuffSlot(1024, index)
	a.lastMouseX, a.lastMouseY = x+4, y+4
	draw()
	if err := render.SavePNG(filepath.Join(dir, "n2-buff-hud.png"), 1024, 734); err != nil {
		t.Fatal(err)
	}
	before := len(st.Buffs)
	controls := a.worldControls(&platform.Input{MouseX: x + 4, MouseY: y + 4, RightPressed: true, RightDown: true}, 1024, 734)
	if controls.RightPressed || controls.RightDown || len(st.Buffs) != before {
		t.Fatal("cancel reached camera or removed state optimistically")
	}
	pumpUntil(func() bool { return st.LastBuffRemoved == instance && len(a.buffs.rows) < before })
	t.Log("Buff HUD restored from snapshot; right click removed the effect only after 0x050E")
	position, eye, target := a.wv.pos, a.wv.eye, st.Target
	a.worldControls(&platform.Input{Chars: []rune{'M'}}, 1024, 734)
	draw()
	if !a.wv.mapView.full || a.wv.mapView.image == nil || a.wv.pos != position || a.wv.eye != eye || st.Target != target {
		t.Fatal("full map missing or map interaction changed gameplay/camera")
	}
	if err := render.SavePNG(filepath.Join(dir, "n2-full-map.png"), 1024, 734); err != nil {
		t.Fatal(err)
	}
	a.worldControls(&platform.Input{Keys: []platform.Key{platform.KeyEscape}}, 1024, 734)
	if a.wv.mapView.full || st.Phase != game.PhaseInWorld {
		t.Fatal("map Escape closed the game session")
	}
	mapTexture := a.wv.mapView.image.ID
	a.worldControls(&platform.Input{Chars: []rune{'I'}}, 1024, 734)
	ix, iy := a.inventoryUI.Origin(1024, 734)
	a.worldControls(&platform.Input{MouseX: ix + 50, MouseY: iy + 15, Pressed: true, MouseDown: true}, 1024, 734)
	a.worldControls(&platform.Input{MouseX: 350, MouseY: 155, MouseDown: true}, 1024, 734)
	a.worldControls(&platform.Input{MouseX: 350, MouseY: 155, Clicked: true}, 1024, 734)
	ix, iy = a.inventoryUI.Origin(1024, 734)
	if ix != 300 || iy != 140 || a.wv.pos != position || a.wv.eye != eye || st.Target != target {
		t.Fatal("moving inventory changed gameplay/camera or failed", ix, iy)
	}
	draw()
	if err := render.SavePNG(filepath.Join(dir, "n2-inventory-moved.png"), 1024, 734); err != nil {
		t.Fatal(err)
	}
	a.worldControls(&platform.Input{Keys: []platform.Key{platform.KeyEscape}}, 1024, 734)
	send(proto.LogoutPacket())
	pumpUntil(func() bool { return st.Phase == game.PhaseCharSelect })
	if gl.IsTexture(mapTexture) {
		t.Fatal("map image survived world release")
	}
	t.Log("Minimap/full map used original image; M/Escape preserved gameplay and released the map texture on logout")
}
