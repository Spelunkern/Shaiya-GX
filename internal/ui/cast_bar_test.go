package ui

import (
	"math"
	"testing"
)

func TestNativeCastProgressColumn(t *testing.T) {
	for _, tc := range []struct {
		left, total float32
		height      int
	}{{3, 3, 0}, {1.5, 3, 31}, {.75, 3, 46}, {.1, 3, 60}, {.001, 3, 62}, {4, 3, 0}} {
		if h, ok := castFillHeight(tc.left, tc.total); !ok || h != tc.height {
			t.Fatalf("remaining %v/%v: %d %v", tc.left, tc.total, h, ok)
		}
	}
	for _, tc := range [][2]float32{{0, 3}, {1, 0}, {-1, 3}, {float32(math.Inf(1)), 3}, {1, float32(math.NaN())}} {
		if _, ok := castFillHeight(tc[0], tc[1]); ok {
			t.Fatal("invalid cast drawn", tc)
		}
	}
}
