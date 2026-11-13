package scene

// SkillActionSound follows the common cast/release branches in 0x5780a0,
// 0x57bdc0 and 0x57e320. IDs 0x21..0x26 are registered at 0x4264b6..0x42653b.
// Physical attacks use separate weapon/voice cues, not a generic spell sound.
func SkillActionSound(action int) string {
	switch action {
	case 85:
		return "ch_maready01.ogg"
	case 87:
		return "ef_in_firing01.ogg"
	case 88:
		return "ch_maready02.ogg"
	case 90:
		return "ch_maattack02.ogg"
	case 91:
		return "ch_buready01.ogg"
	case 93:
		return "ch_buattack01.ogg"
	}
	return ""
}
