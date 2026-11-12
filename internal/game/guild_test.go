package game

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

func TestGuildDirectoryRosterAndStatus(t *testing.T) {
	s := New()
	s.SelfID = 10
	apply := func(p []byte) {
		t.Helper()
		if _, err := s.Apply(p); err != nil {
			t.Fatal(err)
		}
	}
	member := proto.NewWriter(proto.GuildOnlineList)
	member.U8(1)
	member.U32(10)
	member.U8(1)
	member.U16(35)
	member.U8(2)
	member.Fixed("Master", 21, true)
	memberPacket := member.Bytes()
	for i := 2; i < len(memberPacket); i++ {
		if _, err := s.Apply(memberPacket[:i]); err == nil {
			t.Fatalf("accepted truncated member packet %d", i)
		}
		if len(s.Guild.Members) != 0 {
			t.Fatal("partial roster mutation")
		}
	}
	apply(memberPacket)
	if m := s.Guild.Members[10]; m.Name != "Master" || !m.Online || s.Guild.Rank != 1 {
		t.Fatal("member", m)
	}
	apply(proto.NewWriter(proto.GuildListBegin).Bytes())
	list := proto.NewWriter(proto.GuildList)
	list.U8(1)
	list.U32(42)
	list.Fixed("Guild", 25, true)
	list.Fixed("Master", 21, true)
	list.Fixed("Hello", 65, true)
	list.U8(3)
	list.U32(1234)
	packet := list.Bytes()
	for i := 2; i < len(packet); i++ {
		if _, err := s.Apply(packet[:i]); err == nil {
			t.Fatalf("accepted truncated directory packet %d", i)
		}
		if len(s.Guild.Directory) != 0 {
			t.Fatal("partial directory mutation")
		}
	}
	apply(packet)
	apply(proto.NewWriter(proto.GuildListEnd).Bytes())
	if v := s.Guild.Directory[42]; !s.Guild.DirectoryReady || v.Name != "Guild" || v.Points != 1234 || len(s.Guild.Members) != 1 {
		t.Fatal("directory", v)
	}
	status := func(mode uint8) {
		w := proto.NewWriter(proto.GuildStatus)
		w.U8(mode)
		w.U32(10)
		apply(w.Bytes())
	}
	status(105)
	if s.Guild.Members[10].Online {
		t.Fatal("offline update")
	}
	status(204)
	if s.Guild.Rank != 4 {
		t.Fatal("rank update")
	}
	status(44)
	if s.Guild.Members[10].Level != 44 {
		t.Fatal("level update")
	}
	status(102)
	if len(s.Guild.Members) != 0 {
		t.Fatal("leave update")
	}
	invite := proto.NewWriter(proto.GuildInvite)
	invite.U32(20)
	apply(invite.Bytes())
	if s.Guild.InviteID != 20 || s.Guild.InviteSerial != 1 {
		t.Fatal("invitation")
	}
	join := proto.NewWriter(proto.GuildJoinResult)
	join.U8(1)
	join.U32(42)
	join.U8(9)
	join.Fixed("Guild", 25, true)
	apply(join.Bytes())
	if s.Guild.ID != 42 || s.Guild.InviteID != 0 || s.Guild.Rank != 9 {
		t.Fatal("join")
	}
	leave := proto.NewWriter(proto.GuildLeave)
	leave.U8(1)
	apply(leave.Bytes())
	if s.Guild.ID != 0 || s.Guild.Name != "" {
		t.Fatal("left")
	}
}

func TestGuildEventsUpdateVisibleMemberIdentity(t *testing.T) {
	s := New()
	s.Guild.ID = 42
	s.Entities[11] = &Entity{ID: 11, Kind: KindUser}
	s.Entities[12] = &Entity{ID: 12, Kind: KindUser, GuildID: 99}
	member := proto.NewWriter(proto.GuildMemberAdded)
	member.U8(1)
	member.U32(11)
	member.U8(9)
	member.U16(10)
	member.U8(0)
	member.Fixed("Ally", 21, true)
	if _, err := s.Apply(member.Bytes()); err != nil || s.Entities[11].GuildID != 42 {
		t.Fatal("confirmed member not reflected on visible entity", err)
	}
	status := proto.NewWriter(proto.GuildStatus)
	status.U8(103)
	status.U32(11)
	if _, err := s.Apply(status.Bytes()); err != nil || s.Entities[11].GuildID != 0 {
		t.Fatal("expelled member kept guild identity", err)
	}
	s.Entities[11].GuildID = 42
	disband := proto.NewWriter(proto.GuildDisbanded)
	disband.U32(42)
	if _, err := s.Apply(disband.Bytes()); err != nil || s.Entities[11].GuildID != 0 || s.Entities[12].GuildID != 99 {
		t.Fatal("disband affected wrong visible guild", err)
	}
}

