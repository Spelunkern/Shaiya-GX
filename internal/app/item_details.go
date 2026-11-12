package app

import (
	"fmt"
	"strings"

	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
)

type craftValues struct {
	stats   [9]uint16
	enchant uint8
}

// Craftname is ten decimal pairs: six attributes, HP/MP/SP in hundreds,
// then enchantment. Armor enchantment is encoded as 50 plus the level.
func parseCraftValues(name string) (craftValues, bool) {
	var out craftValues
	name = strings.TrimRight(name, " ")
	if len(name) != 20 {
		return out, false
	}
	var values [10]uint16
	for n := range values {
		a, b := name[n*2], name[n*2+1]
		if a < '0' || a > '9' || b < '0' || b > '9' {
			return craftValues{}, false
		}
		values[n] = uint16(a-'0')*10 + uint16(b-'0')
	}
	copy(out.stats[:], values[:9])
	for n := 6; n < 9; n++ {
		out.stats[n] *= 100
	}
	switch level := values[9]; {
	case level <= 20:
		out.enchant = uint8(level)
	case level > 50 && level <= 70:
		out.enchant = uint8(level - 50)
	}
	return out, true
}

func craftSummary(name string) string {
	if strings.TrimSpace(name) == "" {
		return "No bonus attributes"
	}
	v, ok := parseCraftValues(name)
	if !ok {
		return "Unavailable"
	}
	names := [9]string{"STR", "DEX", "REC", "INT", "WIS", "LUC", "HP", "MP", "SP"}
	var parts []string
	for i, n := range v.stats {
		if n > 0 {
			parts = append(parts, fmt.Sprintf("%s +%d", names[i], n))
		}
	}
	if len(parts) == 0 {
		return "No bonus attributes"
	}
	return strings.Join(parts, ", ")
}

type itemDetailLine struct {
	text  string
	color [4]float32
}

// Plain-text consumers of Item.SData omit its inline colour controls.
func stripItemColors(text string) string {
	var out strings.Builder
	for i := 0; i < len(text); {
		if strings.HasPrefix(text[i:], "{/c}") {
			i += 4
			continue
		}
		if strings.HasPrefix(text[i:], "{c") {
			end := i + 2
			for end < len(text) && text[end] >= '0' && text[end] <= '9' {
				end++
			}
			if end > i+2 && end < len(text) && text[end] == '}' {
				i = end + 1
				continue
			}
		}
		out.WriteByte(text[i])
		i++
	}
	return out.String()
}

// Property order and remake colours follow Phoenix ItemTooltip.native_rows;
// values remain Rapture/EP5 data, without Azeroth enhancement formulas.
func itemDetailLines(def *sdata.Item, instance *proto.InventoryItem, count uint8, price string, gemName func(uint8) string) []itemDetailLine {
	lines := make([]itemDetailLine, 0, 20)
	add := func(text string, color [4]float32) {
		if text != "" {
			lines = append(lines, itemDetailLine{text, color})
		}
	}
	craft := craftValues{}
	if instance != nil {
		craft, _ = parseCraftValues(instance.Craft)
	}
	name := def.Name
	if craft.enchant > 0 {
		name = fmt.Sprintf("%s +%d", name, craft.enchant)
	}
	add(name, white)
	if def.Attack != 0 || def.AttackAdd != 0 {
		add(fmt.Sprintf("Attack: %d - %d", def.Attack, uint32(def.Attack)+uint32(def.AttackAdd)), white)
	}
	if def.Def > 0 {
		add(fmt.Sprintf("Defense: %d", def.Def), white)
	}
	if def.Resist > 0 {
		add(fmt.Sprintf("Resistance: %d", def.Resist), white)
	}
	specs := []struct {
		name   string
		base   uint16
		remake uint16
	}{{"HP", def.RecoveryHP, craft.stats[6]}, {"MP", def.RecoveryMP, craft.stats[7]}, {"SP", def.RecoverySP, craft.stats[8]}}
	for i, name := range [6]string{"STR", "DEX", "REC", "INT", "WIS", "LUC"} {
		specs = append(specs, struct {
			name   string
			base   uint16
			remake uint16
		}{name, def.Stats[i], craft.stats[i]})
	}
	yellow := [4]float32{1, 1, 0, 1}
	orange := [4]float32{1, 135.0 / 255, 0, 1}
	for _, stat := range specs {
		if stat.base > 0 {
			text := fmt.Sprintf("%s +%d", stat.name, stat.base)
			if stat.remake > 0 {
				text += fmt.Sprintf("  {r}(+%d){/r}", stat.remake)
			}
			add(text, yellow)
		}
	}
	for _, stat := range specs {
		if stat.base == 0 && stat.remake > 0 {
			add(fmt.Sprintf("%s +%d", stat.name, stat.remake), orange)
		}
	}
	if def.MinLevel > 0 {
		add(fmt.Sprintf("Required level: %d", def.MinLevel), white)
	}
	for i, n := range def.RequiredStats {
		if n > 0 {
			add(fmt.Sprintf("Required %s: %d", [6]string{"STR", "DEX", "REC", "INT", "WIS", "LUC"}[i], n), white)
		}
	}
	var classes []string
	for i, flag := range def.Classes {
		if flag != 0 {
			classes = append(classes, [6]string{"Fighter", "Defender", "Ranger", "Archer", "Mage", "Priest"}[i])
		}
	}
	if len(classes) > 0 && len(classes) < 6 {
		add("Classes: "+strings.Join(classes, ", "), white)
	}
	if instance != nil && def.Quality > 0 {
		add(fmt.Sprintf("Durability: %d / %d", instance.Quality, def.Quality), white)
	}
	if price != "" {
		price = strings.Replace(price, "Sell:", "Sell price:", 1)
		price = strings.Replace(price, "Buy:", "Buy price:", 1)
		price = strings.ReplaceAll(price, " gold", " Gold")
		color := white
		if strings.HasPrefix(price, "Buy price:") {
			color = yellow
		}
		add(price, color)
	}
	if count > 1 || def.Count > 1 {
		if def.Count > 1 {
			add(fmt.Sprintf("Stack: %d / %d", count, def.Count), white)
		} else {
			add(fmt.Sprintf("Stack: %d", count), white)
		}
	}
	// Keep names available to nonvisual consumers; the tooltip places sockets
	// after its description, using the confirmed gem instance IDs.
	if instance != nil {
		for _, id := range instance.Gems {
			if id == 0 {
				continue
			}
			name := ""
			if gemName != nil {
				name = gemName(id)
			}
			if name == "" {
				name = fmt.Sprintf("Lapis %d", id)
			}
			add("Lapis: "+name, white)
		}
	}
	for _, line := range strings.Split(strings.ReplaceAll(def.Description, `\n`, "\n"), "\n") {
		if text := strings.TrimSpace(line); text != "" {
			add(text, white)
		}
	}
	return lines
}
