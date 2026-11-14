package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/text"
	"github.com/Spelunkern/Shaiya-GX/internal/element"
)

// Game.exe 0x41e240: original SysMsg names plus Roman tier (0x5bf240).
func elementName(value uint8, messages ...text.SysMsg) string {
	value = element.Normalize(value)
	family := element.Family(value)
	id := 5270
	name := "non-elemental"
	if family != 0 {
		id = 5203 + int(family)
		name = [4]string{"Fire", "Water", "Earth", "Wind"}[family-1]
	}
	if len(messages) > 0 {
		if translated := messages[0].Get(id); translated != "" {
			name = translated
		}
	}
	if family != 0 {
		name += " " + [2]string{"I", "II"}[element.Tier(value)-1]
	}
	return name
}

// User-selected family colors for skill-element labels, shared across tiers.
func skillElementColor(value uint8) [4]float32 {
	switch element.Family(value) {
	case 1:
		return [4]float32{1, .7, .4, 1}
	case 2:
		return [4]float32{.5, .85, 1, 1}
	case 3:
		return [4]float32{.6, 1, .6, 1}
	case 4:
		return [4]float32{.7, .7, .7, 1}
	}
	return white
}
