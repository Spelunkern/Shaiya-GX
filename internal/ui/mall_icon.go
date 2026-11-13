package ui

// EP5 0x4d8b10 draws Cash.Icon-1 as type 200; 0x4a3400 maps that
// directly to type 100's atlas, or type 102 after index 1000.
func MallIcon(icon uint32) (string, UV, bool) {
	if icon == 0 {
		return "", UV{}, false
	}
	index, typ := icon-1, uint8(100)
	if index >= 1000 {
		index -= 1000
		typ = 102
	}
	if index >= 256 {
		return "", UV{}, false
	}
	name := "icon/icon_somo2_dds.png"
	if typ == 102 {
		name = "icon/icon_somo3_dds.png"
	}
	u, v := float32(index%16)/16, float32(index/16)/16
	return name, UV{u, v, u + 1.0/16, v + 1.0/16}, true
}
