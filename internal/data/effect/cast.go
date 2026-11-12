package effect

// CastLibrary is the detailed-effects branch of Game.exe 0x5765d0,
// resolved through the library catalog loaded by 0x4246c0. Preparation
// uses sequence 1, clamped to the library's last sequence (0x4178a0).
func CastLibrary(skill uint16) string {
	switch skill {
	case 29, 60, 88, 102, 103, 137, 143, 178:
		return "Effect/IN_ma_generic.EFT"
	case 33, 62, 76, 78, 97, 147, 173:
		return "Effect/IN_md_black.EFT"
	case 34, 47, 48, 164, 321:
		return "Effect/IN_ma_fire.EFT"
	case 35, 50, 165, 323:
		return "Effect/IN_ma_earth.EFT"
	case 36, 52, 166, 314, 322, 325:
		return "Effect/IN_ma_wind.EFT"
	case 37, 53, 167, 324:
		return "Effect/IN_ma_water.EFT"
	case 42, 65, 66, 79, 98, 99, 100, 101, 108, 109, 110, 125, 146, 155:
		return "Effect/IN_mc_black.EFT"
	case 43:
		return "Effect/IN_md_yellow.EFT"
	case 45, 80, 122:
		return "Effect/IN_mc_white.EFT"
	case 49, 51:
		return "Effect/IN_md_generic.EFT"
	case 68, 82:
		return "Effect/IN_mb_fire.EFT"
	case 69, 83:
		return "Effect/IN_mb_earth.EFT"
	case 70:
		return "Effect/IN_mb_wind.EFT"
	case 71:
		return "Effect/IN_mb_water.EFT"
	case 72, 126, 318:
		return "Effect/IN_mb_generic.EFT"
	case 81, 92, 93, 154, 177, 183:
		return "Effect/IN_mb_redflare.EFT"
	case 84:
		return "Effect/IN_mc_wind.EFT"
	case 85:
		return "Effect/IN_mc_water.EFT"
	case 87:
		return "Effect/IN_mc_suicide.EFT"
	case 95:
		return "Effect/IN_ma_light.EFT"
	case 96, 306, 307:
		return "Effect/IN_mc_generic.EFT"
	case 105:
		return "Effect/IN_mc_taunt.EFT"
	case 115, 149, 315, 354:
		return "Effect/IN_mc_cowardice.EFT"
	case 123, 179, 185, 304, 317, 326:
		return "Effect/IN_ma_white.EFT"
	case 162, 163, 353, 355, 356:
		return "Effect/IN_ma_redflare.EFT"
	case 168, 186:
		return "Effect/IN_mb_white.EFT"
	case 184:
		return "Effect/IN_ma_dark.EFT"
	case 188, 189:
		return "Effect/DA_target_ebreath.EFT"
	case 190, 191:
		return "Effect/DA_target_dayfly.EFT"
	case 201, 202, 203, 207, 208, 209, 216, 217, 218, 219, 220, 221, 256, 257, 258, 263, 264, 265, 273:
		return "Effect/IN_me_generic.EFT"
	case 204, 205, 206, 210, 211, 212, 213, 214, 215, 242, 243, 244, 259, 260, 319:
		return "Effect/IN_mf_generic.EFT"
	case 222, 224, 226, 229, 240, 245, 249, 250, 251, 252, 255, 262, 286:
		return "Effect/IN_mb_commercial.EFT"
	case 223, 225, 227, 228, 230, 234, 235, 236, 261:
		return "Effect/IN_ma_commercial.EFT"
	case 341:
		return "Effect/EnergyBlackhole.EFT"
	case 344:
		return "Effect/Detonation.EFT"
	case 345:
		return "Effect/Elate.EFT"
	case 347:
		return "Effect/DanceOfDeath.EFT"
	case 364:
		return "Effect/FrostBarrier.EFT"
	}
	return "Effect/IN_mc_yellow.EFT"
}
