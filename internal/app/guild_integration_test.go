//go:build integration

package app

import (
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

func TestGuildDirectoryAgainstRapture(t *testing.T) {
	st, _, _, wait := testworld.Enter(t)
	wait(func() bool { return st.Guild.DirectoryReady })
	for id, guild := range st.Guild.Directory {
		if id == 0 || guild.ID != id || guild.Name == "" {
			t.Fatalf("invalid guild directory entry: %+v", guild)
		}
	}
	t.Logf("received complete Rapture guild directory (%d entries)", len(st.Guild.Directory))
}

func TestGuildInviteRefusedAgainstRapture(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_PARTNER") == "" {
		t.Skip("requires two disposable accounts")
	}
	a, wa, _, waitA := testworld.Enter(t)
	b, _, _, waitB := testworld.EnterAs(t, os.Getenv("SHAIYA_TEST_PARTNER"), os.Getenv("SHAIYA_TEST_PARTNER_PASSWORD"))
	waitA(func() bool { return a.Entities[b.SelfID] != nil })
	waitB(func() bool { return b.Entities[a.SelfID] != nil })
	if a.Guild.ID != 0 {
		t.Fatal("fixture should not belong to a guild")
	}
	// The context menu hides this request for non-admins. Send it explicitly
	// to check that the server's rejection is reflected in the UI.
	if err := wa.Send(proto.GuildInvitePacket(b.SelfID)); err != nil {
		t.Fatal(err)
	}
	app := inputApp()
	app.state, app.world = a, wa
	app.guildMessage, _ = ui.NewMessage(inputTextures{})
	deadline := time.Now().Add(8 * time.Second)
	for a.Guild.ReplySerial == 0 {
		app.pumpWorld()
		if app.state == nil || time.Now().After(deadline) {
			t.Fatal("no guild invitation rejection")
		}
		time.Sleep(10 * time.Millisecond)
	}
	app.updateGuildInvite(&platform.Input{}, 1024, 768)
	if a.Guild.LastReply.Result != 0 || len(a.Chat) == 0 || a.Chat[len(a.Chat)-1].Text != "Guild invitation declined or unavailable." {
		t.Fatalf("incorrect refusal state: %+v %+v", a.Guild.LastReply, a.Chat)
	}
}

