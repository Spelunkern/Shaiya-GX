package game

import (
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"testing"
)

func TestQuestSnapshotsAndAuthoritativeEvents(t *testing.T) {
	s := New()
	list := proto.NewWriter(proto.QuestList)
	list.U8(1)
	list.U16(42)
	list.U16(15)
	list.Raw([]byte{1, 2, 3})
	p := list.Bytes()
	for i := 2; i < len(p); i++ {
		if _, err := s.Apply(p[:i]); err == nil {
			t.Fatalf("accepted truncated snapshot %d", i)
		}
		if len(s.Quests) != 0 {
			t.Fatal("truncated snapshot mutated state")
		}
	}
	if _, err := s.Apply(p); err != nil {
		t.Fatal(err)
	}
	counter := proto.NewWriter(proto.QuestCounter)
	counter.U16(42)
	counter.U8(1)
	counter.U8(9)
	if _, err := s.Apply(counter.Bytes()); err != nil || s.Quests[42].Counters != [3]uint8{1, 9, 3} {
		t.Fatal("counter", s.Quests, err)
	}
	invalid := proto.NewWriter(proto.QuestCounter)
	invalid.U16(42)
	invalid.U8(3)
	invalid.U8(1)
	if _, err := s.Apply(invalid.Bytes()); err == nil || s.Quests[42].Counters != [3]uint8{1, 9, 3} {
		t.Fatal("invalid counter changed state")
	}
	choose := proto.NewWriter(proto.QuestChoose)
	choose.U16(42)
	if _, err := s.Apply(choose.Bytes()); err != nil || !s.QuestChoosing {
		t.Fatal("choice", err)
	}
	end := proto.NewWriter(proto.QuestEnd)
	end.U32(20)
	end.U16(42)
	end.U8(1)
	end.U8(2)
	end.U32(400)
	end.U32(100)
	end.Raw([]byte{1, 2, 3, 4})
	ep := end.Bytes()
	for i := 2; i < len(ep); i++ {
		if _, err := s.Apply(ep[:i]); err == nil {
			t.Fatalf("accepted truncated result %d", i)
		}
		if len(s.Quests) != 1 {
			t.Fatal("truncated result removed quest")
		}
	}
	if _, err := s.Apply(ep); err != nil || len(s.Quests) != 0 || s.QuestChoosing || s.LastQuestEnd.Experience != 400 {
		t.Fatal("result", s.LastQuestEnd, err)
	}
}
