//go:build integration

package app

import (
	"fmt"
	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/testworld"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
	"github.com/go-gl/gl/v4.5-core/gl"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestNativeGuildAndQuestWindowsGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL and original assets")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Width: 1024, Height: 768, Hidden: true, Debug: true})
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
	a, err := New(Config{}, arc)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	a.state = game.New()
	a.state.Guild.ID, a.state.Guild.Name, a.state.Guild.Rank = 42, "Guardians", 1
	a.state.Guild.Members = make(map[uint32]proto.GuildMember)
	a.state.Guild.Applicants = make(map[uint32]proto.GuildCandidate)
	for i := uint32(1); i <= 35; i++ {
		a.state.Guild.Members[i] = proto.GuildMember{ID: i, Name: fmt.Sprintf("Member%02d", i), Level: uint16(10 + i), Rank: 9, Class: uint8(i % 6), Online: i%3 != 0}
	}
	a.guild.open, a.guild.rosterDirty, a.guild.selected = true, true, 4
	dir := os.Getenv("SHAIYA_TEST_ARTIFACTS")
	if dir == "" {
		dir = t.TempDir()
	}
	draw := func(name string) {
		t.Helper()
		gl.Viewport(0, 0, 1024, 768)
		gl.ClearColor(.08, .1, .12, 1)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		a.sprites.Begin(1024, 768)
		a.drawGuild(1024, 768)
		a.drawQuestLog(1024, 768)
		a.bottom.GuildOpen, a.bottom.QuestOpen = a.guild.open, a.questLog.open
		a.bottom.Draw(a.sprites, a.font, a.white, 1024, 768, ui.HUDState{}, false, false, false, false)
		if a.guildCreateUI.Visible {
			a.guildCreateUI.Draw(a.sprites, a.font, 1024, 768)
		}
		a.sprites.End()
		if code := gl.GetError(); code != gl.NO_ERROR {
			t.Fatalf("native window GL error %#x", code)
		}
		if err := render.SavePNG(filepath.Join(dir, name+".png"), 1024, 768); err != nil {
			t.Fatal(err)
		}
	}
	draw("native-guild-roster")
	a.guild.waiting, a.guild.rosterDirty = true, true
	a.state.Guild.Applicants[80] = proto.GuildCandidate{ID: 80, Name: "Applicant", Level: 45, Class: 5}
	draw("native-guild-applicants")
	a.guild.waiting, a.guild.rosterDirty = false, true
	a.guildCreateUI.OpenNotice("Guardians", "Welcome to our guild.")
	draw("native-guild-notice")
	a.guildCreateUI.Visible = false
	a.guild.open = false
	a.state.Quests = make(map[uint16]proto.ActiveQuest)
	if err = a.loadQuests(); err != nil {
		t.Fatal(err)
	}
	for i := uint16(1); i <= 10; i++ {
		a.state.Quests[i] = proto.ActiveQuest{ID: i, Counters: [3]uint8{2}}
	}
	a.questLog = questLogView{open: true, selected: 1}
	draw("native-quest-journal")
	a.questLog.offset = 3
	draw("native-quest-scrolled")
}

func TestNativeGuildQuestButtonsAgainstRapture(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_GUILD") == "" {
		t.Skip("requires disposable guild and quest fixture")
	}
	st, world, _, wait := testworld.Enter(t)
	other, _, _, waitB := testworld.EnterAs(t, os.Getenv("SHAIYA_TEST_PARTNER"), os.Getenv("SHAIYA_TEST_PARTNER_PASSWORD"))
	waitB(func() bool { return other.Guild.ID != 0 })
	wait(func() bool { return st.Guild.ID != 0 && len(st.Guild.Members) == 2 })
	a := inputApp()
	a.state, a.world = st, world
	arc, err := archive.Open(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	a.arc = arc
	a.guildUI, _ = ui.NewGuildPanel(inputTextures{})
	a.questUI, _ = ui.NewQuestLog(inputTextures{})
	a.questConfirm, _ = ui.NewMessage(inputTextures{})
	a.bottom, _ = ui.NewBottomBar(inputTextures{})
	click := func(x, y int) {
		a.worldControls(&platform.Input{MouseX: x, MouseY: y, Pressed: true, MouseDown: true}, 1024, 768)
		a.worldControls(&platform.Input{MouseX: x, MouseY: y, Clicked: true}, 1024, 768)
	}
	click(646+5*35+2, 768-54+2)
	if !a.questLog.open || a.guild.open {
		t.Fatal("Quest button opened the wrong window")
	}
	a.updateQuestLog(&platform.Input{}, 1024, 768)
	if _, ok := st.Quests[1]; !ok {
		t.Fatal("disposable quest missing")
	}
	x, y := a.questUI.Origin(1024, 768)
	click(x+410, y+460)
	if !a.questConfirm.Visible {
		t.Fatal("native abandon button did not request confirmation")
	}
	a.updateQuestLog(&platform.Input{Keys: []platform.Key{platform.KeyEnter}}, 1024, 768)
	if _, ok := st.Quests[1]; !ok {
		t.Fatal("quest removed before server confirmation")
	}
	pump := func(cond func() bool) {
		t.Helper()
		until := time.Now().Add(5 * time.Second)
		for !cond() {
			a.pumpWorld()
			if a.state == nil || time.Now().After(until) {
				t.Fatal("guild/quest acknowledgment timeout")
			}
			time.Sleep(time.Millisecond)
		}
	}
	pump(func() bool { _, ok := st.Quests[1]; return !ok })
	click(646+6*35+2, 768-54+2)
	if !a.guild.open || a.questLog.open {
		t.Fatal("Guild button opened the wrong window")
	}
	a.updateGuild(&platform.Input{}, 1024, 768)
	x, y = a.guildUI.Origin(1024, 768)
	row := -1
	for i, m := range a.guild.roster {
		if m.ID == other.SelfID {
			row = i
		}
	}
	if row < 0 {
		t.Fatal("confirmed guild member absent from roster")
	}
	click(x+80, y+112+row*23+2)
	if a.guild.selected != other.SelfID {
		t.Fatal("native roster did not select member")
	}
	click(x+40, y+470)
	if st.Guild.Members[other.SelfID].Rank != 9 {
		t.Fatal("rank predicted before acknowledgment")
	}
	pump(func() bool { return st.Guild.Members[other.SelfID].Rank == 8 })
	t.Log("native Quest/Guild buttons, abandon confirmation and roster promotion confirmed by Rapture")
}
