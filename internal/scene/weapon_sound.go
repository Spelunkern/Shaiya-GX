package scene

import "fmt"

// weaponKind is the weapon subset of Game.exe 0x4cba90. Other item types
// are deliberately not mapped to a fallback weapon.
func weaponKind(weapon uint8) uint8 {
	if weapon >= 45 && weapon <= 65 {
		weapon = [...]uint8{1, 2, 3, 4, 5, 5, 6, 6, 7, 7, 8, 8, 9, 10, 11, 12, 12, 13, 13, 14, 15}[weapon-45]
	}
	if weapon > 15 {
		return 0
	}
	return weapon
}

// WeaponSounds follows registration 0x53b9f0 and selectors 0x4101e0/0x410250.
// Normal groups have two variants; criticals use 003. Crossbow has no hit group.
func WeaponSounds(weapon uint8, critical bool, variant uint8) (swing, hit string) {
	kind := weaponKind(weapon)
	if kind == 0 {
		return
	}
	stem := [...]string{"", "swordone", "swordtwo", "axeone", "axetwo", "twin", "spear", "weaponone", "weapontwo", "daggerbk", "dagger", "javelin", "staff", "bow", "crobow", "knuckle"}[kind]
	n := 1 + variant%2
	if critical {
		n = 3
	}
	swing = fmt.Sprintf("ch_att_%s%03d.ogg", stem, n)
	if kind != 14 {
		hit = fmt.Sprintf("ch_hit_%s%03d.ogg", stem, n)
	}
	return
}

// WeaponSwingTime is the original animation threshold (0x41b300), at 30 Hz.
func WeaponSwingTime(weapon uint8) float32 {
	switch weaponKind(weapon) {
	case 13:
		return 1.0 / 30
	case 6:
		return 11.0 / 30
	case 11, 14:
		return 12.0 / 30
	default:
		return 9.0 / 30
	}
}
