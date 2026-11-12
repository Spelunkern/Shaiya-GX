package app

import (
	"strings"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

func TestGuildRankingNoticesAndResultsReachPlayer(t *testing.T) {
	a := inputApp()
	for _, tc := range []struct {
		kind     uint8
		contains string
	}{{21, "21:00"}, {100, "opened"}, {101, "10 minutes"}, {102, "1 minute"}} {
		w := proto.NewWriter(proto.GuildRankingNotice)
		w.U16(50)
		w.U8(tc.kind)
		if _, err := a.state.Apply(w.Bytes()); err != nil {
			t.Fatal(err)
		}
		a.confirmedGuildRanking(w.Bytes())
		if len(a.state.Chat) == 0 || !strings.Contains(a.state.Chat[len(a.state.Chat)-1].Text, tc.contains) {
			t.Fatalf("notice %d missing", tc.kind)
		}
	}
	w := proto.NewWriter(proto.GuildRankingList)
	w.U8(0)
	if _, err := a.state.Apply(w.Bytes()); err != nil {
		t.Fatal(err)
	}
	a.guild.directoryDirty = false
	a.confirmedGuildRanking(w.Bytes())
	if !a.guild.directoryDirty || !strings.Contains(a.state.Chat[len(a.state.Chat)-1].Text, "results") {
		t.Fatal("ranking results not published")
	}
}
