// Package element describes the EP5 element IDs and original damage table.
package element

// 0 neutral; 1 Fire, 2 Water, 3 Earth, 4 Wind; 5..8 are the same order at tier II.
func Normalize(value uint8) uint8 {
	if value > 8 {
		return 0
	}
	return value
}

// MobElement follows CGameData::LoadConstMob 0x4c1720: mob base elements are 0..4.
func MobElement(value uint8) uint8 {
	if value > 4 {
		return 0
	}
	return value
}
func Family(value uint8) uint8 {
	value = Normalize(value)
	if value == 0 {
		return 0
	}
	return (value-1)%4 + 1
}
func Tier(value uint8) uint8 {
	value = Normalize(value)
	if value == 0 {
		return 0
	}
	return (value-1)/4 + 1
}

// Resolve applies GX's item priority: an intrinsic element always wins;
// otherwise the first elemental gem in socket order wins. Source -1 means
// intrinsic or neutral; all other elemental sockets are inactive, even repeats.
func Resolve(base uint8, gems []uint8) (value uint8, source int) {
	if base = Normalize(base); base != 0 {
		return base, -1
	}
	for i, g := range gems {
		if g = Normalize(g); g != 0 {
			return g, i
		}
	}
	return 0, -1
}
func Merge(base uint8, gems []uint8) uint8 { value, _ := Resolve(base, gems); return value }

// Multiplier preserves the native out-of-range fallback. Definitions are
// normalized on load; this protects every damage path from malformed indices.
func Multiplier(attacker, defender uint8) float32 {
	if attacker > 8 || defender > 8 {
		return 1
	}
	return damage[attacker][defender]
}

// DamageTable returns an independent copy for documentation/oracle inspection.
func DamageTable() [9][9]float32 { return damage }

// CGameData::m_AttackAttr, raw float32 values at ps_game.exe VA 0x00542260.
var damage = [9][9]float32{
	{1.0, 0.8, 0.8, 0.8, 0.8, 0.7, 0.7, 0.7, 0.7},
	{1.2, 1.0, 0.5, 1.0, 1.4, 1.0, 0.4, 1.0, 1.3},
	{1.2, 1.4, 1.0, 0.5, 1.0, 1.3, 1.0, 0.4, 1.0},
	{1.2, 1.0, 1.4, 1.0, 0.5, 1.0, 1.3, 1.0, 0.4},
	{1.2, 0.5, 1.0, 1.4, 1.0, 0.4, 1.0, 1.3, 1.0},
	{1.3, 1.0, 0.5, 1.0, 1.6, 1.0, 0.5, 1.0, 1.4},
	{1.3, 1.6, 1.0, 0.5, 1.0, 1.4, 1.0, 0.5, 1.0},
	{1.3, 1.0, 1.6, 1.0, 0.5, 1.0, 1.4, 1.0, 0.5},
	{1.3, 0.5, 1.0, 1.6, 1.0, 0.5, 1.0, 1.4, 1.0},
}
