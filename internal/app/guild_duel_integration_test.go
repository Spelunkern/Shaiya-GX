//go:build integration

package app

import (
	"os"
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/testworld"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

func TestGuildDuelAgainstRapture(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_GUILD") == "" || os.Getenv("SHAIYA_TEST_PARTNER") == "" || os.Getenv("SHAIYA_TEST_GUILD_THIRD") == "" {
		t.Skip("requires two disposable guilds")
	}
	a, wa, _, waitA := testworld.Enter(t)
	b, wb, _, waitB := testworld.EnterAs(t, os.Getenv("SHAIYA_TEST_PARTNER"), os.Getenv("SHAIYA_TEST_PARTNER_PASSWORD"))
	c, wc, sendC, waitC := testworld.EnterAs(t, os.Getenv("SHAIYA_TEST_GUILD_THIRD"), os.Getenv("SHAIYA_TEST_GUILD_THIRD_PASSWORD"))
	appA, appB := inputApp(), inputApp()
	appC := inputApp()
	appC.state, appC.world = c, wc
	appA.state, appA.world = a, wa
	appB.state, appB.world = b, wb
	appA.guildDuelMessage, _ = ui.NewMessage(inputTextures{})
	appB.guildDuelMessage, _ = ui.NewMessage(inputTextures{})
	waitA(func() bool {
		e := a.Entities[b.SelfID]
		return a.Guild.ID != 0 && a.Guild.Rank == 1 && e != nil && e.GuildID != 0
	})
	waitB(func() bool {
		e := b.Entities[a.SelfID]
		return b.Guild.ID != 0 && b.Guild.Rank == 1 && e != nil && e.GuildID != 0
	})
	waitC(func() bool { return c.Guild.ID == b.Guild.ID && c.Guild.Rank == 9 })
	if a.Guild.ID == b.Guild.ID {
		t.Fatal("fixture guilds must differ")
	}
	poll := func(cond func() bool) {
		t.Helper()
		deadline := time.Now().Add(20 * time.Second)
		for !cond() {
			appA.pumpWorld()
			appB.pumpWorld()
			appC.pumpWorld()
			if appA.state == nil || appB.state == nil || appC.state == nil || time.Now().After(deadline) {
				t.Fatal("guild duel timed out")
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
	a.Target = b.SelfID
	appA.worldControls(&platform.Input{MouseX: 300, MouseY: 20, RightPressed: true, RightDown: true}, 1024, 768)
	row := -1
	for i, action := range appA.contextActions(b.SelfID) {
		if action.label == "Guild duel" {
			row = i
		}
	}
	if !appA.playerContext.open || row < 0 {
		t.Fatal("guild duel action missing")
	}
	x, y := appA.playerContext.x+5, appA.playerContext.y+row*contextRowHeight+5
	appA.worldControls(&platform.Input{MouseX: x, MouseY: y, Pressed: true, MouseDown: true}, 1024, 768)
	appA.worldControls(&platform.Input{MouseX: x, MouseY: y, Clicked: true}, 1024, 768)
	poll(func() bool { return a.GuildDuel.Phase == game.DuelOutgoing && b.GuildDuel.Phase == game.DuelIncoming })
	appB.updateGuildDuelInvite(&platform.Input{}, 1024, 768)
	if !appB.guildDuelMessage.Visible {
		t.Fatal("guild duel invitation missing")
	}
	appB.updateGuildDuelInvite(&platform.Input{Keys: []platform.Key{platform.KeyEscape}}, 1024, 768)
	poll(func() bool { return a.GuildDuel.Phase == game.DuelNone })
	if b.GuildDuel.Phase != game.DuelNone || a.GuildDuel.Response.Result != 0 {
		t.Fatal("guild duel rejection failed")
	}
	appA.guildDuelCommand("/guildduel")
	poll(func() bool {
		return a.GuildDuel.Phase == game.DuelOutgoing && b.GuildDuel.Phase == game.DuelIncoming && b.GuildDuel.RequestSerial == 2
	})
	appB.updateGuildDuelInvite(&platform.Input{}, 1024, 768)
	appB.updateGuildDuelInvite(&platform.Input{Keys: []platform.Key{platform.KeyEnter}}, 1024, 768)
	poll(func() bool {
		return a.GuildDuel.Phase == game.DuelCounting && b.GuildDuel.Phase == game.DuelCounting && c.GuildDuel.Phase == game.DuelCounting
	})
	if a.GuildDuel.OpponentGuild != b.Guild.ID || b.GuildDuel.OpponentGuild != a.Guild.ID || !a.GuildDuel.Active || !b.GuildDuel.Active {
		t.Fatal("incorrect guild duel participants")
	}
	poll(func() bool {
		return a.GuildDuel.Phase == game.DuelFighting && b.GuildDuel.Phase == game.DuelFighting && c.GuildDuel.Phase == game.DuelFighting
	})
	sendC(proto.MovePacket(0, 0, 650, c.Self.Y, 1762))
	poll(func() bool { return !c.GuildDuel.Active && c.GuildDuel.LastLeave.Player == c.SelfID })
	if a.GuildDuel.Phase != game.DuelFighting || b.GuildDuel.Phase != game.DuelFighting {
		t.Fatal("non-leader leaving ended the guild duel")
	}
	a.Target = b.SelfID
	appA.updateCombat(&platform.Input{Chars: []rune{'f'}}, 1024, 768)
	poll(func() bool {
		return a.GuildDuel.Phase == game.DuelNone && b.GuildDuel.Phase == game.DuelNone && c.GuildDuel.Phase == game.DuelNone && b.Self.Dead
	})
	if a.GuildDuel.Outcome != 1 || b.GuildDuel.Outcome != 2 || c.GuildDuel.Outcome != 2 || a.GuildDuel.Active || b.GuildDuel.Active {
		t.Fatal("guild duel result mismatch")
	}
	if !b.DeathLossFree || c.DeathLossFree {
		t.Fatalf("personal death protection confused with guild result: dead=%v other=%v", b.DeathLossFree, c.DeathLossFree)
	}
	if os.Getenv("SHAIYA_TEST_GUILD_DUEL_RUNE") == "1" {
		arc, err := archive.Open(os.Getenv("SHAIYA_DATA"))
		if err != nil {
			t.Fatal(err)
		}
		defer arc.Close()
		raw, err := arc.ReadFile("Item/Item.SData")
		if err != nil {
			t.Fatal(err)
		}
		appB.items, err = sdata.ParseItems(raw)
		if err != nil || !appB.hasResurrectionRune() {
			t.Fatal("guild duel rune fixture missing", err)
		}
		appB.deathUI, _ = ui.NewDeathDialog(inputTextures{})
		appB.clock = 1
		appB.updateDeath(&platform.Input{}, 1024, 768)
		if appB.wv.death.rune {
			t.Fatal("loss-free guild duel offered to consume the rune")
		}
		clickDeathChoice(appB, 1024/2-128+96)
		poll(func() bool { return !b.Self.Dead })
		if !appB.hasResurrectionRune() || b.DeathLossFree {
			t.Fatal("revival consumed the rune or retained death protection")
		}
	}
	t.Log("guild duel rejection, acceptance, countdown, participant leaving, combat and result confirmed")
}
