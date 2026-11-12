//go:build integration

package app

import (
	"testing"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/testworld"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

func TestStatusAgainstRapture(t *testing.T) {
	st, w, send, wait := testworld.Enter(t)
	a := inputApp()
	a.state, a.world = st, w
	defer a.Close()
	before, _ := a.selfSlot()
	points, hp, sp := st.Details.StatPoint, st.Details.MaxHP, st.Details.MaxSP
	if points < 2 || !st.StatsKnown {
		t.Fatal("fixture requires points and initial server totals")
	}
	a.worldControls(&platform.Input{Chars: []rune{'T'}}, 1024, 768)
	click := func(x, y int) {
		t.Helper()
		a.worldControls(&platform.Input{MouseX: x, MouseY: y, Pressed: true, MouseDown: true}, 1024, 768)
		a.worldControls(&platform.Input{MouseX: x, MouseY: y, Clicked: true}, 1024, 768)
	}
	x, y := ui.StatusOrigin(1024, 768, false)
	click(x+1, y+157)
	if !a.status.open || !a.statusUI.Expanded {
		t.Fatal("Status hotkey/tab did not open allocation")
	}
	x, y = ui.StatusOrigin(1024, 768, true)
	click(x+86, y+149)
	click(x+86, y+209) // REC, DEX; wire delta is DEX, REC.
	want := proto.StatPoints{0, 1, 1, 0, 0, 0}
	if a.statusUI.Draft != want || st.Details.StatPoint != points {
		t.Fatal("draft altered confirmed points")
	}
	click(x+16, y+270)
	unchanged, _ := a.selfSlot()
	if !a.status.pending || unchanged != before || st.Details.StatPoint != points {
		t.Fatal("apply predicted allocation before acknowledgement")
	}
	deadline := time.Now().Add(5 * time.Second)
	for a.status.pending {
		a.pumpWorld()
		if a.state == nil || time.Now().After(deadline) {
			t.Fatal("stat allocation acknowledgement missing")
		}
		time.Sleep(time.Millisecond)
	}
	after, _ := a.selfSlot()
	if st.LastStatPoints != want || st.Details.StatPoint != points-2 || after.Dex != before.Dex+1 || after.Rec != before.Rec+1 {
		t.Fatal("server allocation/order mismatch", st.LastStatPoints, after)
	}
	// Max-energy updates can arrive just after the acknowledgement.
	for st.Details.MaxHP <= hp || st.Details.MaxSP <= sp {
		a.pumpWorld()
		if time.Now().After(deadline) {
			t.Fatal("server maxima did not update")
		}
		time.Sleep(time.Millisecond)
	}
	if a.statusUI.Draft.Total() != 0 {
		t.Fatal("acknowledgement retained editable draft")
	}
	id := st.SelfID
	send(proto.LogoutPacket())
	wait(func() bool { return st.Phase == game.PhaseCharSelect })
	send(proto.SelectCharacterPacket(id))
	wait(func() bool { return st.Phase == game.PhaseInWorld && st.InventoryReady })
	restored, _ := a.selfSlot()
	if restored.Dex != after.Dex || restored.Rec != after.Rec || st.Details.StatPoint != points-2 || !st.StatsKnown {
		t.Fatal("assigned points did not persist on re-entry", restored)
	}
	t.Log("Status UI assigned REC/DEX only after Rapture acknowledgement; maxima and persisted allocation confirmed on re-entry")
	send(proto.LogoutPacket())
	wait(func() bool { return st.Phase == game.PhaseCharSelect })
}
