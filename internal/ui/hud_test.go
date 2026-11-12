package ui

import "testing"

func TestProgressHandlesLevelBoundaries(t *testing.T) {
	for _, tc := range []struct {
		value, start, end uint32
		want              float32
	}{
		{0, 0, 0, 0}, {100, 100, 100, 0}, {100, 200, 100, 0},
		{50, 100, 200, 0}, {100, 100, 200, 0}, {125, 100, 200, .25},
		{200, 100, 200, 1}, {300, 100, 200, 1},
		{^uint32(0), 0, ^uint32(0), 1},
	} {
		if got := progress(tc.value, tc.start, tc.end); got != tc.want {
			t.Errorf("progress(%d,%d,%d)=%g, want %g", tc.value, tc.start, tc.end, got, tc.want)
		}
	}
}
