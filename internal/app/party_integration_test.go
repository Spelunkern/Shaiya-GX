//go:build integration

package app

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/testworld"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
	"github.com/go-gl/gl/v4.5-core/gl"
)

func TestPartyAgainstRapture(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_PARTNER") == "" {
		t.Skip("requires two disposable accounts")
	}
	st1, w1, send1, wait1 := testworld.Enter(t)
	st2, w2, send2, wait2 := testworld.EnterAs(t, os.Getenv("SHAIYA_TEST_PARTNER"), os.Getenv("SHAIYA_TEST_PARTNER_PASSWORD"))
	a1, a2 := inputApp(), inputApp()
	graphics := os.Getenv("SHAIYA_TEST_RENDER") == "1"
	if graphics {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		win, err := platform.Open(platform.Config{Title: "party probe", Width: 1024, Height: 734, Hidden: true, Debug: true})
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
		a1, err = New(Config{}, arc)
		if err != nil {
			t.Fatal(err)
		}
		defer a1.Close()
	}
	a1.state, a1.world = st1, w1
	a2.state, a2.world = st2, w2
	if graphics {
		a1.screen = screenWorld
		a1.enterWorld()
	}
	a2.partyMessage, _ = ui.NewMessage(inputTextures{})
	a2.partyUI, _ = ui.NewPartyPanel(inputTextures{})
	name1, name2 := st1.Slots[0].Name, st2.Slots[0].Name
	a1.partyCommand("/invite " + name2)
	wait2(func() bool { return st2.Party.InviteSerial == 1 })
	a2.updatePartyInvite(&platform.Input{}, 1024, 768)
	if !a2.partyMessage.Visible || len(st1.Party.Members) != 0 || len(st2.Party.Members) != 0 {
		t.Fatal("invitation changed membership or prompt missing")
	}
	a2.updatePartyInvite(&platform.Input{Keys: []platform.Key{platform.KeyEscape}}, 1024, 768)
	wait1(func() bool { return st1.Party.ReplySerial == 1 })
	if st1.Party.Reply.Result != 0 {
		t.Fatal("decline not acknowledged", st1.Party.Reply)
	}
	st1.Target = st2.SelfID
	a1.partyCommand("/invite")
	wait2(func() bool { return st2.Party.InviteSerial == 2 })
	a2.updatePartyInvite(&platform.Input{}, 1024, 768)
	a2.updatePartyInvite(&platform.Input{Keys: []platform.Key{platform.KeyEnter}}, 1024, 768)
	if len(st2.Party.Members) != 0 {
		t.Fatal("accept changed membership before ACK")
	}
	wait2(func() bool { return len(st2.Party.Members) == 2 })
	wait1(func() bool { return len(st1.Party.Members) == 2 })
	if st1.Party.Leader != st1.SelfID || st2.Party.Leader != st1.SelfID || st1.Party.Member(st2.SelfID).Name != name2 {
		t.Fatal("roster/leader mismatch", st1.Party, st2.Party)
	}
	if graphics {
		for i := 0; i < 240; i++ {
			gl.Viewport(0, 0, 1024, 734)
			gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
			a1.Draw(1024, 734)
			if a1.SceneReady() {
				break
			}
		}
		if !a1.SceneReady() {
			t.Fatal("party scene did not finish streaming")
		}
		if actor := a1.wv.actors[st2.SelfID]; actor != nil && actor.character != nil {
			matrix := actor.character.World
			if matrix[13] != st1.Entities[st2.SelfID].Y {
				t.Fatal("remote party member ignored server height")
			}
		} else {
			t.Fatal("remote party member missing from scene")
		}
		dir := os.Getenv("SHAIYA_TEST_ARTIFACTS")
		if dir == "" {
			dir = t.TempDir()
		}
		if err := render.SavePNG(filepath.Join(dir, "n2-party.png"), 1024, 734); err != nil {
			t.Fatal(err)
		}
		if err := gl.GetError(); err != gl.NO_ERROR {
			t.Fatalf("party GL error %#x", err)
		}
	}
	x, y := ui.PartyOrigin(1024, 768, 2)
	a2.worldControls(&platform.Input{MouseX: x + 40, MouseY: y + 15, Pressed: true, MouseDown: true}, 1024, 768)
	controls := a2.worldControls(&platform.Input{MouseX: x + 40, MouseY: y + 15, Clicked: true}, 1024, 768)
	if controls.Clicked || st2.Target != st1.SelfID {
		t.Fatal("party row selection reached world or failed")
	}
	a1.partyCommand("/leader " + name2)
	wait1(func() bool { return st1.Party.Leader == st2.SelfID })
	wait2(func() bool { return st2.Party.Leader == st2.SelfID })
	a2.chatOpen = true
	a2.chatInput = ui.TextField{Text: "/party N2 party channel", Max: 120}
	a2.updateChat(&platform.Input{Keys: []platform.Key{platform.KeyEnter}})
	wait1(func() bool {
		for _, line := range st1.Chat {
			if line.Op == proto.ChatParty && line.Text == "N2 party channel" {
				return true
			}
		}
		return false
	})
	a2.partyCommand("/kick " + name1)
	wait1(func() bool { return len(st1.Party.Members) == 0 })
	wait2(func() bool { return len(st2.Party.Members) == 0 })
	a1.partyCommand("/invite " + name2)
	wait2(func() bool { return st2.Party.InviteSerial == 3 })
	a2.updatePartyInvite(&platform.Input{}, 1024, 768)
	a2.updatePartyInvite(&platform.Input{Keys: []platform.Key{platform.KeyEnter}}, 1024, 768)
	wait2(func() bool { return len(st2.Party.Members) == 2 })
	wait1(func() bool { return len(st1.Party.Members) == 2 })
	a2.partyCommand("/leave")
	wait1(func() bool { return len(st1.Party.Members) == 0 })
	wait2(func() bool { return len(st2.Party.Members) == 0 })
	send2(proto.LogoutPacket())
	wait2(func() bool { return st2.SelfID == 0 })
	send1(proto.LogoutPacket())
	wait1(func() bool { return st1.SelfID == 0 })
	t.Log("Two disposable players: name/id invite, modal decline/accept, authoritative roster, row selection, leadership, party chat, kick and leave confirmed")
}
