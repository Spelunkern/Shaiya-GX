package scene

import (
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"testing"
)

// Expected rows follow the stores at 0x5427a4..0x54281a for cardinal
// input vectors, independent of the renderer's object placement helper.
func TestNativeEffectPlacementAxes(t *testing.T) {
	for _, tc := range []struct {
		forward render.V3
		want    render.M4
	}{
		{render.V3{0, 0, 1}, render.M4{-1, 0, 0, 0, 0, 1, 0, 0, 0, 0, -1, 0, 0, 0, 0, 1}},
		{render.V3{1, 0, 0}, render.M4{0, 0, 1, 0, 0, 1, 0, 0, -1, 0, 0, 0, 0, 0, 0, 1}},
	} {
		got := nativeEffectBasis(render.Basis(render.V3{}, tc.forward, render.V3{0, 1, 0}))
		if got != tc.want {
			t.Fatalf("forward %v: got %v want %v", tc.forward, got, tc.want)
		}
	}
}
