package scene

// SkillCastAction is the ready action in Game.exe 0x5780a0. The next ALT
// slot is its casting loop; Animation is the low byte of the SData field.
func SkillCastAction(animation uint16, effect uint8) int {
	if uint8(animation) == 1 {
		return 88
	}
	if effect == 1 {
		return 85
	}
	return 91
}

// SkillAction follows the common branches in 0x57bdc0/0x57e320.
// Skill-specific projectile/impact exceptions are handled separately.
func SkillAction(animation uint16, attack, effect, weapon, variant uint8) int {
	if (effect == 1 || effect == 3 || effect == 5) && attack != 0 {
		if attack == 1 || attack == 2 {
			if a := int(uint8(animation)); a >= 100 && a <= 110 {
				return a
			}
			return AttackAction(weapon, variant)
		}
		if uint8(animation) == 1 {
			return 90
		}
		return 87
	}
	return 93
}

// AttackAction is Game.exe 0x4cb760, using the selected original variant.
func AttackAction(weapon, variant uint8) int {
	v := int(variant % 4)
	switch weaponKind(weapon) {
	case 1, 3, 7:
		return 35 + v
	case 2, 4, 8:
		return 24 + v
	case 5:
		return 42 + v
	case 6:
		return 49 + v
	case 9:
		return 65 + v
	case 10:
		return 79 + v
	case 11, 14:
		return 56
	case 12:
		return 60 + int(variant%2)
	case 13:
		return 31
	case 15:
		return 72 + v
	default:
		return 1
	}
}
