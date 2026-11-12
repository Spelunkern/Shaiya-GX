package scene

import "testing"

func TestPlayingGestureOnlyCoversActiveSocialSlots(t *testing.T) {
	c := &Character{locked: true, actionSlot: 120}
	if !c.PlayingGesture() {
		t.Fatal("social action not recognized")
	}
	c.actionSlot = 11
	if c.PlayingGesture() {
		t.Fatal("standing transition treated as gesture")
	}
	c.actionSlot = 120
	c.locked = false
	if c.PlayingGesture() {
		t.Fatal("finished gesture still active")
	}
}
