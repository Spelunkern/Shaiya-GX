package ui

import "fmt"

// ItemIcon follows 0x4a3400 (grid and alternate atlas), 0x4cba90
// (shared item families), and 0x4a1510 (atlas filenames).
func ItemIcon(typ, icon uint8) (string, UV, bool) {
	if icon == 0 {
		return "", UV{}, false
	}
	index, columns := int(icon)-1, 4
	name := ""
	switch typ {
	case 25, 38, 41, 42, 43, 44, 94:
		name, columns = "icon_somo_dds.png", 16
	case 78, 79, 80:
		name, columns = "icon_somo1_dds.png", 16
	case 100, 101:
		name, columns = "icon_somo2_dds.png", 16
	case 102, 103:
		name, columns = "icon_somo3_dds.png", 16
	case 27, 28, 29:
		name, columns = "icon_quest_dds.png", 8
	case 99:
		name, columns = "icon_quest2_dds.png", 8
	case 30, 95:
		name, columns = "icon_rapis_dds.png", 8
	}
	family := itemIconFamily(typ)
	if name == "" {
		if family < 1 || family > 41 {
			return "", UV{}, false
		}
		name = fmt.Sprintf("%02d_dds.png", family)
		if columns == 4 && index >= 100 {
			index -= 100
			name = fmt.Sprintf("1%02d_dds.png", family)
		}
	} else if columns == 8 && index >= 100 && typ != 30 && typ != 95 {
		return "", UV{}, false // Alternate quest atlases are absent from EP5 data.
	}
	if index >= columns*16 {
		return "", UV{}, false
	}
	u, v := float32(index%columns)/float32(columns), float32(index/columns)/16
	return "icon/" + name, UV{u, v, u + 1/float32(columns), v + 1.0/16}, true
}

func itemIconFamily(typ uint8) uint8 {
	switch {
	case typ >= 45 && typ <= 48:
		return typ - 44
	case typ >= 49 && typ <= 56:
		return (typ-49)/2 + 5
	case typ >= 57 && typ <= 59:
		return typ - 48
	case typ >= 60 && typ <= 63:
		return (typ-60)/2 + 12
	case typ >= 64 && typ <= 65:
		return typ - 50
	case typ >= 66 && typ <= 77:
		return (typ-66)%6 + 16
	case typ >= 81 && typ <= 92:
		return (typ-81)%6 + 31
	case typ == 95:
		return 30
	default:
		return typ
	}
}
