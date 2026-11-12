package game

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

func TestGuildRankingUpdatesAreAtomicAndServerDriven(t *testing.T) {
	s := New()
	s.SelfID = 1
	s.MapID = 5
	s.Guild.Directory = map[uint32]proto.GuildInfo{10: {ID: 10, Name: "Example", Rank: 5, Points: 100}}
	points := proto.NewWriter(proto.GuildRankingPoints)
	points.U32(321)
	points.U32(456)
	points.U32(10)
	if _, err := s.Apply(points.Bytes()[:len(points.Bytes())-1]); err == nil {
		t.Fatal("truncated points accepted")
	}
	if s.Guild.RankingPointsKnown {
		t.Fatal("truncated points changed state")
	}
	if _, err := s.Apply(points.Bytes()); err != nil {
		t.Fatal(err)
	}
	if !s.Guild.RankingPointsKnown || s.Guild.RankingPoints.Points != 321 {
		t.Fatal("points missing")
	}
	list := proto.NewWriter(proto.GuildRankingList)
	list.U8(1)
	list.U32(10)
	list.U32(600)
	list.U8(2)
	if _, err := s.Apply(list.Bytes()); err != nil {
		t.Fatal(err)
	}
	if g := s.Guild.Directory[10]; g.Rank != 2 || g.Points != 600 {
		t.Fatal("directory ranking stale", g)
	}
	invalid := proto.NewWriter(proto.GuildRankingList)
	invalid.U8(2)
	invalid.U32(10)
	invalid.U32(900)
	invalid.U8(1)
	if _, err := s.Apply(invalid.Bytes()); err == nil {
		t.Fatal("partial ranking accepted")
	}
	if s.Guild.Ranking[10].Points != 600 || s.Guild.Directory[10].Points != 600 {
		t.Fatal("partial ranking mutated state")
	}
	duplicate := proto.NewWriter(proto.GuildRankingList)
	duplicate.U8(2)
	for range 2 {
		duplicate.U32(10)
		duplicate.U32(999)
		duplicate.U8(1)
	}
	if _, err := s.Apply(duplicate.Bytes()); err == nil {
		t.Fatal("duplicate guild accepted")
	}
	if s.Guild.Ranking[10].Points != 600 {
		t.Fatal("invalid duplicate ranking mutated state")
	}
	teleport := proto.NewWriter(proto.CharacterMapTeleport)
	teleport.U32(1)
	teleport.U16(6)
	teleport.F32(0)
	teleport.F32(0)
	teleport.F32(0)
	if _, err := s.Apply(teleport.Bytes()); err != nil {
		t.Fatal(err)
	}
	if s.Guild.RankingPointsKnown {
		t.Fatal("battle points leaked to next map")
	}
}
