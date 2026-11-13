package app

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

func TestGuildInvitationPromptAndExpiry(t *testing.T) {
	a := inputApp()
	a.guildMessage, _ = ui.NewMessage(inputTextures{})
	a.state.Entities[5] = &game.Entity{ID: 5, Kind: game.KindUser, Shape: &proto.UserShape{Name: "Ally"}}
	a.state.Guild.InviteID, a.state.Guild.InviteSerial = 5, 1
	if !a.updateGuildInvite(&platform.Input{}, 1024, 768) || !a.guildMessage.Visible || a.guildMessage.Text != "Ally invites you to a guild. Join?" {
		t.Fatal("invitation not presented")
	}
	a.clock = 31
	if a.updateGuildInvite(&platform.Input{}, 1024, 768) || a.guildMessage.Visible || a.state.Guild.InviteID != 0 {
		t.Fatal("expired invitation retained")
	}
	a.state.Guild.InviteID, a.state.Guild.InviteSerial = 5, 2
	a.guild.expires = a.clock + 30
	a.updateGuildInvite(&platform.Input{}, 1024, 768)
	a.state.Guild.ID = 42
	if a.updateGuildInvite(&platform.Input{}, 1024, 768) || a.guildMessage.Visible {
		t.Fatal("prompt remained after guild join")
	}
}

func TestGuildContextInvitationRequiresAdmin(t *testing.T) {
	a := inputApp()
	a.state.SelfID = 1
	a.state.Entities[2] = &game.Entity{ID: 2, Kind: game.KindUser}
	a.state.Guild.ID, a.state.Guild.Rank = 42, 9
	hasInvite := func() bool {
		for _, action := range a.contextActions(2) {
			if action.label == "Invite to guild" {
				return true
			}
		}
		return false
	}
	if hasInvite() {
		t.Fatal("ordinary member can invite")
	}
	a.state.Guild.Rank = 2
	if !hasInvite() {
		t.Fatal("admin missing invitation action")
	}
	a.state.Guild.Members = map[uint32]proto.GuildMember{2: {ID: 2}}
	if hasInvite() {
		t.Fatal("invitation offered to current member")
	}
}

func TestGuildRosterControlsAndCapture(t *testing.T) {
	a := inputApp()
	a.guildUI, _ = ui.NewGuildPanel(inputTextures{})
	a.state.Guild.Members = map[uint32]proto.GuildMember{
		1: {ID: 1, Name: "Member", Rank: 9},
		2: {ID: 2, Name: "Leader", Rank: 1, Online: true},
	}
	controls := a.worldControls(&platform.Input{Chars: []rune{'g'}}, 1024, 768)
	if !a.guild.open || len(controls.Chars) != 0 {
		t.Fatal("G did not open and capture guild panel")
	}
	x, y := a.guildUI.Origin(1024, 768)
	a.worldControls(&platform.Input{MouseX: x + 60, MouseY: y + 116, Pressed: true, MouseDown: true}, 1024, 768)
	controls = a.worldControls(&platform.Input{MouseX: x + 60, MouseY: y + 116, Clicked: true}, 1024, 768)
	if a.guild.selected != 2 || controls.Clicked {
		t.Fatal("guild selection not captured", a.guild.selected)
	}
	controls = a.worldControls(&platform.Input{Keys: []platform.Key{platform.KeyEscape}}, 1024, 768)
	if a.guild.open || controls.HasKey(platform.KeyEscape) {
		t.Fatal("escape did not close guild panel")
	}
}

func TestNativeGuildHeadersSortWithoutChangingSelection(t *testing.T) {
	a := inputApp()
	a.guildUI, _ = ui.NewGuildPanel(inputTextures{})
	a.guild.open, a.guild.rosterDirty, a.guild.selected = true, true, 2
	a.state.Guild.Members = map[uint32]proto.GuildMember{1: {ID: 1, Name: "Zed", Level: 60, Rank: 1}, 2: {ID: 2, Name: "Ally", Level: 20, Rank: 9}}
	x, y := a.guildUI.Origin(1024, 768)
	for _, want := range []uint32{2, 1} {
		a.updateGuild(&platform.Input{MouseX: x + 40, MouseY: y + 75, Pressed: true, MouseDown: true}, 1024, 768)
		a.updateGuild(&platform.Input{MouseX: x + 40, MouseY: y + 75, Clicked: true}, 1024, 768)
		if a.guild.roster[0].ID != want || a.guild.selected != 2 {
			t.Fatal("column sort lost ordering or selected member", a.guild.roster, a.guild.selected)
		}
	}
}

func TestGuildManagementPermissions(t *testing.T) {
	a := inputApp()
	a.state.SelfID = 1
	a.state.Guild.ID, a.state.Guild.Rank = 42, 2
	a.state.Guild.Members = map[uint32]proto.GuildMember{
		1: {ID: 1, Rank: 2}, 2: {ID: 2, Rank: 3}, 3: {ID: 3, Rank: 9}, 4: {ID: 4, Rank: 2},
	}
	if got := a.guildActions(3); got != [3]bool{true, false, true} {
		t.Fatal("rank 9", got)
	}
	if got := a.guildActions(2); got != [3]bool{false, true, true} {
		t.Fatal("admin cannot promote to admin", got)
	}
	if got := a.guildActions(4); got != [3]bool{} {
		t.Fatal("same-rank target", got)
	}
	if got := a.guildActions(1); got != [3]bool{} {
		t.Fatal("self target", got)
	}
	a.state.Guild.Rank = 1
	if got := a.guildActions(2); got != [3]bool{true, true, true} {
		t.Fatal("master", got)
	}
}

func TestGuildDirectorySelectsAndCaptures(t *testing.T) {
	a := inputApp()
	a.guildDirectoryUI, _ = ui.NewGuildDirectory(inputTextures{})
	a.state.Entities[42] = &game.Entity{ID: 42, Kind: game.KindNpc, NpcType: 11}
	a.shop.npcID = 42
	a.state.Guild.DirectoryReady = true
	a.state.Guild.Directory = map[uint32]proto.GuildInfo{7: {ID: 7, Name: "Alpha"}, 8: {ID: 8, Name: "Zeta"}}
	x, y := a.guildDirectoryUI.Origin(1024, 768)
	a.worldControls(&platform.Input{MouseX: x + 275, MouseY: y + 95, Pressed: true, MouseDown: true}, 1024, 768)
	controls := a.worldControls(&platform.Input{MouseX: x + 275, MouseY: y + 95, Clicked: true}, 1024, 768)
	if a.guild.directorySelected != 7 || controls.Clicked {
		t.Fatal("directory selection escaped", a.guild.directorySelected)
	}
}