func TestGuildInviteAcceptAgainstRapture(t *testing.T) {
	name := os.Getenv("SHAIYA_TEST_GUILD")
	if name == "" || os.Getenv("SHAIYA_TEST_PARTNER") == "" {
		t.Skip("requires disposable guild fixture")
	}
	a, wa, _, waitA := testworld.Enter(t)
	b, wb, _, waitB := testworld.EnterAs(t, os.Getenv("SHAIYA_TEST_PARTNER"), os.Getenv("SHAIYA_TEST_PARTNER_PASSWORD"))
	appA, appB := inputApp(), inputApp()
	appA.state, appA.world = a, wa
	appB.state, appB.world = b, wb
	appB.guildMessage, _ = ui.NewMessage(inputTextures{})
	waitA(func() bool {
		return a.Entities[b.SelfID] != nil && a.Guild.ID != 0 && a.Guild.Rank == 1 && a.Guild.DirectoryReady
	})
	waitB(func() bool { return b.Entities[a.SelfID] != nil && b.Guild.DirectoryReady })
	guildID := a.Guild.ID
	if a.Guild.Name != name || a.Guild.Directory[guildID].Name != name || b.Guild.Directory[guildID].Name != name || a.Guild.Members[a.SelfID].Rank != 1 || b.Guild.ID != 0 {
		t.Fatalf("fixture mismatch: master %+v partner %+v", a.Guild, b.Guild)
	}
	a.Target = b.SelfID
	appA.worldControls(&platform.Input{MouseX: 300, MouseY: 20, RightPressed: true, RightDown: true}, 1024, 768)
	actions := appA.contextActions(b.SelfID)
	row := -1
	for i, action := range actions {
		if action.label == "Invite to guild" {
			row = i
		}
	}
	if !appA.playerContext.open || row < 0 {
		t.Fatal("guild invitation action missing")
	}
	x, y := appA.playerContext.x+5, appA.playerContext.y+row*contextRowHeight+5
	appA.worldControls(&platform.Input{MouseX: x, MouseY: y, Pressed: true, MouseDown: true}, 1024, 768)
	appA.worldControls(&platform.Input{MouseX: x, MouseY: y, Clicked: true}, 1024, 768)
	waitB(func() bool { return b.Guild.InviteSerial == 1 })
	appB.updateGuildInvite(&platform.Input{}, 1024, 768)
	if !appB.guildMessage.Visible || b.Guild.ID != 0 {
		t.Fatal("missing prompt or premature guild membership")
	}
	appB.updateGuildInvite(&platform.Input{Keys: []platform.Key{platform.KeyEscape}}, 1024, 768)
	waitA(func() bool { return a.Guild.ReplySerial == 1 })
	if a.Guild.LastReply.Result != 0 || b.Guild.ID != 0 {
		t.Fatal("guild invitation rejection")
	}
	appA.guildCommand("/guildinvite")
	waitB(func() bool { return b.Guild.InviteSerial == 2 })
	appB.updateGuildInvite(&platform.Input{}, 1024, 768)
	appB.updateGuildInvite(&platform.Input{Keys: []platform.Key{platform.KeyEnter}}, 1024, 768)
	if b.Guild.ID != 0 {
		t.Fatal("guild membership changed before acknowledgement")
	}
	waitB(func() bool { return b.Guild.ID == guildID && len(b.Guild.Members) == 2 })
	waitA(func() bool { return a.Guild.ReplySerial == 2 && len(a.Guild.Members) == 2 })
	if a.Guild.LastReply.Result != 1 || b.Guild.Rank != 9 || b.Guild.Name != name || !a.Guild.Members[b.SelfID].Online {
		t.Fatalf("guild membership mismatch: %+v %+v", a.Guild, b.Guild)
	}
	if a.Entities[b.SelfID].GuildID != guildID || guildCaption(a, a.Entities[b.SelfID].GuildID) != "<"+name+">" {
		t.Fatal("visible member did not acquire confirmed guild caption")
	}
	appA.guildUI, _ = ui.NewGuildPanel(inputTextures{})
	appA.guildConfirm, _ = ui.NewMessage(inputTextures{})
	appA.worldControls(&platform.Input{Chars: []rune{'g'}}, 1024, 768)
	if !appA.guild.open {
		t.Fatal("guild panel did not open")
	}
	appA.worldControls(&platform.Input{}, 1024, 768)
	if len(appA.guild.roster) != 2 || appA.guild.roster[0].ID != a.SelfID || appA.guild.roster[1].ID != b.SelfID {
		t.Fatalf("guild UI roster: %+v", appA.guild.roster)
	}
	x, y = appA.guildUI.Origin(1024, 768)
	appA.worldControls(&platform.Input{MouseX: x + 45, MouseY: y + 120, Pressed: true, MouseDown: true}, 1024, 768)
	appA.worldControls(&platform.Input{MouseX: x + 45, MouseY: y + 120, Clicked: true}, 1024, 768)
	if appA.guild.selected != b.SelfID {
		t.Fatal("guild UI did not select partner")
	}
	if err := wb.Shutdown(10 * time.Second); err != nil {
		t.Fatal(err)
	}
	waitA(func() bool { return !a.Guild.Members[b.SelfID].Online })
	b2, _, _, waitB2 := testworld.EnterAs(t, os.Getenv("SHAIYA_TEST_PARTNER"), os.Getenv("SHAIYA_TEST_PARTNER_PASSWORD"))
	waitB2(func() bool { return b2.Guild.ID == guildID && len(b2.Guild.Members) == 2 && b2.Guild.DirectoryReady })
	waitA(func() bool { return a.Guild.Members[b.SelfID].Online })
	waitA(func() bool { return a.Entities[b.SelfID] != nil })
	if b2.Guild.Name != name || b2.Guild.Rank != 9 {
		t.Fatalf("guild not restored: %+v", b2.Guild)
	}
	click := func(cx int) {
		appA.worldControls(&platform.Input{MouseX: x + cx, MouseY: y + 457, Pressed: true, MouseDown: true}, 1024, 768)
		appA.worldControls(&platform.Input{MouseX: x + cx, MouseY: y + 457, Clicked: true}, 1024, 768)
	}
	click(40)
	if a.Guild.Members[b.SelfID].Rank != 9 {
		t.Fatal("rank changed before server acknowledgement")
	}
	waitA(func() bool { return a.Guild.Members[b.SelfID].Rank == 8 })
	waitB2(func() bool { return b2.Guild.Rank == 8 })
	appA.clock = 1
	click(109)
	waitA(func() bool { return a.Guild.Members[b.SelfID].Rank == 9 })
	waitB2(func() bool { return b2.Guild.Rank == 9 })
	click(178)
	if !appA.guildConfirm.Visible {
		t.Fatal("expel confirmation missing")
	}
	appA.updateGuildConfirm(&platform.Input{Keys: []platform.Key{platform.KeyEscape}}, 1024, 768)
	if _, ok := a.Guild.Members[b.SelfID]; !ok {
		t.Fatal("cancel removed member")
	}
	click(178)
	appA.updateGuildConfirm(&platform.Input{Keys: []platform.Key{platform.KeyEnter}}, 1024, 768)
	if _, ok := a.Guild.Members[b.SelfID]; !ok {
		t.Fatal("expel applied before confirmation")
	}
	waitA(func() bool { _, ok := a.Guild.Members[b.SelfID]; return !ok })
	waitB2(func() bool { return b2.Guild.ID == 0 })
	if member := a.Entities[b.SelfID]; member != nil && member.GuildID != 0 {
		t.Fatal("expelled visible member retained a guild caption")
	}
	t.Log("guild directory, invitations, roster, relog, rank changes and expulsion confirmed")
}

