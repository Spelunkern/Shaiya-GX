//go:build integration

package app

import (
	"os"
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/testworld"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

func TestGuildNoticeAndEtinAgainstRapture(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_GUILD") == "" || os.Getenv("SHAIYA_TEST_PARTNER") == "" {
		t.Skip("requires disposable guild")
	}
	a, wa, _, waitA := testworld.Enter(t)
	b, wb, sendB, waitB := testworld.EnterAs(t, os.Getenv("SHAIYA_TEST_PARTNER"), os.Getenv("SHAIYA_TEST_PARTNER_PASSWORD"))
	appA, appB := inputApp(), inputApp()
	appA.state, appA.world = a, wa
	appB.state, appB.world = b, wb
	appA.guildUI, _ = ui.NewGuildPanel(inputTextures{})
	appA.guildCreateUI, _ = ui.NewGuildCreateDialog(inputTextures{})
	waitA(func() bool { return a.Guild.ID != 0 && a.Guild.Rank == 1 && a.Guild.RemarkKnown })
	waitB(func() bool { return b.Guild.ID == a.Guild.ID && b.Guild.Rank == 9 && b.Guild.RemarkKnown })
	poll := func(cond func() bool) {
		t.Helper()
		deadline := time.Now().Add(15 * time.Second)
		for !cond() {
			appA.pumpWorld()
			appB.pumpWorld()
			if appA.state == nil || appB.state == nil || time.Now().After(deadline) {
				t.Fatal("guild notice response timed out")
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
	appA.guild.open = true
	appA.updateGuild(&platform.Input{}, 1024, 768)
	poll(func() bool { return a.Guild.EtinKnown })
	old := a.Guild.Remark
	x, y := appA.guildUI.Origin(1024, 768)
	appA.updateGuild(&platform.Input{MouseX: x + 255, MouseY: y + 470, Pressed: true, MouseDown: true}, 1024, 768)
	appA.updateGuild(&platform.Input{MouseX: x + 255, MouseY: y + 470, Clicked: true}, 1024, 768)
	if !appA.guildCreateUI.Visible || !appA.guildCreateUI.NoticeMode {
		t.Fatal("notice editor did not open")
	}
	appA.guildCreateUI.Remark.Text = "New notice from master"
	appA.updateGuildCreate(&platform.Input{Keys: []platform.Key{platform.KeyEnter}}, 1024, 768)
	if !appA.guild.remarkPending || a.Guild.Remark != old {
		t.Fatal("notice changed before server confirmation")
	}
	poll(func() bool { return !appA.guild.remarkPending && b.Guild.Remark == "New notice from master" })
	if a.Guild.Remark != b.Guild.Remark {
		t.Fatal("members received different notices")
	}
	serial := b.Guild.RemarkSerial
	sendB(proto.GuildRemarkPacket("Unauthorized"))
	poll(func() bool { return b.Guild.RemarkSerial > serial })
	if b.Guild.RemarkResult != 0 || b.Guild.Remark != "New notice from master" {
		t.Fatal("non-master changed notice")
	}
	if err := wa.Shutdown(10 * time.Second); err != nil {
		t.Fatal(err)
	}
	a2, _, _, waitA2 := testworld.Enter(t)
	waitA2(func() bool { return a2.Guild.ID == b.Guild.ID && a2.Guild.RemarkKnown })
	if a2.Guild.Remark != "New notice from master" {
		t.Fatalf("notice did not persist: %q", a2.Guild.Remark)
	}
	t.Logf("Etin %d, master notice, member broadcast, rejection and relog confirmed", a.Guild.Etin)
}
