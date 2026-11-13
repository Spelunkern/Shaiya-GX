package ui

import "testing"

func TestNativeMobLevelColorBoundaries(t *testing.T) {
	for _, row := range []struct {
		delta int
		rgb   uint32
	}{{-99, 0xffffff}, {-6, 0xffffff}, {-5, 0x80ffff}, {-4, 0x80ffff}, {-3, 0x0000ff}, {-2, 0x0000ff}, {-1, 0x00ff00}, {0, 0x00ff00}, {1, 0x00ff00}, {2, 0xffff00}, {3, 0xffff00}, {4, 0xff8000}, {5, 0xff8000}, {6, 0xff0000}, {7, 0xff0000}, {8, 0xff00ff}, {9, 0xff00ff}, {10, 0x808080}, {100, 0x808080}} {
		if got := MobNameRGB(uint16(100+row.delta), 100); got != row.rgb {
			t.Fatalf("delta %d: #%06x, want #%06x", row.delta, got, row.rgb)
		}
	}
	if MobNameRGB(0, 65535) != 0xffffff || MobNameRGB(65535, 0) != 0x808080 {
		t.Fatal("unsigned level difference overflowed")
	}
}
