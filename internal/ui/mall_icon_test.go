package ui

import "testing"

func TestMallIconsUseCashAtlasAndRawIndex(t *testing.T) {
	for _, tc := range []struct {
		icon  uint32
		file  string
		index int
	}{{75, "icon/icon_somo2.dds", 74}, {256, "icon/icon_somo2.dds", 255}, {1001, "icon/icon_somo3.dds", 0}} {
		file, uv, ok := MallIcon(tc.icon)
		if !ok || file != tc.file || uv.U0 != float32(tc.index%16)/16 || uv.V0 != float32(tc.index/16)/16 {
			t.Fatal(tc, file, uv, ok)
		}
	}
	for _, icon := range []uint32{0, 999, 1257} {
		if _, _, ok := MallIcon(icon); ok {
			t.Fatal("invalid cash icon accepted", icon)
		}
	}
}
