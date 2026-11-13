package ui

// Exact byte palette in Game.exe 0x41cfe0 (RGB at message offsets 0x108..0x10a).
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
		return 0xd5d4d3
	case 10:
		return 0xa8f3a0
	case 11:
		return 0xf3a0a0
	case 12:
		return 0xeea4ed
	case 13:
		return 0xf3dea0
	case 14:
		return 0xff3000
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
