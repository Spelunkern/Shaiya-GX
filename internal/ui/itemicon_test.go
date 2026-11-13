package ui

import "testing"

func TestItemIconNativeFamiliesAndAtlases(t *testing.T) {
	for _, tc := range []struct {
		typ, icon   uint8
		file        string
		u, v, width float32
	}{
		{1, 1, "01_dds.png", 0, 0, .25},
		{19, 6, "19_dds.png", .25, 1.0 / 16, .25},
		{49, 101, "105_dds.png", 0, 0, .25},
		{72, 5, "16_dds.png", 0, 1.0 / 16, .25},
		{92, 64, "36_dds.png", .75, 15.0 / 16, .25},
		{30, 101, "icon_rapis_dds.png", .5, 12.0 / 16, .125},
		{95, 128, "icon_rapis_dds.png", .875, 15.0 / 16, .125},
		{29, 9, "icon_quest_dds.png", 0, 1.0 / 16, .125},
		{99, 9, "icon_quest2_dds.png", 0, 1.0 / 16, .125},
		{42, 17, "icon_somo_dds.png", 0, 1.0 / 16, 1.0 / 16},
		{80, 17, "icon_somo1_dds.png", 0, 1.0 / 16, 1.0 / 16},
		{101, 1, "icon_somo2_dds.png", 0, 0, 1.0 / 16},
		{103, 1, "icon_somo3_dds.png", 0, 0, 1.0 / 16},
	} {
		file, uv, ok := ItemIcon(tc.typ, tc.icon)
		if !ok || file != "icon/"+tc.file || uv.U0 != tc.u || uv.V0 != tc.v || uv.U1-uv.U0 != tc.width {
			t.Errorf("type=%d icon=%d: %s %+v %v", tc.typ, tc.icon, file, uv, ok)
		}
	}
	for _, tc := range [][2]uint8{{1, 0}, {1, 65}, {1, 165}, {27, 101}, {200, 1}} {
		if _, _, ok := ItemIcon(tc[0], tc[1]); ok {
			t.Errorf("accepted unsupported icon %v", tc)
		}
	}
}
