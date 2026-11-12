package ui

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

func TestStatusDraftLimitsCancellationAndWireOrder(t *testing.T) {
	v, err := NewStatus(marketTextures{})
	if err != nil {
		t.Fatal(err)
	}
	state := StatusState{Points: 3, Base: proto.StatPoints{65535, 10, 10, 10, 10, 10}}
	click := func(x, y int) bool {
		t.Helper()
		v.Update(&platform.Input{MouseX: x, MouseY: y, Pressed: true, MouseDown: true}, 1024, 768, state)
		apply, _ := v.Update(&platform.Input{MouseX: x, MouseY: y, Clicked: true}, 1024, 768, state)
		return apply
	}
	x, y := StatusOrigin(1024, 768, false)
	click(x+1, y+157)
	if !v.Expanded {
		t.Fatal("allocation tab did not expand")
	}
	x, y = StatusOrigin(1024, 768, true)
	click(x+86, y+129) // STR saturated: no point may be wasted.
	if v.Draft.Total() != 0 {
		t.Fatal("allocation overflow")
	}
	click(x+86, y+149) // REC row -> wire index 2.
	click(x+86, y+209) // DEX row -> wire index 1.
	click(x+86, y+209)
	click(x+86, y+209)
	if v.Draft != (proto.StatPoints{0, 2, 1, 0, 0, 0}) || state.Points != 3 || state.Base[1] != 10 {
		t.Fatal("draft/order/budget changed confirmed state", v.Draft, state)
	}
	state.Busy = true
	if click(x+16, y+270) {
		t.Fatal("duplicate apply while pending")
	}
	state.Busy = false
	if !click(x+16, y+270) {
		t.Fatal("valid allocation did not apply")
	}
	click(x+81, y+270)
	if v.Expanded || v.Draft.Total() != 0 {
		t.Fatal("cancel retained unconfirmed points")
	}
}
