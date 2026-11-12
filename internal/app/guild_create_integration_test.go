//go:build integration

package app

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/testworld"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

func TestGuildCreationAgainstRapture(t *testing.T) {
	var accounts [][2]string
	if os.Getenv("SHAIYA_TEST_GUILD_ACCOUNTS") == "" {
		t.Skip("requires seven disposable accounts")
	}
	if err := json.Unmarshal([]byte(os.Getenv("SHAIYA_TEST_GUILD_ACCOUNTS")), &accounts); err != nil || len(accounts) != 7 {
		t.Fatal("invalid disposable accounts")
	}
	name := os.Getenv("SHAIYA_TEST_GUILD")
	apps := make([]*App, len(accounts))
	for i, credentials := range accounts {
		st, world, _, _ := testworld.EnterAs(t, credentials[0], credentials[1])
		a := inputApp()
		a.state, a.world = st, world
		a.guildVoteMessage, _ = ui.NewMessage(inputTextures{})
		a.guildConfirm, _ = ui.NewMessage(inputTextures{})
		a.guildCreateUI, _ = ui.NewGuildCreateDialog(inputTextures{})
		a.guildDirectoryUI, _ = ui.NewGuildDirectory(inputTextures{})
		apps[i] = a
	}
	master := apps[0]
	poll := func(cond func() bool) {
		t.Helper()
		deadline := time.Now().Add(15 * time.Second)
		for !cond() {
			for _, a := range apps {
				a.pumpWorld()
				if a.state == nil {
					t.Fatal("disconnected")
				}
			}
			if time.Now().After(deadline) {
				t.Fatal("guild creation timed out")
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
	for i := 1; i < len(apps); i++ {
		target := apps[i]
		if err := master.world.Send(proto.PartyInvitePacket(target.state.SelfID)); err != nil {
			t.Fatal(err)
		}
		poll(func() bool { return target.state.Party.InviteSerial > 0 })
		if err := target.world.Send(proto.PartyAnswerPacket(true, target.state.SelfID)); err != nil {
			t.Fatal(err)
		}
		want := i + 1
		poll(func() bool { return len(master.state.Party.Members) == want })
	}
	poll(func() bool {
		for _, e := range master.state.Entities {
			if e.Kind == game.KindNpc && e.NpcType == 11 && e.TypeID == 1 {
				return true
			}
		}
		return false
	})
	var manager *game.Entity
	for _, e := range master.state.Entities {
		if e.Kind == game.KindNpc && e.NpcType == 11 && e.TypeID == 1 {
			manager = e
			break
		}
	}
	master.wv.pos = render.V3{manager.X, manager.Y, manager.Z}
	master.worldControls(&platform.Input{Chars: []rune{'n'}}, 1024, 768)
	if master.shop.npcID != manager.ID {
		t.Fatal("guild manager not opened")
	}
	x, y := master.guildDirectoryUI.Origin(1024, 768)
	request := func() {
		t.Helper()
		master.worldControls(&platform.Input{MouseX: x + 80, MouseY: y + 290, Pressed: true, MouseDown: true}, 1024, 768)
		master.worldControls(&platform.Input{MouseX: x + 80, MouseY: y + 290, Clicked: true}, 1024, 768)
		if !master.guildCreateUI.Visible {
			t.Fatal("guild creation form missing")
		}
		master.updateGuildCreate(&platform.Input{Chars: []rune(name)}, 1024, 768)
		master.updateGuildCreate(&platform.Input{Keys: []platform.Key{platform.KeyTab}}, 1024, 768)
		master.updateGuildCreate(&platform.Input{Chars: []rune("Seven companions"), Keys: []platform.Key{platform.KeyEnter}}, 1024, 768)
		if !master.guild.createPending {
			t.Fatal("creation request not sent")
		}
	}
	request()
	poll(func() bool {
		for _, a := range apps {
			if a.state.Guild.VoteSerial < 1 {
				return false
			}
		}
		return true
	})
	for _, a := range apps {
		a.updateGuildVote(&platform.Input{}, 1024, 768)
		if !a.guildVoteMessage.Visible || a.state.Guild.ID != 0 {
			t.Fatal("missing vote or premature membership")
		}
	}
	apps[6].updateGuildVote(&platform.Input{Keys: []platform.Key{platform.KeyEscape}}, 1024, 768)
	poll(func() bool {
		for _, a := range apps {
			if a.state.Guild.FailureSerial < 1 {
				return false
			}
		}
		return true
	})
	for _, a := range apps {
		a.updateGuildVote(&platform.Input{}, 1024, 768)
		if a.guildVoteMessage.Visible || a.state.Guild.ID != 0 {
			t.Fatal("declined vote remained active")
		}
	}
	request()
	poll(func() bool {
		for _, a := range apps {
			if a.state.Guild.VoteSerial < 2 {
				return false
			}
		}
		return true
	})
	for _, a := range apps {
		a.updateGuildVote(&platform.Input{}, 1024, 768)
		a.updateGuildVote(&platform.Input{Keys: []platform.Key{platform.KeyEnter}}, 1024, 768)
	}
	poll(func() bool {
		for _, a := range apps {
			if a.state.Guild.ID == 0 || len(a.state.Guild.Members) != 7 {
				return false
			}
		}
		return true
	})
	id := master.state.Guild.ID
	for i, a := range apps {
		if a.state.Guild.ID != id || a.state.Guild.Name != name || (i == 0 && a.state.Guild.Rank != 1) || (i > 0 && a.state.Guild.Rank != 9) {
			t.Fatal("guild creation state mismatch")
		}
	}
	poll(func() bool { return master.state.Details.Money == 9000 })
	for _, a := range apps {
		a.updateGuildVote(&platform.Input{}, 1024, 768)
	}
	leaver := apps[6]
	leaver.shop.npcID = manager.ID
	lx, ly := leaver.guildDirectoryUI.Origin(1024, 768)
	click := func(a *App, cx, cy int) {
		a.worldControls(&platform.Input{MouseX: cx, MouseY: cy, Pressed: true, MouseDown: true}, 1024, 768)
		a.worldControls(&platform.Input{MouseX: cx, MouseY: cy, Clicked: true}, 1024, 768)
	}
	click(leaver, lx+22, ly+325)
	if !leaver.guildConfirm.Visible {
		t.Fatal("leave confirmation missing")
	}
	leaver.updateGuildConfirm(&platform.Input{Keys: []platform.Key{platform.KeyEnter}}, 1024, 768)
	if leaver.state.Guild.ID != id {
		t.Fatal("left guild before acknowledgement")
	}
	poll(func() bool { return leaver.state.Guild.ID == 0 && len(master.state.Guild.Members) == 6 })
	leaver.updateGuildVote(&platform.Input{}, 1024, 768)
	for _, admin := range apps[1:3] {
		for rank := uint8(9); rank > 2; rank-- {
			if err := master.world.Send(proto.GuildRankPacket(admin.state.SelfID, false)); err != nil {
				t.Fatal(err)
			}
			want := rank - 1
			poll(func() bool {
				return master.state.Guild.Members[admin.state.SelfID].Rank == want && admin.state.Guild.Rank == want
			})
		}
	}
	disband := func() {
		t.Helper()
		click(master, x+130, y+325)
		if !master.guildConfirm.Visible {
			t.Fatal("disband confirmation missing")
		}
		master.updateGuildConfirm(&platform.Input{Keys: []platform.Key{platform.KeyEnter}}, 1024, 768)
		if !master.guild.lifecyclePending || master.state.Guild.ID != id {
			t.Fatal("disband not pending confirmation")
		}
	}
	disband()
	poll(func() bool {
		for _, a := range apps[:3] {
			if a.state.Guild.DisbandVoteSerial < 1 {
				return false
			}
		}
		return true
	})
	for _, a := range apps[:3] {
		a.updateGuildVote(&platform.Input{}, 1024, 768)
		if !a.guildVoteMessage.Visible {
			t.Fatal("disband vote missing")
		}
	}
	apps[1].updateGuildVote(&platform.Input{Keys: []platform.Key{platform.KeyEscape}}, 1024, 768)
	poll(func() bool {
		for _, a := range apps[:3] {
			if a.state.Guild.DisbandSerial < 1 {
				return false
			}
		}
		return true
	})
	for _, a := range apps[:3] {
		a.updateGuildVote(&platform.Input{}, 1024, 768)
		if a.guildVoteMessage.Visible || a.state.Guild.ID != id {
			t.Fatal("declined disband changed guild")
		}
	}
	disband()
	poll(func() bool {
		for _, a := range apps[:3] {
			if a.state.Guild.DisbandVoteSerial < 2 {
				return false
			}
		}
		return true
	})
	for _, a := range apps[:3] {
		a.updateGuildVote(&platform.Input{}, 1024, 768)
		a.updateGuildVote(&platform.Input{Keys: []platform.Key{platform.KeyEnter}}, 1024, 768)
	}
	poll(func() bool {
		for _, a := range apps {
			if a.state.Guild.ID != 0 {
				return false
			}
			if _, exists := a.state.Guild.Directory[id]; exists {
				return false
			}
		}
		return true
	})
	for _, a := range apps[:3] {
		a.updateGuildVote(&platform.Input{}, 1024, 768)
		if a.guildVoteMessage.Visible || a.guild.voting {
			t.Fatal("disband vote did not close")
		}
	}
	t.Log("guild creation, member leave, rejected and accepted disband votes confirmed")
}
