package effect

// ActivationLibrary is the detailed 0x5768e0 selector, resolved through
// 0x4246c0. Physical skill release uses this body before its target impact.
func ActivationLibrary(skill uint16) string {
	switch skill {
	case 38:
		return "Effect/IN_sd_wind.EFT"
	case 39, 302, 309:
		return "Effect/IN_sb_deathly.EFT"
	case 40, 301, 312:
		return "Effect/IN_sa_round.EFT"
	case 41:
		return "Effect/IN_se_protection.EFT"
	case 55, 307, 310, 311:
		return "Effect/IN_sc_rising.EFT"
	case 57, 78, 147:
		return "Effect/IN_sa_ddamage.EFT"
	case 61, 135, 145:
		return "Effect/IN_sa_gattack.EFT"
	case 62, 76:
		return "Effect/IN_sa_stopblo.EFT"
	case 64:
		return "Effect/IN_sa_vclash.EFT"
	case 79, 150, 161:
		return "Effect/IN_sa_sclash.EFT"
	case 94, 176, 182:
		return "Effect/IN_sa_saiarick.EFT"
	case 104:
		return "Effect/IN_sa_phantom.EFT"
	case 113, 169:
		return "Effect/IN_sa_trap.EFT"
	case 118:
		return "Effect/IN_sa_dragon.EFT"
	case 119:
		return "Effect/IN_sa_power.EFT"
	case 134, 170:
		return "Effect/IN_sa_slasher.EFT"
	case 151, 159:
		return "Effect/IN_sa_hdrein.EFT"
	case 152, 171, 174, 180, 306, 313:
		return "Effect/IN_sa_spirits.EFT"
	case 153, 157:
		return "Effect/IN_sa_wdance.EFT"
	case 154, 177, 183:
		return "Effect/IN_mb_redflare.EFT"
	case 156:
		return "Effect/IN_sd_Mwind.EFT"
	case 158:
		return "Effect/IN_sa_red.EFT"
	case 160:
		return "Effect/IN_sa_black.EFT"
	case 340:
		return "Effect/ExplosiveSting.EFT"
	case 345:
		return "Effect/Elate.EFT"
	case 351:
		return "Effect/StunningDisruptionl.EFT"
	}
	return ""
}
