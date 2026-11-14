//go:build integration

package app

import (
	"math"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
	"github.com/Spelunkern/Shaiya-GX/internal/testworld"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
	"github.com/go-gl/gl/v4.5-core/gl"
)

func TestGatekeeperAgainstRapture(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_GATEKEEPER") != "1" {
		t.Skip("requires disposable gatekeeper fixture")
	}
	st, world, send, wait := testworld.Enter(t)
	arc, err := archive.Open(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	a := inputApp()
	a.state, a.world, a.arc = st, world, arc
	a.npcDefs = nil
	a.gateUI, _ = ui.NewGatekeeper(inputTextures{})
	a.gateConfirm, _ = ui.NewMessage(inputTextures{})
	a.npcQuestUI, _ = ui.NewNPCQuest(inputTextures{})
	a.wv.pos[0], a.wv.pos[1], a.wv.pos[2] = st.Self.X, st.Self.Y, st.Self.Z
	a.wv.mapID = st.MapID
	var id uint32
	wait(func() bool {
		for k, e := range st.Entities {
			if e.Kind == game.KindNpc && e.NpcType == 2 && e.TypeID == 32 {
				id = k
				return true
			}
		}
		return false
	})
	a.worldControls(&platform.Input{Chars: []rune{'n'}}, 1024, 768)
	if a.shop.npcID != id {
		t.Fatal("gatekeeper did not open")
	}
	def := a.npcDefs[sdata.NPCKey{Type: 2, ID: 32}]
	if def.Gates[0].MapID == 0 || def.Gates[0].Name == "" || def.Gates[2].MapID != 0 {
		t.Fatalf("unexpected destinations: %+v", def.Gates)
	}
	money, serial := st.Details.Money, st.GatekeeperSerial
	// Empty destination is rejected with authoritative money and no movement.
	send(proto.GatekeeperPacket(id, 2))
	wait(func() bool { return st.GatekeeperSerial > serial })
	if st.LastGatekeeper.Result != 0 || st.Details.Money != money {
		t.Fatal("invalid destination not refused")
	}
	oldX, oldZ := st.Self.X, st.Self.Z
	x, y := a.gateUI.Origin(1024, 768)
	a.worldControls(&platform.Input{MouseX: x + 440, MouseY: y + 90, Pressed: true, MouseDown: true}, 1024, 768)
	a.worldControls(&platform.Input{MouseX: x + 440, MouseY: y + 90, Clicked: true}, 1024, 768)
	if !a.gateConfirm.Visible || st.Self.X != oldX || st.Self.Z != oldZ {
		t.Fatal("travel must ask before moving")
	}
	a.worldControls(&platform.Input{Keys: []platform.Key{platform.KeyEnter}}, 1024, 768)
	if !a.shop.pending || st.Self.X != oldX || st.Self.Z != oldZ {
		t.Fatal("travel applied optimistically")
	}
	deadline := time.Now().Add(10 * time.Second)
	for a.shop.pending || math.Abs(float64(st.Self.X-def.Gates[0].X)) > .1 || math.Abs(float64(st.Self.Z-def.Gates[0].Z)) > .1 {
		a.pumpWorld()
		if a.state == nil || time.Now().After(deadline) {
			t.Fatal("gatekeeper travel timed out")
		}
		time.Sleep(time.Millisecond)
	}
	if st.LastGatekeeper.Result != 1 || st.MapID != def.Gates[0].MapID || a.shop.npcID != 0 || math.Abs(float64(a.wv.pos[0]-st.Self.X)) > .1 {
		t.Fatal("confirmed travel not reconciled")
	}
	if err := world.Shutdown(10 * time.Second); err != nil {
		t.Fatal(err)
	}
	next, _, _, _ := testworld.Enter(t)
	if next.MapID != st.MapID || math.Abs(float64(next.Self.X-st.Self.X)) > .1 || math.Abs(float64(next.Self.Z-st.Self.Z)) > .1 {
		t.Fatal("gatekeeper travel did not persist")
	}
	t.Log("gatekeeper refusal, confirmation, authoritative travel and persistence verified")
}

