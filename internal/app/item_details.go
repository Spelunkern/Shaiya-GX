package app

import (
	"fmt"
	"strings"

	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/data/text"
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

// Native 0x4a3950 distinguishes equipment bonuses from consumable restoration.
// SysMsg owns labels; inline description controls remain archive data.
func itemDetailLines(def *sdata.Item, instance *proto.InventoryItem, count uint8, price string, gemName func(uint8) string, messages ...text.SysMsg) []itemDetailLine {
	label := func(id int, fallback string) string {
		if len(messages) > 0 {
			if value := messages[0].Get(id); value != "" {
				return value
			}
		}
		return fallback
	}
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
		name = fmt.Sprintf("%s [%d]", name, craft.enchant)
	}
	add(name, [4]float32{1, 1, 0, 1})
	if def.Attack != 0 || def.AttackAdd != 0 {
		add(fmt.Sprintf("%s %d - %d", label(5191, "attack-power"), def.Attack, uint32(def.Attack)+uint32(def.AttackAdd)), white)
	}
	if def.Def > 0 {
		add(fmt.Sprintf("%s %d", label(5202, "defense-power"), def.Def), white)
	}
	if def.Resist > 0 {
		add(fmt.Sprintf("%s %d", label(5203, "resistence"), def.Resist), white)
	}
	hp, mp, sp := def.RecoveryHP, def.RecoveryMP, def.RecoverySP
	if itemHasRestoration(def.Type) {
		hp, mp, sp = 0, 0, 0
	}
	specs := []struct {
		name   string
		base   uint16
		remake uint16
	}{{label(5208, "max.") + " HP", hp, craft.stats[6]}, {label(5208, "max.") + " SP", sp, craft.stats[8]}, {label(5208, "max.") + " MP", mp, craft.stats[7]}}
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
		add(fmt.Sprintf("%s %d", label(5211, "level"), def.MinLevel), white)
	}

	if price != "" && price != "Sell: 0 gold" {
		color := white
		if strings.HasPrefix(price, "Buy:") {
			price = strings.TrimPrefix(price, "Buy:")
			price = label(5220, "Purchasing price") + " :" + price
		} else if strings.HasPrefix(price, "Sell:") {
			price = strings.TrimPrefix(price, "Sell:")
			price = label(5219, "Selling price") + " :" + price
		}
		price = strings.TrimSuffix(price, " gold")
		add(price, color)
	}
	if instance != nil && def.Quality > 0 {
		add(fmt.Sprintf("%s : %d/%d", label(5221, "endurance"), instance.Quality, def.Quality), white)
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

// These native item families use ConstHP/SP/MP as restoration amounts, not
// equipment max-pool bonuses (0x4a3950). Their descriptions carry that effect.
func itemHasRestoration(typ uint8) bool {
	switch typ {
	case 25, 44, 38, 41, 78, 79, 80, 100, 101, 102, 103:
		return true
	}
	return false
}
