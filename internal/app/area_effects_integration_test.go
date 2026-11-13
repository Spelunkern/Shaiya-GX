//go:build integration

package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/testworld"
	"github.com/go-gl/gl/v4.5-core/gl"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestFrozenIceAreaAgainstRapture(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" || os.Getenv("SHAIYA_TEST_DISPOSABLE") != "1" {
		t.Skip("requires GL and disposable priest")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Width: 1024, Height: 734, Hidden: true, Debug: true})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
	if err = render.Init(true); err != nil {
		t.Fatal(err)
	}
	arc, err := archive.Open(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	st, world, send, wait := testworld.Enter(t)
	send(proto.LearnSkillPacket(85, 1))
	wait(func() bool { return st.LastLearn.ID == 85 })
	if st.LastLearn.Result != 0 {
		t.Fatal("priest could not learn Frozen Ice", st.LastLearn)
	}
	a, err := New(Config{}, arc)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	a.state, a.world, a.screen = st, world, screenWorld
	a.enterWorld()
	st.Target = 0
	var victim uint32
	if os.Getenv("SHAIYA_TEST_FROZEN_VICTIMS") == "1" {
		a.wv.place(render.V3{799.4, 39, 1671.7})
		send(a.wv.movePacket(0))
		mob := testworld.LivingMob(t, st, send, wait)
		victim, st.Target = mob.ID, mob.ID
		// Mob snapshots may carry Y=0; the live controller obtains the
		// actual ground from the map, rather than teleporting below it.
		a.wv.place(render.V3{mob.X + 1, a.wv.world.Height(mob.X+1, mob.Z), mob.Z})
		a.wv.snapRenderedCamera()
		send(a.wv.movePacket(0))
	}
	a.frameDT = 0
	a.Draw(1024, 734) // Load the real actor before input, as in a live session.
	st.QuickSlots = []proto.QuickSlot{{Bar: 0, Slot: 0, Bag: 100, Number: uint16(st.LastLearn.Number)}}
	a.useAction(0)
	predictedAction := a.wv.self.ActionID()
	if a.wv.travel || a.wv.approachTarget != 0 {
		t.Fatal("caster area approached a selected victim")
	}
	deadline := time.Now().Add(5 * time.Second)
	lastFrame := time.Now()
	for st.LastSkill.ID != 85 {
		a.pumpWorld()
		if a.state == nil || time.Now().After(deadline) {
			t.Fatal("area announcement missing")
		}
		if st.LastSkill.ID == 85 {
			break
		}
		now := time.Now()
		a.frameDT = now.Sub(lastFrame).Seconds()
		lastFrame = now
		a.Draw(1024, 734)
		time.Sleep(10 * time.Millisecond)
	}
	// This server announces a cast for Frozen Ice even though local SData
	// declares it instant. Preparation time must not age the release effects.
	if st.LastCast.ID == 85 && (a.wv.self.ActionID() == predictedAction || a.wv.self.Time > .1) {
		t.Fatal("server cast reused the stale predicted release clock", a.wv.self.Time)
	}
	if len(a.wv.impacts) > 0 && a.wv.impacts[0].instance.Time > .1 {
		t.Fatal("cast duration consumed the ice effect before its first frame", a.wv.impacts[0].instance.Time)
	}
	if victim == 0 && (st.LastSkill.Result != 4 || st.LastSkill.Target != 0 || len(a.wv.impacts) != 1) {
		t.Fatal("Frozen Ice with no victims has no centre effect", st.LastSkill, len(a.wv.impacts))
	}
	if len(a.wv.impacts) == 0 || a.wv.impacts[0].target != st.SelfID {
		t.Fatal("caster area lost its source anchor")
	}
	if len(a.wv.releaseEffects) != 1 {
		t.Fatal("Frozen Ice omitted its caster release phase", len(a.wv.releaseEffects))
	}
	if victim != 0 {
		found := false
		for !found && time.Now().Before(deadline) {
			a.pumpWorld()
			for _, impact := range a.wv.impacts {
				if impact.target != st.SelfID {
					victim, found = impact.target, true
				}
			}
			time.Sleep(time.Millisecond)
		}
		if !found {
			// Rapture rolls real hit chances. A miss must not create an
			// impact; use the emitted native notices to distinguish all
			// misses from a successful recipient whose effect was lost.
			success, missed := false, false
			for _, line := range st.Chat {
				success = success || line.MessageID >= 1101 && line.MessageID <= 1107
				missed = missed || line.MessageID == 1108
			}
			if success || !missed {
				t.Fatal("server-confirmed Frozen Ice victim has no effect", victim, st.LastSkill)
			}
			t.Log("Frozen Ice missed every recipient; no hit effect expected, checking the caster area")
			victim = 0
		}
	}
	a.frameDT = 1.0 / 60
	for i := 0; i < 50; i++ {
		gl.Viewport(0, 0, 1024, 734)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		a.Draw(1024, 734)
	}
	if len(a.wv.impacts) == 0 || a.wv.impacts[0].instance.Done() {
		t.Fatal("Frozen Ice disappeared before presentation")
	}
	// Prove that the area reaches the framebuffer, rather than merely
	// existing in a queue. Freeze all clocks for the two comparable draws.
	a.frameDT = 0
	impacts := a.wv.impacts
	a.wv.impacts = nil
	a.Draw(1024, 734)
	without := make([]byte, 1024*734*4)
	gl.ReadPixels(0, 0, 1024, 734, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&without[0]))
	a.wv.impacts = impacts
	a.Draw(1024, 734)
	with := make([]byte, len(without))
	gl.ReadPixels(0, 0, 1024, 734, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&with[0]))
	changed := 0
	for i := 0; i < len(with); i += 4 {
		delta := 0
		for c := 0; c < 3; c++ {
			delta += max(int(with[i+c])-int(without[i+c]), int(without[i+c])-int(with[i+c]))
		}
		if delta > 24 {
			changed++
		}
	}
	if changed < 1000 {
		t.Fatal("Frozen Ice instance produced no visible area", changed)
	}
	if code := gl.GetError(); code != gl.NO_ERROR {
		t.Fatalf("area GL error %#x", code)
	}
	dir := os.Getenv("SHAIYA_TEST_ARTIFACTS")
	if dir == "" {
		dir = t.TempDir()
	}
	name := "frozen-ice-area.png"
	if victim != 0 {
		name = "frozen-ice-victims.png"
	}
	if err = render.SavePNG(filepath.Join(dir, name), 1024, 734); err != nil {
		t.Fatal(err)
	}
	t.Logf("Frozen Ice: caster area and release confirmed; victim %d, impact instances %d; changed pixels %d", victim, len(a.wv.impacts), changed)
}

