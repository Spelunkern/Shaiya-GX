package effect

// ImpactLibrary resolves the detailed-effects branch of Game.exe 0x577390
// through the catalog at 0x4246c0. An empty name means no library was selected.
// Sequences are skill level minus one, clamped by the native effect loader.
func ImpactLibrary(skill uint16) string {
	switch skill {
	case 29, 34, 37, 38, 39, 40, 41, 45, 55, 57, 64, 67, 80, 93, 137, 150, 152, 154, 164, 165, 166, 167, 281:
		return "Effect/DA_target_generic.EFT"
	case 30:
		return "Effect/DA_target_bash.EFT"
	case 31, 91, 133:
		return "Effect/DA_target_concentration.EFT"
	case 33, 159, 169:
		return "Effect/DA_target_slow.EFT"
	case 35:
		return "Effect/DA_target_root.EFT"
	case 36:
		return "Effect/DA_target_sonic.EFT"
	case 42:
		return "Effect/DA_target_slumber.EFT"
	case 44:
		return "Effect/DA_target_agi.EFT"
	case 46:
		return "Effect/DA_target_long.EFT"
	case 49, 60:
		return "Effect/DA_target_manadrain.EFT"
	case 51, 310, 314, 315:
		return "Effect/DA_target_slumpshot.EFT"
	case 54:
		return "Effect/DA_target_protect.EFT"
	case 58:
		return "Effect/DA_target_burserkfighter.EFT"
	case 59:
		return "Effect/DA_target_hard.EFT"
	case 61:
		return "Effect/DA_target_stun.EFT"
	case 62:
		return "Effect/DA_target_stop.EFT"
	case 63, 97, 107, 172:
		return "Effect/DA_target_degeneration.EFT"
	case 65, 155:
		return "Effect/DA_target_stamina.EFT"
	case 66:
		return "Effect/DA_target_mana.EFT"
	case 73:
		return "Effect/DA_target_iveision.EFT"
	case 74:
		return "Effect/DA_target_healthasis.EFT"
	case 75:
		return "Effect/DA_target_cqguard.EFT"
	case 76:
		return "Effect/DA_target_darkness.EFT"
	case 77, 127, 274, 275:
		return "Effect/DA_target_transformation.EFT"
	case 78:
		return "Effect/DA_target_silence.EFT"
	case 79:
		return "Effect/DA_target_distraction.EFT"
	case 82, 321:
		return "Effect/DA_target_hellfire.EFT"
	case 85:
		return "Effect/DA_area_frozen.EFT"
	case 86:
		return "Effect/DA_target_mirror.EFT"
	case 87, 95:
		return "Effect/DA_target_ebomb.EFT"
	case 88:
		return "Effect/DA_target_heath.EFT"
	case 89:
		return "Effect/DA_target_mattack.EFT"
	case 90:
		return "Effect/DA_target_untouchable.EFT"
	case 92, 94:
		return "Effect/DA_target_dying.EFT"
	case 98:
		return "Effect/DA_target_panic.EFT"
	case 99:
		return "Effect/DA_target_corse.EFT"
	case 101:
		return "Effect/DA_target_misfortune.EFT"
	case 102, 131, 144, 148:
		return "Effect/DA_target_healing.EFT"
	case 103, 130, 173, 326:
		return "Effect/DA_target_dispel.EFT"
	case 106:
		return "Effect/DA_target_sacrifice.EFT"
	case 108:
		return "Effect/DA_target_shotpointer.EFT"
	case 109:
		return "Effect/DA_target_attribute.EFT"
	case 110:
		return "Effect/DA_target_blunt.EFT"
	case 111:
		return "Effect/DA_target_tactical.EFT"
	case 112, 128:
		return "Effect/DA_target_leadership.EFT"
	case 113:
		return "Effect/DA_target_trapbomb.EFT"
	case 115:
		return "Effect/DA_target_manashot.EFT"
	case 116:
		return "Effect/DA_target_aurora.EFT"
	case 117:
		return "Effect/DA_target_support.EFT"
	case 118, 324:
		return "Effect/DA_target_dragon.EFT"
	case 119:
		return "Effect/DA_target_power.EFT"
	case 120:
		return "Effect/DA_target_incantation.EFT"
	case 121:
		return "Effect/DA_target_marksman.EFT"
	case 122, 138:
		return "Effect/DA_target_heavyswat.EFT"
	case 123:
		return "Effect/DA_target_prayer.EFT"
	case 124:
		return "Effect/DA_target_healpole.EFT"
	case 125:
		return "Effect/DA_target_soul.EFT"
	case 126:
		return "Effect/DA_target_resurrection.EFT"
	case 129, 142:
		return "Effect/DA_target_sattack.EFT"
	case 132:
		return "Effect/DA_target_bodyprotect.EFT"
	case 134, 145, 151, 302:
		return "Effect/DA_target_sbomb.EFT"
	case 135:
		return "Effect/DA_target_dizacent.EFT"
	case 136:
		return "Effect/DA_target_dilcritical.EFT"
	case 139, 140:
		return "Effect/DA_target_econtrol.EFT"
	case 141:
		return "Effect/DA_target_meveision.EFT"
	case 143:
		return "Effect/DA_target_nemesys.EFT"
	case 146:
		return "Effect/DA_target_sirsteindom.EFT"
	case 147, 305, 308:
		return "Effect/DA_target_aruzblo.EFT"
	case 149:
		return "Effect/DA_area_disease.EFT"
	case 153, 157, 177:
		return "Effect/DA_target_nuke.EFT"
	case 156:
		return "Effect/DA_target_Mwind.EFT"
	case 160:
		return "Effect/DA_target_dreinge.EFT"
	case 168:
		return "Effect/DA_target_healingprayer.EFT"
	case 170:
		return "Effect/DA_target_atthersing.EFT"
	case 171:
		return "Effect/DA_target_suicidal.EFT"
	case 174, 180, 282, 309, 312:
		return "Effect/DA_target_valribomb.EFT"
	case 175:
		return "Effect/DA_target_pguard.EFT"
	case 176, 306, 311, 313:
		return "Effect/DA_target_tattack.EFT"
	case 178:
		return "Effect/DA_target_bomb.EFT"
	case 181:
		return "Effect/DA_target_dayfly.EFT"
	case 182, 183:
		return "Effect/DA_target_pantera.EFT"
	case 185:
		return "Effect/DA_target_ebreath.EFT"
	case 186, 325:
		return "Effect/DA_target_recover.EFT"
	case 199, 204, 205, 206, 210, 211, 212, 213, 214, 215, 242, 243, 244, 259, 260, 266, 267, 276:
		return "Effect/DA_target_nostrum.EFT"
	case 201, 208, 264:
		return "Effect/DA_target_spremedy.EFT"
	case 202, 207, 256, 257, 258, 263:
		return "Effect/DA_target_hpremedy.EFT"
	case 203, 209, 265:
		return "Effect/DA_target_mpremedy.EFT"
	case 216, 217, 218, 219, 220, 221, 277, 278, 279, 280:
		return "Effect/DA_target_remedy.EFT"
	case 222, 224, 226, 228, 229, 234, 240, 245, 249, 250, 251, 252, 255, 261, 262, 268, 269, 270:
		return "Effect/DA_target_commercialB.EFT"
	case 223, 225, 227, 230, 235, 236:
		return "Effect/DA_target_commercialA.EFT"
	case 307, 322:
		return "Effect/DA_area_turbulent.EFT"
	case 323:
		return "Effect/DA_area_earthquake.EFT"
	case 343:
		return "Effect/MentalImpact.EFT"
	case 345:
		return "Effect/Elate.EFT"
	case 346:
		return "Effect/AbsortionAura.EFT"
	case 348:
		return "Effect/MentalStormLure.EFT"
	case 349:
		return "Effect/MentalStormConfusing.EFT"
	case 350:
		return "Effect/MentalStormDistortion.EFT"
	case 352:
		return "Effect/SoulMenace.EFT"
	case 354:
		return "Effect/DeadlyPoisonShot.EFT"
	case 363:
		return "Effect/CompleteRestore.EFT"
	}
	return ""
}