func TestGatekeeperCrossMapAgainstRapture(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_GATEKEEPER_PAID") != "1" {
		t.Skip("requires disposable paid gatekeeper fixture")
	}
	st, world, _, wait := testworld.Enter(t)
	arc, err := archive.Open(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	graphics := os.Getenv("SHAIYA_TEST_RENDER") == "1"
	a := inputApp()
	if graphics {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		win, openErr := platform.Open(platform.Config{Title: "gatekeeper map change", Width: 1024, Height: 734, Hidden: true, Debug: true})
		if openErr != nil {
			t.Fatal(openErr)
		}
		defer win.Close()
		if err := render.Init(true); err != nil {
			t.Fatal(err)
		}
		a, err = New(Config{}, arc)
		if err != nil {
			t.Fatal(err)
		}
		defer a.Close()
		a.state, a.world, a.screen = st, world, screenWorld
		a.enterWorld()
	} else {
		a.state, a.world, a.arc = st, world, arc
		a.npcDefs = nil
		a.gateUI, _ = ui.NewGatekeeper(inputTextures{})
		a.gateConfirm, _ = ui.NewMessage(inputTextures{})
		a.npcQuestUI, _ = ui.NewNPCQuest(inputTextures{})
		a.wv.pos[0], a.wv.pos[1], a.wv.pos[2] = st.Self.X, st.Self.Y, st.Self.Z
		a.wv.mapID = st.MapID
	}
	var oldWorld *scene.World
	if graphics {
		if a.wv == nil || a.wv.world == nil {
			t.Fatal("source map did not load")
		}
		oldWorld = a.wv.world
		for streamDeadline := time.Now().Add(10 * time.Second); time.Now().Before(streamDeadline); {
			gl.Viewport(0, 0, 1024, 734)
			gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
			a.Draw(1024, 734)
			if a.SceneReady() {
				break
			}
			time.Sleep(time.Millisecond) // background decode needs wall time, not a fixed draw count
		}
		if !a.SceneReady() {
			t.Fatal("source map did not finish streaming")
		}
	}
	var id uint32
	wait(func() bool {
		for k, e := range st.Entities {
			if e.Kind == game.KindNpc && e.NpcType == 2 && e.TypeID == 2 {
				id = k
				return true
			}
		}
		return false
	})
	a.worldControls(&platform.Input{Chars: []rune{'n'}}, 1024, 768)
	if graphics && a.wv.approachNPC {
		if a.shop.npcID != 0 {
			t.Fatal("portal conversation opened before approach finished")
		}
		for i := 0; i < 600 && a.wv.approachTarget != 0; i++ {
			a.updateWorld(&platform.Input{}, 1.0/60)
		}
		if a.wv.approachTarget != 0 {
			t.Fatal("portal NPC approach did not finish")
		}
	}
	if a.shop.npcID != id {
		t.Fatal("paid gatekeeper did not open")
	}
	def := a.npcDefs[sdata.NPCKey{Type: 2, ID: 2}]
	if def.Gates[0].MapID == st.MapID || def.Gates[0].Cost == 0 {
		t.Fatal("fixture does not leave map for a fee")
	}
	oldMoney, oldMap := st.Details.Money, st.MapID
	x, y := a.gateUI.Origin(1024, 768)
	a.worldControls(&platform.Input{MouseX: x + 440, MouseY: y + 90, Pressed: true, MouseDown: true}, 1024, 768)
	a.worldControls(&platform.Input{MouseX: x + 440, MouseY: y + 90, Clicked: true}, 1024, 768)
	if !a.gateConfirm.Visible {
		t.Fatal("paid travel confirmation missing")
	}
	a.worldControls(&platform.Input{Keys: []platform.Key{platform.KeyEnter}}, 1024, 768)
	if !a.shop.pending || st.MapID != oldMap || st.Details.Money != oldMoney {
		t.Fatal("paid travel changed state before confirmation")
	}
	deadline := time.Now().Add(10 * time.Second)
	for a.shop.pending || st.MapID == oldMap {
		if graphics {
			a.Update(&platform.Input{}, 1024, 734, 1.0/60)
		} else {
			a.pumpWorld()
		}
		if a.state == nil || time.Now().After(deadline) {
			t.Fatal("paid travel timed out")
		}
		time.Sleep(time.Millisecond)
	}
	if st.LastGatekeeper.Result != 1 || st.MapID != def.Gates[0].MapID || st.Details.Money != oldMoney-def.Gates[0].Cost || a.shop.npcID != 0 {
		t.Fatal("cross-map travel or payment differs from server")
	}
	if graphics {
		// Map installation waits until its loading artwork has been presented.
		// Exercise both application phases before asserting scene replacement.
		for deadline := time.Now().Add(20 * time.Second); a.loading.active && time.Now().Before(deadline); {
			a.Update(&platform.Input{}, 1024, 734, 1.0/60)
			a.Draw(1024, 734)
			time.Sleep(time.Millisecond)
		}
		// The application update, rather than the test, must replace the scene.
		if a.wv == nil || a.wv.mapID != st.MapID || a.wv.world == nil || a.wv.world == oldWorld {
			t.Fatal("destination scene was not installed")
		}
		if oldWorld.Sky != nil {
			t.Fatal("source streaming world did not release its sky")
		}
		for streamDeadline := time.Now().Add(10 * time.Second); time.Now().Before(streamDeadline); {
			gl.Viewport(0, 0, 1024, 734)
			gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
			a.Draw(1024, 734)
			if a.SceneReady() {
				break
			}
			time.Sleep(time.Millisecond) // background decode needs wall time, not a fixed draw count
		}
		if !a.SceneReady() || math.Abs(float64(a.wv.pos[0]-st.Self.X)) > .1 || math.Abs(float64(a.wv.pos[2]-st.Self.Z)) > .1 {
			t.Fatal("destination scene did not finish streaming at the confirmed position")
		}
		if code := gl.GetError(); code != gl.NO_ERROR {
			t.Fatalf("gatekeeper scene GL error %#x", code)
		}
		dir := os.Getenv("SHAIYA_TEST_ARTIFACTS")
		if dir == "" {
			dir = t.TempDir()
		}
		if err := render.SavePNG(filepath.Join(dir, "gatekeeper-destination.png"), 1024, 734); err != nil {
			t.Fatal(err)
		}
	}
	if err := world.Shutdown(10 * time.Second); err != nil {
		t.Fatal(err)
	}
	if graphics {
		a.world = nil
	}
	next, _, _, _ := testworld.Enter(t)
	if next.MapID != st.MapID || next.Details.Money != st.Details.Money {
		t.Fatal("paid destination or balance did not persist")
	}
	t.Logf("paid map change %d -> %d and %d gold persisted", oldMap, st.MapID, def.Gates[0].Cost)
}
