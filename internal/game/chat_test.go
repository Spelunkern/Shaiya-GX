package game

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

func TestChatKeepsKnownSenderNamesAfterEntityLeaves(t *testing.T) {
	s := New()
	s.Slots[0] = proto.CharacterSlot{ID: 1, Name: "Self"}
	s.Entities[2] = &Entity{ID: 2, Shape: &proto.UserShape{Name: "Remote"}}
	for _, id := range []uint32{1, 2, 3} {
		w := proto.NewWriter(proto.ChatNormal)
		w.U32(id)
		w.Text("Hello")
		if _, err := s.Apply(w.Bytes()); err != nil {
			t.Fatal(err)
		}
	}
	delete(s.Entities, 2)
	if s.Chat[0].Sender != "Self" || s.Chat[1].Sender != "Remote" || s.Chat[2].Sender != "" {
		t.Fatal("sender identity was lost or invented", s.Chat)
	}
}

func TestChatChannelsUseServerWireLayouts(t *testing.T) {
	s := New()
	for _, op := range []proto.Opcode{proto.ChatWorld, proto.ChatGuild, proto.ChatMap, proto.ChatWorldShout} {
		w := proto.NewWriter(op)
		w.Fixed("Alice", 21, true)
		w.Text("Hello")
		if _, err := s.Apply(w.Bytes()); err != nil {
			t.Fatalf("%x: %v", op, err)
		}
		got := s.Chat[len(s.Chat)-1]
		if got.Sender != "Alice" || got.Text != "Hello" {
			t.Fatalf("%x: %+v", op, got)
		}
	}
	w := proto.NewWriter(proto.ChatShout)
	w.U32(42)
	w.Text("Shout")
	if _, err := s.Apply(w.Bytes()); err != nil {
		t.Fatal(err)
	}
	if got := s.Chat[len(s.Chat)-1]; got.SenderID != 42 || got.Text != "Shout" {
		t.Fatal(got)
	}
	for _, direction := range []uint8{0, 1} {
		w := proto.NewWriter(proto.ChatWhisper)
		w.U8(direction)
		w.Fixed("Bob", 21, true)
		w.Text("Private")
		if _, err := s.Apply(w.Bytes()); err != nil {
			t.Fatal(err)
		}
		got := s.Chat[len(s.Chat)-1]
		if got.Sender != "Bob" || got.Text != "Private" || got.Direction != direction {
			t.Fatal(got)
		}
		before := len(s.Chat)
		if _, err := s.Apply(w.Bytes()[:len(w.Bytes())-1]); err == nil || len(s.Chat) != before {
			t.Fatal("truncated whisper accepted")
		}
	}
	errPacket := proto.NewWriter(proto.ChatError)
	errPacket.U8(1)
	if _, err := s.Apply(errPacket.Bytes()); err != nil {
		t.Fatal(err)
	}
	if got := s.Chat[len(s.Chat)-1].Text; got != "Whisper recipient not found or unavailable." {
		t.Fatal(got)
	}
}
