package app

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

func TestNPCTypeLabelsFollowOriginalCategories(t *testing.T) {
	for kind, want := range map[uint8]string{1: "Merchant", 2: "GateKeeper", 3: "Blacksmith", 4: "PvPManager", 5: "", 6: "Warehouse", 7: "", 8: "Guard", 9: "", 10: "Apprentice", 11: "GuildMaster", 12: "DeadNpc", 255: ""} {
		if got := npcTypeLabel(kind); got != want {
			t.Errorf("type %d: %q, want %q", kind, got, want)
		}
	}
}

func TestGroundItemNameDoesNotInventGoldAmount(t *testing.T) {
	a := inputApp()
	a.items = map[sdata.ItemKey]*sdata.Item{
		{Type: 26, TypeID: 0}: {Name: "Wrong gold definition"},
		{Type: 25, TypeID: 4}: {Name: "Potion"},
	}
	for _, tc := range []struct {
		item proto.GroundItem
		want string
	}{
		{proto.GroundItem{Type: 26, Count: 1}, "Gold"},
		{proto.GroundItem{Type: 25, TypeID: 4, Count: 5}, "Potion (x5)"},
		{proto.GroundItem{Type: 25, TypeID: 4, Count: 1}, "Potion"},
		{proto.GroundItem{Type: 99, TypeID: 2, Count: 3}, "Item 99/2 (x3)"},
	} {
		if got := a.groundItemName(tc.item); got != tc.want {
			t.Errorf("%+v: %q, want %q", tc.item, got, tc.want)
		}
	}
}

func TestGuildCaptionUsesConfirmedIdentity(t *testing.T) {
	s := game.New()
	s.Guild.ID, s.Guild.Name = 7, "Rangers"
	s.Guild.Directory = map[uint32]proto.GuildInfo{8: {ID: 8, Name: "Guardians"}}
	for _, tc := range []struct {
		id   uint32
		want string
	}{{0, ""}, {7, "<Rangers>"}, {8, "<Guardians>"}, {9, ""}} {
		if got := guildCaption(s, tc.id); got != tc.want {
			t.Errorf("guild %d: %q, want %q", tc.id, got, tc.want)
		}
	}
	delete(s.Guild.Directory, 8)
	if guildCaption(s, 8) != "" {
		t.Fatal("removed directory entry remained visible")
	}
}
