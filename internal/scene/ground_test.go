package scene

import "testing"

func TestGroundItemModel(t *testing.T) {
	for kind, types := range [][]uint8{
		{0, 1, 21, 31, 255},
		{22, 23, 24, 30, 37, 39, 40, 94, 96, 97},
		{25, 27, 28, 29, 38, 41, 44, 78, 79, 80, 99, 100, 101, 102, 103},
		{26},
	} {
		for _, typ := range types {
			if got := GroundItemModel(typ); got != kind {
				t.Errorf("type %d: got %d, want %d", typ, got, kind)
			}
		}
	}
}
