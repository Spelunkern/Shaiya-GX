package effect

// ProjectileLibrary is Game.exe 0x576c00's detailed-effects selection.
func ProjectileLibrary(skill uint16) string {
	switch skill {
	case 29, 137, 192:
		return "Effect/CA_magic.EFT"
	case 34:
		return "Effect/CA_fire.EFT"
	case 37:
		return "Effect/CA_ice.EFT"
	case 45:
		return "Effect/CA_piercing.EFT"
	case 67:
		return "Effect/CA_triple.EFT"
	case 72:
		return "Effect/CA_elemental.EFT"
	case 80:
		return "Effect/DA_target_recover.EFT"
	case 81:
		return "Effect/CA_bomb.EFT"
	case 82:
		return "Effect/CA_hellfire.EFT"
	case 92, 94, 183:
		return "Effect/CA_death.EFT"
	case 93, 177:
		return "Effect/CA_sharp.EFT"
	case 106:
		return "Effect/CA_sacrifice.EFT"
	case 115, 315, 354:
		return "Effect/CA_killer.EFT"
	case 122, 154, 314:
		return "Effect/CA_jairshot.EFT"
	case 149:
		return "Effect/CA_disease.EFT"
	case 162, 163:
		return "Effect/CA_shower.EFT"
	case 164:
		return "Effect/CA_frameQ.EFT"
	case 165:
		return "Effect/CA_earthQ.EFT"
	case 166:
		return "Effect/CA_stormQ.EFT"
	case 167:
		return "Effect/CA_sirgiQ.EFT"
	case 178:
		return "Effect/CA_souanlip.EFT"
	}
	return ""
}
