package world

import (
	"fmt"
	"github.com/Spelunkern/Shaiya-GX/server/internal/db"
	"math/rand/v2"
)

// All recreation runes share Special 62. ReqVg 0 retains the full native roll;
// 101..109 reroll STR..SP individually; 201..209 set that stat to its maximum.
// Existing attributes and enchantment survive a targeted operation. A new stat
// cannot exceed Server's distinct-attribute budget or the equipment's pool.
func recreateWithRune(rune, target *db.ItemDef, item db.Item) (string, error) {
	if target == nil || target.Server == 0 || target.ReqWis <= 0 {
		return "", fmt.Errorf("item not recreatable")
	}
	maxValue := uint32(min(target.ReqWis, 99))
	if rune.ReqVg == 0 {
		return rollCraftName(min(target.Server, 9), uint8(maxValue), rollsStatsOnly(item.Type), item.CraftName), nil
	}
	mode, stat := int(rune.ReqVg)/100, int(rune.ReqVg)%100-1
	if (mode != 1 && mode != 2) || stat < 0 || stat >= 9 {
		return "", fmt.Errorf("unknown recreation rune property")
	}
	pool := 9
	if rollsStatsOnly(item.Type) {
		pool = 6
	}
	if stat >= pool {
		return "", fmt.Errorf("stat is not allowed on this equipment")
	}
	var pairs [10]uint32
	if c, ok := ParseCraftName(item.CraftName); ok {
		pairs = craftPairs(c)
	}
	used := 0
	for _, v := range pairs[:9] {
		if v > 0 {
			used++
		}
	}
	if pairs[stat] == 0 && used >= min(int(target.Server), pool) {
		return "", fmt.Errorf("maximum distinct recreation attributes reached")
	}
	value := maxValue
	if mode == 1 {
		value = rand.Uint32N(maxValue) + 1
	}
	pairs[stat] = value
	return craftString(pairs), nil
}
