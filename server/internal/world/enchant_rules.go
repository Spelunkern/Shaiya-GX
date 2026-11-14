package world

import "fmt"

const (
	defaultEnchantMaxLevel = 20
	enchantMaxLevel        = 49 // armor encodes 50+level in the two-digit Craftname pair
)

// This is the authoritative enchant configuration. Edit these Go values and
// rebuild/restart gx_world. No ItemEnchant.ini or runtime text is read.
// Step N is the probability of N -> N+1, in successes per million.
var enchantConfig = struct {
	MaxLevel                uint8
	WeaponRate, DefenseRate [enchantMaxLevel + 1]uint32
}{
	MaxLevel:    defaultEnchantMaxLevel,
	WeaponRate:  [enchantMaxLevel + 1]uint32{900000, 800000, 700000, 600000, 400000, 200000, 100000, 80000, 60000, 10000, 6000, 2000, 1000, 600, 200, 200, 200, 200, 200, 200, 0},
	DefenseRate: [enchantMaxLevel + 1]uint32{990000, 980000, 970000, 800000, 600000, 400000, 200000, 100000, 80000, 60000, 10000, 6000, 2000, 1000, 600, 200, 200, 200, 200, 200, 0},
}

// Bonuses are indexed by the resulting enchantment level, starting at zero.
var weaponEnchantAttackBonus = [enchantMaxLevel + 1]uint32{0, 7, 14, 21, 31, 41, 51, 64, 77, 90, 106, 122, 138, 157, 176, 195, 217, 239, 261, 286, 311}
var armorEnchantAbsorption = [enchantMaxLevel + 1]uint32{0, 5, 10, 15, 20, 25, 30, 35, 40, 45, 50, 55, 60, 65, 70, 75, 80, 85, 90, 95, 100}

func validateEnchantRules() error {
	if enchantConfig.MaxLevel > enchantMaxLevel {
		return fmt.Errorf("enchant cap exceeds two-digit Craftname capacity (%d)", enchantMaxLevel)
	}
	for i := 0; i <= int(enchantConfig.MaxLevel); i++ {
		if enchantConfig.WeaponRate[i] > 1000000 || enchantConfig.DefenseRate[i] > 1000000 {
			return fmt.Errorf("enchant step %d exceeds 100%%", i)
		}
	}
	return nil
}