func TestGuildApplicantsRejectTruncation(t *testing.T) {
	s := New()
	w := proto.NewWriter(proto.GuildApplicantList)
	w.U8(1)
	w.U32(42)
	w.U16(17)
	w.U8(3)
	w.Fixed("Candidate", 21, true)
	p := w.Bytes()
	for i := 2; i < len(p); i++ {
		if _, err := s.Apply(p[:i]); err == nil || len(s.Guild.Applicants) != 0 {
			t.Fatalf("partial applicant packet accepted at %d", i)
		}
	}
	if _, err := s.Apply(p); err != nil {
		t.Fatal(err)
	}
	if candidate := s.Guild.Applicants[42]; candidate.Name != "Candidate" || candidate.Level != 17 {
		t.Fatal(candidate)
	}
	remove := proto.NewWriter(proto.GuildApplicantRemoved)
	remove.U32(42)
	if _, err := s.Apply(remove.Bytes()); err != nil || len(s.Guild.Applicants) != 0 {
		t.Fatal("applicant not removed", err)
	}
}

func TestGuildCreationVoteRejectsTruncation(t *testing.T) {
	s := New()
	w := proto.NewWriter(proto.GuildCreateAgree)
	w.U32(42)
	w.Fixed("Guild", 25, true)
	w.Fixed("Remark", 65, true)
	p := w.Bytes()
	for i := 2; i < len(p); i++ {
		if _, err := s.Apply(p[:i]); err == nil || s.Guild.VoteSerial != 0 {
			t.Fatalf("partial vote accepted at %d", i)
		}
	}
	if _, err := s.Apply(p); err != nil || s.Guild.VoteSerial != 1 || s.Guild.CreateVote.Master != 42 || s.Guild.CreateVote.Name != "Guild" {
		t.Fatal("vote", err, s.Guild.CreateVote)
	}
}

func TestGuildDisbandVoteAndLeaveAreConfirmed(t *testing.T) {
	s := New()
	s.Guild.ID = 7
	s.Guild.Name = "Example"
	s.Guild.Rank = 9
	w := proto.NewWriter(proto.GuildDisbandAgree)
	w.U32(42)
	p := w.Bytes()
	for i := 2; i < len(p); i++ {
		if _, err := s.Apply(p[:i]); err == nil || s.Guild.DisbandVoteSerial != 0 {
			t.Fatalf("partial vote accepted at %d", i)
		}
	}
	if _, err := s.Apply(p); err != nil || s.Guild.DisbandMaster != 42 || s.Guild.ID != 7 {
		t.Fatal("vote", err)
	}
	w = proto.NewWriter(proto.GuildDisband)
	w.U8(3)
	if _, err := s.Apply(w.Bytes()); err != nil || s.Guild.DisbandResult != 3 || s.Guild.ID != 7 {
		t.Fatal("declined vote", err)
	}
	w = proto.NewWriter(proto.GuildLeave)
	w.U8(1)
	if _, err := s.Apply(w.Bytes()); err != nil || s.Guild.LeaveResult != 1 || s.Guild.ID != 0 {
		t.Fatal("leave", err)
	}
}

func TestGuildDisbandNotificationClearsMembership(t *testing.T) {
	s := New()
	s.Guild.ID = 7
	s.Guild.Name = "Example"
	s.Guild.Rank = 1
	s.Guild.Members = map[uint32]proto.GuildMember{3: {ID: 3}}
	w := proto.NewWriter(proto.GuildStatus)
	w.U8(101)
	w.U32(0)
	if _, err := s.Apply(w.Bytes()); err != nil || s.Guild.ID != 0 || s.Guild.DisbandedSerial != 1 || len(s.Guild.Members) != 0 {
		t.Fatal("disband notification", err)
	}
}
