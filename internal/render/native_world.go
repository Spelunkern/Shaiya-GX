package render

import "math"

// NativeWorldAlphaRef translates ALPHAFUNC=GREATEREQUAL to the shared shader's
// strict-greater discard. Game.exe 0x44867a uses 80 for normal SMOD; 0x5a4090
// uses 8 for dungeon geometry and the MAni pass at 0x44846a.
func NativeWorldAlphaRef(mani bool) float32 {
	value := float32(80) / 255
	if mani {
		value = float32(8) / 255
	}
	return math.Nextafter32(value, 0)
}
