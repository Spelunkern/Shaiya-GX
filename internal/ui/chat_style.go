package ui

// Channel colours follow Phoenix chat_hud.gd. SysMsg notices retain their
// original palette and message IDs.
func ChatRGB(style uint8) uint32 {
	switch style {
	case 0:
		return 0xff9b32
	case 1, 2, 18, 21:
		return 0xff0000
	case 3:
		return 0xffcc00
	case 4:
		return 0x33ff33
	case 5, 23:
		return 0xff00ff
	case 6:
		return 0x99ffff
	case 7:
		return 0xccff33
	case 9:
		return 0xffffff
	case 10:
		return 0xa5f078
	case 11:
		return 0xd65b75
	case 12:
		return 0xefa4d5
	case 13:
		return 0xd7b48d
	case 14:
		return 0x70b9f5
	case 15, 33:
		return 0xffff00
	case 17:
		return 0xcccccc
	case 30:
		return 0xff80ff
	case 31:
		return 0x7875e6
	case 32:
		return 0xff03da
	default:
		return 0xffffff
	}
}