func TestGuildApplicationAgainstRapture(t *testing.T) {
	name := os.Getenv("SHAIYA_TEST_GUILD")
	if name == "" || os.Getenv("SHAIYA_TEST_PARTNER") == "" {
		t.Skip("requires disposable guild fixture")
	}
	a, wa, _, waitA := testworld.Enter(t)
	b, wb, _, waitB := testworld.EnterAs(t, os.Getenv("SHAIYA_TEST_PARTNER"), os.Getenv("SHAIYA_TEST_PARTNER_PASSWORD"))
	appA, appB := inputApp(), inputApp()
	appA.state, appA.world = a, wa
	appB.state, appB.world = b, wb
	appA.guildUI, _ = ui.NewGuildPanel(inputTextures{})
	appB.guildMessage, _ = ui.NewMessage(inputTextures{})
	appB.guildDirectoryUI, _ = ui.NewGuildDirectory(inputTextures{})
	waitA(func() bool { return a.Guild.Rank == 1 && a.Guild.DirectoryReady })
	waitB(func() bool { return b.Guild.DirectoryReady })
	guildID := a.Guild.ID
	if b.Guild.ID != 0 || b.Guild.Directory[guildID].Name != name {
		t.Fatal("application fixture unavailable")
	}
	poll := func(app *App, cond func() bool) {
		t.Helper()
		deadline := time.Now().Add(10 * time.Second)
		for !cond() {
			app.pumpWorld()
			if app.state == nil || time.Now().After(deadline) {
				t.Fatal("guild application timed out")
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
	appA.worldControls(&platform.Input{Chars: []rune{'g'}}, 1024, 768)
	x, y := appA.guildUI.Origin(1024, 768)
	click := func(cx, cy int) {
		appA.worldControls(&platform.Input{MouseX: x + cx, MouseY: y + cy, Pressed: true, MouseDown: true}, 1024, 768)
		appA.worldControls(&platform.Input{MouseX: x + cx, MouseY: y + cy, Clicked: true}, 1024, 768)
	}
	click(240, 50)
	if !appA.guild.waiting {
		t.Fatal("waiting list not opened")
	}
	var npcID uint32
	waitB(func() bool {
		for id, e := range b.Entities {
			if e.Kind == game.KindNpc && e.NpcType == 11 && e.TypeID == 1 {
				npcID = id
				return true
			}
		}
		return false
	})
	manager := b.Entities[npcID]
	appB.wv.pos = render.V3{manager.X, manager.Y, manager.Z}
	appB.worldControls(&platform.Input{Chars: []rune{'n'}}, 1024, 768)
	if appB.shop.npcID != npcID {
		t.Fatal("did not open the actual guild manager")
	}
	dx, dy := appB.guildDirectoryUI.Origin(1024, 768)
	appB.worldControls(&platform.Input{}, 1024, 768)
	directoryRow := -1
	for i, entry := range appB.guild.directory {
		if entry.ID == guildID {
			directoryRow = i
			break
		}
	}
	if directoryRow < 0 || directoryRow >= ui.GuildDirectoryRows {
		t.Fatal("fixture outside visible guild directory")
	}
	clickDirectory := func(cx, cy int) {
		appB.worldControls(&platform.Input{MouseX: dx + cx, MouseY: dy + cy, Pressed: true, MouseDown: true}, 1024, 768)
		appB.worldControls(&platform.Input{MouseX: dx + cx, MouseY: dy + cy, Clicked: true}, 1024, 768)
	}
	clickDirectory(275, 95+directoryRow*18)
	if appB.guild.directorySelected != guildID {
		t.Fatal("guild directory selection failed")
	}
	clickDirectory(130, 290)
	poll(appB, func() bool { return b.Guild.ApplySerial == 1 })
	poll(appA, func() bool { _, ok := a.Guild.Applicants[b.SelfID]; return ok })
	if b.Guild.ApplyResult != 1 || b.Guild.ID != 0 {
		t.Fatal("application changed membership early")
	}
	click(105, 103)
	if appA.guild.selected != b.SelfID {
		t.Fatal("applicant not selectable")
	}
	click(115, 457) // Reject.
	poll(appA, func() bool { _, ok := a.Guild.Applicants[b.SelfID]; return !ok })
	poll(appB, func() bool { return b.Guild.JoinSerial == 1 })
	if b.Guild.JoinResult != 0 || b.Guild.ID != 0 {
		t.Fatal("rejected application changed membership")
	}
	appB.guildCommand("/guildapply " + name)
	poll(appB, func() bool { return b.Guild.ApplySerial == 2 })
	poll(appA, func() bool { _, ok := a.Guild.Applicants[b.SelfID]; return ok })
	appA.clock = 1
	click(105, 103)
	click(45, 457) // Accept.
	if b.Guild.ID != 0 {
		t.Fatal("application accepted before server confirmation")
	}
	poll(appA, func() bool { _, ok := a.Guild.Members[b.SelfID]; return ok })
	poll(appB, func() bool { return b.Guild.ID == guildID && b.Guild.Rank == 9 })
	if _, ok := a.Guild.Applicants[b.SelfID]; ok {
		t.Fatal("accepted applicant still waiting")
	}
	t.Log("application, waiting list, rejection and acceptance confirmed")
}

func TestGuildDisbandSmallAgainstRapture(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_GUILD") == "" || os.Getenv("SHAIYA_TEST_PARTNER") == "" {
		t.Skip("requires disposable guild")
	}
	a, wa, _, waitA := testworld.Enter(t)
	b, wb, _, waitB := testworld.EnterAs(t, os.Getenv("SHAIYA_TEST_PARTNER"), os.Getenv("SHAIYA_TEST_PARTNER_PASSWORD"))
	appA, appB := inputApp(), inputApp()
	appA.state, appA.world = a, wa
	appB.state, appB.world = b, wb
	appA.guildDirectoryUI, _ = ui.NewGuildDirectory(inputTextures{})
	appA.guildConfirm, _ = ui.NewMessage(inputTextures{})
	appA.guildVoteMessage, _ = ui.NewMessage(inputTextures{})
	waitA(func() bool {
		return a.Guild.ID != 0 && a.Guild.Rank == 1 && a.Guild.DirectoryReady && len(a.Guild.Members) == 2
	})
	waitB(func() bool { return b.Guild.ID == a.Guild.ID && b.Guild.DirectoryReady })
	id := a.Guild.ID
	var manager *game.Entity
	waitA(func() bool {
		for _, e := range a.Entities {
			if e.Kind == game.KindNpc && e.NpcType == 11 && e.TypeID == 1 {
				manager = e
				return true
			}
		}
		return false
	})
	appA.wv.pos = render.V3{manager.X, manager.Y, manager.Z}
	appA.worldControls(&platform.Input{Chars: []rune{'n'}}, 1024, 768)
	if appA.shop.npcID != manager.ID {
		t.Fatal("guild manager not opened")
	}
	x, y := appA.guildDirectoryUI.Origin(1024, 768)
	appA.worldControls(&platform.Input{MouseX: x + 130, MouseY: y + 325, Pressed: true, MouseDown: true}, 1024, 768)
	appA.worldControls(&platform.Input{MouseX: x + 130, MouseY: y + 325, Clicked: true}, 1024, 768)
	if !appA.guildConfirm.Visible {
		t.Fatal("disband confirmation missing")
	}
	appA.updateGuildConfirm(&platform.Input{Keys: []platform.Key{platform.KeyEnter}}, 1024, 768)
	if a.Guild.ID != id {
		t.Fatal("disband applied before confirmation")
	}
	deadline := time.Now().Add(10 * time.Second)
	for a.Guild.ID != 0 || b.Guild.ID != 0 {
		appA.pumpWorld()
		appB.pumpWorld()
		if appA.state == nil || appB.state == nil || time.Now().After(deadline) {
			t.Fatal("small guild disband timed out")
		}
		time.Sleep(10 * time.Millisecond)
	}
	appA.updateGuildVote(&platform.Input{}, 1024, 768)
	if appA.guild.lifecyclePending || a.Guild.DisbandVoteSerial != 0 || b.Guild.DisbandVoteSerial != 0 {
		t.Fatal("small guild disband should not require a vote")
	}
	t.Log("two-member guild disband confirmed without vote")
}