func verifyInstantSkillTiming(t *testing.T, a *App, pump func(func() bool)) {
	t.Helper()
	if err := a.world.Send(proto.LearnSkillPacket(31, 1)); err != nil {
		t.Fatal(err)
	}
	pump(func() bool { return a.state.LastLearn.ID == 31 })
	if a.state.LastLearn.Result != 0 {
		t.Fatal("fighter could not learn Concentration", a.state.LastLearn)
	}
	old := a.state.QuickSlots
	a.state.QuickSlots = []proto.QuickSlot{{Bar: 0, Slot: 0, Bag: 100, Number: uint16(a.state.LastLearn.Number)}}
	a.useAction(0)
	action := a.wv.self.ActionID()
	a.wv.self.Update(.15)
	pump(func() bool { return a.state.LastSkill.ID == 31 })
	if a.wv.self.ActionID() != action || a.wv.self.Time < .15 {
		t.Fatal("instant skill ACK restarted release animation")
	}
	if len(a.wv.impacts) == 0 {
		t.Fatal("self buff did not create its original effect")
	}
	if a.wv.impacts[len(a.wv.impacts)-1].instance.Time < .12 {
		t.Fatal("effect clock did not catch up with its release animation")
	}
	if len(a.wv.releaseEffects) == 0 || a.wv.releaseEffects[len(a.wv.releaseEffects)-1].instance.Time < .12 {
		t.Fatal("Concentration omitted its release or restarted its clock")
	}
	release := a.wv.releaseEffects[len(a.wv.releaseEffects)-1].instance
	origin, matrix := release.Origin, a.wv.self.World
	a.wv.self.World[12] += 3
	a.updateSkillReleaseEffects(1.0 / 60)
	a.wv.self.World = matrix
	if release.Origin != origin {
		t.Fatal("world release followed the caster after emission")
	}
	a.state.QuickSlots = old
	t.Log("instant self spell began on request; ACK retained its animation clock and created its effect")
}
