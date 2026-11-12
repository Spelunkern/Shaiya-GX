package game

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

func TestGuildNoticeAndEtinAreServerConfirmed(t *testing.T) {
	s := New()
	s.Guild.ID = 9
	s.Guild.Directory = map[uint32]proto.GuildInfo{9: {ID: 9, Remark: "old"}}
	s.Guild.Remark = "old"
	s.Guild.RemarkKnown = true
	response := func(result uint8, text string) []byte {
		w := proto.NewWriter(proto.GuildRemark)
		w.U8(result)
		w.Fixed(text, 65, true)
		return w.Bytes()
	}
	bad := response(1, "new")
	if _, err := s.Apply(bad[:len(bad)-1]); err == nil {
		t.Fatal("truncated notice accepted")
	}
	if s.Guild.Remark != "old" || s.Guild.RemarkSerial != 0 {
		t.Fatal("truncated notice changed state")
	}
	if _, err := s.Apply(response(0, "new")); err != nil {
		t.Fatal(err)
	}
	if s.Guild.Remark != "old" || s.Guild.RemarkResult != 0 {
		t.Fatal("refusal changed notice")
	}
	if _, err := s.Apply(response(1, "new")); err != nil {
		t.Fatal(err)
	}
	if s.Guild.Remark != "new" || s.Guild.Directory[9].Remark != "new" {
		t.Fatal("confirmed notice missing")
	}
	etin := proto.NewWriter(proto.GuildEtin)
	etin.U32(123)
	if _, err := s.Apply(etin.Bytes()); err != nil {
		t.Fatal(err)
	}
	if !s.Guild.EtinKnown || s.Guild.Etin != 123 {
		t.Fatal("etin missing")
	}
	leave := proto.NewWriter(proto.GuildLeave)
	leave.U8(1)
	if _, err := s.Apply(leave.Bytes()); err != nil {
		t.Fatal(err)
	}
	if s.Guild.EtinKnown || s.Guild.RemarkKnown {
		t.Fatal("guild data survived departure")
	}
}
