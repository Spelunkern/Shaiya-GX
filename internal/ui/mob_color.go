package ui

// Game.exe 0x4cb250, called by world names (0x44c396) and target HUD
// (0x50c16f): signed monster level minus the local player's level.
func MobNameRGB(monster, player uint16) uint32 {
	difference := int(monster) - int(player)
	switch {
	case difference >= 10:
		return 0x808080
	case difference >= 8:
		return 0xff00ff
	case difference >= 6:
		return 0xff0000
	case difference >= 4:
		return 0xff8000
	case difference >= 2:
		return 0xffff00
	case difference >= -1:
		return 0x00ff00
	case difference >= -3:
		return 0x0000ff
	case difference >= -5:
		return 0x80ffff
	default:
		return 0xffffff
	}
}

func RGBColor(rgb uint32) [4]float32 {
	return [4]float32{float32(rgb>>16&255) / 255, float32(rgb>>8&255) / 255, float32(rgb&255) / 255, 1}
}
