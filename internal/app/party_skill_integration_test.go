//go:build integration

package app

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/testworld"
	"github.com/go-gl/gl/v4.5-core/gl"
)

func TestPartySkillAgainstRapture(t *testing.T) {
	id, err := strconv.Atoi(os.Getenv("SHAIYA_TEST_PARTY_SKILL"))
	if err != nil || os.Getenv("SHAIYA_TEST_PARTNER") == "" {
		t.Skip("requires two disposable accounts and a party skill")
	}
	a, wa, sendA, waitA := testworld.Enter(t)
	b, _, sendB, waitB := testworld.EnterAs(t, os.Getenv("SHAIYA_TEST_PARTNER"), os.Getenv("SHAIYA_TEST_PARTNER_PASSWORD"))
	waitA(func() bool { return a.Entities[b.SelfID] != nil })
	waitB(func() bool { return b.Entities[a.SelfID] != nil })
	sendA(proto.PartyInvitePacket(b.SelfID))
	waitB(func() bool { return b.Party.InviteSerial != 0 })
	sendB(proto.PartyAnswerPacket(true, b.SelfID))
	waitA(func() bool { return len(a.Party.Members) == 2 })
	waitB(func() bool { return len(b.Party.Members) == 2 })
	var number uint8
	found := false
	for _, s := range a.Skills {
		if int(s.ID) == id && s.Level == 1 {
			number, found = s.Number, true
			break
		}
	}
	if !found {
		t.Fatal("fixture skill absent from character", id)
	}
	var view *App
	if os.Getenv("SHAIYA_TEST_RENDER") == "1" {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		win, err := platform.Open(platform.Config{Title: "party skill probe", Width: 1024, Height: 734, Hidden: true, Debug: true})
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
		view, err = New(Config{}, arc)
		if err != nil {
			t.Fatal(err)
		}
		defer view.Close()
		view.state, view.world, view.screen = a, wa, screenWorld
		view.enterWorld()
		for i := 0; i < 240 && !view.SceneReady(); i++ {
			gl.Viewport(0, 0, 1024, 734)
			gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
			view.Draw(1024, 734)
		}
		if !view.SceneReady() {
			t.Fatal("party scene did not finish streaming")
		}
	}
	sendA(proto.UseSkillPacket(number, 0, true))
	seen := map[uint32]bool{}
	announced := false
	timer := time.NewTimer(20 * time.Second)
	defer timer.Stop()
	for !announced || !seen[a.SelfID] || !seen[b.SelfID] {
		select {
		case p, ok := <-wa.Packets:
			if !ok {
				t.Fatal("world disconnected")
			}
			op := proto.Op(p)
			if _, err := a.Apply(p); err != nil {
				t.Fatalf("packet 0x%04x: %v", op, err)
			}
			if view != nil {
				view.confirmedEffects(p)
			}
			if op == proto.SkillUseNoTarget && int(a.LastSkill.ID) == id && a.LastSkill.Result == 4 {
				announced = true
			}
			if op == proto.SkillPartyMember && int(a.LastSkill.ID) == id {
				if a.LastSkill.Source != a.SelfID || a.LastSkill.Result != 0 {
					t.Fatalf("unexpected party skill: %+v", a.LastSkill)
				}
				seen[a.LastSkill.Target] = true
			}
		case <-timer.C:
			t.Fatalf("party skill timed out: announced=%v targets=%v", announced, seen)
		}
	}
	if view != nil {
		targets := map[uint32]bool{}
		for _, hit := range view.wv.impacts {
			targets[hit.target] = true
			if id == 325 && hit.instance.Skipped != 0 {
				t.Fatalf("healing effect skipped %d emitters", hit.instance.Skipped)
			}
		}
		if !targets[a.SelfID] || !targets[b.SelfID] {
			t.Fatalf("party effects missing: %v", targets)
		}
		view.frameDT = 1.0 / 60
		for i := 0; i < 45; i++ {
			gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
			view.Draw(1024, 734)
		}
		if err := gl.GetError(); err != gl.NO_ERROR {
			t.Fatalf("party effects GL error: %#x", err)
		}
		dir := os.Getenv("SHAIYA_TEST_ARTIFACTS")
		if dir == "" {
			dir = t.TempDir()
		}
		if err := render.SavePNG(filepath.Join(dir, "n2-party-skill.png"), 1024, 734); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("party skill %d confirmed on both members", id)
}
