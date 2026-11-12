package game

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

func TestGuildDuelPhasesAndParticipantLeave(t *testing.T) {
	s := New()
	s.SelfID = 10
	w := proto.NewWriter(proto.GuildDuelRequest)
	w.U32(20)
	w.U32(10)
	p := w.Bytes()
	for i := 2; i < len(p); i++ {
		if _, err := s.Apply(p[:i]); err == nil || s.GuildDuel.Phase != DuelNone {
			t.Fatalf("partial request %d", i)
		}
	}
	if _, err := s.Apply(p); err != nil || s.GuildDuel.Phase != DuelIncoming {
		t.Fatal("request", err)
	}
	w = proto.NewWriter(proto.GuildDuelCountdown)
	w.F32(100)
	w.F32(200)
	w.U32(42)
	p = w.Bytes()
	for i := 2; i < len(p); i++ {
		if _, err := s.Apply(p[:i]); err == nil || s.GuildDuel.Phase != DuelIncoming {
			t.Fatalf("partial countdown %d", i)
		}
	}
	if _, err := s.Apply(p); err != nil || s.GuildDuel.OpponentGuild != 42 || !s.GuildDuel.Active {
		t.Fatal("countdown", err)
	}
	if _, err := s.Apply(proto.NewWriter(proto.GuildDuelStart).Bytes()); err != nil || s.GuildDuel.Phase != DuelFighting {
		t.Fatal("start", err)
	}
	w = proto.NewWriter(proto.GuildDuelLeave)
	w.U8(3)
	w.U32(10)
	if _, err := s.Apply(w.Bytes()); err != nil || s.GuildDuel.Active || s.GuildDuel.Phase != DuelFighting {
		t.Fatal("participant leave", err)
	}
	w = proto.NewWriter(proto.GuildDuelResult)
	w.U8(2)
	if _, err := s.Apply(w.Bytes()); err != nil || s.GuildDuel.Phase != DuelNone || s.GuildDuel.Outcome != 2 {
		t.Fatal("result", err)
	}
}

func TestGuildDuelDeathProtectionDependsOnPersonalEventOrder(t *testing.T) {
	packet := func(op proto.Opcode, kind uint8, id uint32) []byte {
		w := proto.NewWriter(op)
		w.U8(kind)
		w.U32(id)
		return w.Bytes()
	}
	death := func(kind uint8) []byte {
		w := proto.NewWriter(proto.CharacterDeath)
		w.U32(10)
		w.U8(kind)
		w.U32(20)
		return w.Bytes()
	}
	tests := []struct {
		name   string
		before bool
		kind   uint8
		player uint32
		want   bool
	}{
		{"opposing guild", true, 1, 10, true},
		{"mob death", false, 2, 10, false},
		{"outside attacker", false, 1, 10, false},
		{"another member", true, 1, 11, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := New()
			s.SelfID = 10
			s.GuildDuel.Active = true
			s.GuildDuel.Phase = DuelFighting
			leave := packet(proto.GuildDuelLeave, 3, tc.player)
			if tc.before {
				if _, err := s.Apply(leave); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := s.Apply(death(tc.kind)); err != nil {
				t.Fatal(err)
			}
			if !tc.before {
				if _, err := s.Apply(leave); err != nil {
					t.Fatal(err)
				}
			}
			if s.DeathLossFree != tc.want {
				t.Fatalf("loss-free=%v", s.DeathLossFree)
			}
			w := proto.NewWriter(proto.CharacterResurrect)
			w.U32(10)
			w.U8(4)
			w.U32(0)
			w.F32(0)
			w.F32(0)
			w.F32(0)
			if _, err := s.Apply(w.Bytes()); err != nil {
				t.Fatal(err)
			}
			if s.DeathLossFree || s.GuildDuel.deathLeavePending {
				t.Fatal("death protection persisted")
			}
		})
	}
}

func TestPersonalDuelDeathRequiresPlayerDeathAndResets(t *testing.T) {
	for _, tc := range []struct {
		kind uint8
		want bool
	}{{1, true}, {2, false}} {
		s := New()
		s.SelfID = 10
		result := proto.NewWriter(proto.DuelResult)
		result.U8(2)
		if _, err := s.Apply(result.Bytes()); err != nil {
			t.Fatal(err)
		}
		w := proto.NewWriter(proto.CharacterDeath)
		w.U32(10)
		w.U8(tc.kind)
		w.U32(20)
		if _, err := s.Apply(w.Bytes()); err != nil {
			t.Fatal(err)
		}
		if s.DeathLossFree != tc.want {
			t.Fatalf("kind %d loss-free %v", tc.kind, s.DeathLossFree)
		}
	}
}
