package game

import (
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"testing"
)

func TestConfirmedGesturesPreservePostureAndCountRepeats(t *testing.T) {
	s := New()
	s.SelfID = 1
	for code := uint8(116); code <= 125; code++ {
		action, ok := proto.GestureAction(code)
		expected := [...]int{117, 116, 119, 118, 121, 120, 123, 122, 124, 125}
		if !ok || action != expected[code-116] {
			t.Fatalf("motion %d: %d/%v", code, action, ok)
		}
		w := proto.NewWriter(proto.CharacterMotion)
		w.U32(2)
		w.U8(code)
		if _, err := s.Apply(w.Bytes()); err != nil {
			t.Fatal(err)
		}
	}
	e := s.Entities[2]
	if e.Gesture != 125 || e.GestureSerial != 10 || e.Sitting {
		t.Fatal("gesture state", e)
	}
	w := proto.NewWriter(proto.CharacterMotion)
	w.U32(2)
	w.U8(125)
	if _, err := s.Apply(w.Bytes()); err != nil || e.GestureSerial != 11 {
		t.Fatal("repeat lost", err, e)
	}
	if _, ok := proto.GestureAction(10); ok {
		t.Fatal("sit is not a gesture")
	}
}
