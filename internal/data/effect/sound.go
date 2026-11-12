package effect

// ImpactSound resolves Game.exe 0x5761b0 through the 0x4246c0 EFT catalog.
// The water recipe adds the missing .wav suffix found in the original archive.
// Unknown libraries stay silent; Skill.SData.Sound is a separate selector.
func ImpactSound(library string) string {
	switch library {
	case "Effect/DA_target_slumpshot.EFT", "Effect/DA_target_sbomb.EFT", "Effect/DA_target_nuke.EFT",
		"Effect/DA_area_rock.EFT", "Effect/DA_area_earthquake.EFT", "Effect/DA_area_earthpole.EFT":
		return "ef_dmg_earth01.wav"
	case "Effect/DA_self_confusion.EFT":
		return "ef_dmg_confusion01.wav"
	case "Effect/DA_target_valribomb.EFT", "Effect/DA_target_ebomb.EFT":
		return "ef_dmg_boom01.wav"
	case "Effect/DA_area_frozen.EFT":
		return "ef_dmg_water01.wav"
	case "Effect/DA_target_root.EFT":
		return "ef_dmg_hold01.wav"
	case "Effect/DA_target_sonic.EFT", "Effect/DA_area_lightning.EFT":
		return "ef_dmg_light01.wav"
	case "Effect/DA_target_transformation.EFT", "Effect/DA_target_degeneration.EFT", "Effect/DA_target_sirsteindom.EFT":
		return "ef_dmg_transf01.wav"
	case "Effect/DA_area_gale.EFT", "Effect/DA_area_turbulent.EFT":
		return "ef_dmg_wind01.wav"
	case "Effect/DA_area_hailstorm.EFT":
		return "ef_dmg_hail01.wav"
	case "Effect/DA_area_meteor.EFT", "Effect/DA_area_shower.EFT", "Effect/DA_self_kamainb.EFT":
		return "ef_dmg_meteo01.wav"
	case "Effect/DA_area_blizzard.EFT", "Effect/DA_area_bloodshower.EFT", "Effect/ArrowBomberShot.EFT", "Effect/MissileRage.EFT":
		return "ef_dmg_blizzard01.wav"
	}
	return ""
}
