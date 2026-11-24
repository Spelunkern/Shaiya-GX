package app

import (
	"fmt"
	"github.com/Spelunkern/Shaiya-GX/internal/attackspeed"
	"github.com/Spelunkern/Shaiya-GX/internal/element"
	"github.com/Spelunkern/Shaiya-GX/internal/itemtype"
	"regexp"
	"slices"
	"strings"

	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

// effectiveItemElement is shared by the tooltip and every icon presentation.
// withoutCloakGems hides the six emblem bytes of a cape: the game stores them
// in the gem fields, but a cape has no lapis sockets.
func withoutCloakGems(instance *proto.InventoryItem) *proto.InventoryItem {
	if instance == nil || (instance.Type != 24 && instance.Type != 39) || instance.Gems == [6]uint8{} {
		return instance
	}
	c := *instance
	c.Gems = [6]uint8{}
	return &c
}

func (a *App) itemElementState(d *sdata.Item, instance *proto.InventoryItem) (uint8, [6]bool) {
	instance = withoutCloakGems(instance)
	var inactive [6]bool
	if d == nil {
		return 0, inactive
	}
	if instance == nil || itemtype.Real(d.Type) < 1 || itemtype.Real(d.Type) > 25 {
		return element.Normalize(d.Attribute), inactive
	}
	var gems [6]uint8
	for i, id := range instance.Gems {
		if id != 0 {
			if gem := a.itemsItem(30, id); gem != nil {
				gems[i] = element.Normalize(gem.Attribute)
			}
		}
	}
	value, source := element.Resolve(d.Attribute, gems[:])
	for i, g := range gems {
		inactive[i] = g != 0 && source != i
	}
	return value, inactive
}
func (a *App) effectiveItemElement(d *sdata.Item, instance *proto.InventoryItem) uint8 {
	value, _ := a.itemElementState(d, instance)
	return value
}

// Inactive socketed lapis use true luminance grayscale, leaving inventory
// definitions and independently carried gems untouched.
func (a *App) drawSocketLapis(d *sdata.Item, x, y, size int, inactive bool) {
	if !inactive {
		a.drawItemIcon(d, x, y, size, white)
		return
	}
	if file, uv, ok := ui.ItemIcon(d.Type, d.Icon); ok {
		if tex := a.assets.Texture("interface/" + file); tex != nil {
			a.sprites.DrawGrayscale(tex, float32(x), float32(y), float32(size), float32(size), uv.U0, uv.V0, uv.U1, uv.V1, [4]float32{.7, .7, .7, 1})
		}
	}
}
func itemHasElementBadge(d *sdata.Item) bool {
	if d == nil {
		return false
	}
	rt := itemtype.Real(d.Type)
	return rt >= 1 && rt <= 21 || rt == 24 // Weapons, armor/shields and capes; no gems/consumables/jewelry.
}
func (a *App) itemBadge(d *sdata.Item, instance *proto.InventoryItem) ui.ItemBadge {
	if !itemHasElementBadge(d) || a.assets == nil {
		return ui.ItemBadge{}
	}
	uv, ok := ui.ElementUV(a.effectiveItemElement(d, instance))
	if !ok {
		return ui.ItemBadge{}
	}
	return ui.ItemBadge{Tex: a.assets.Texture("interface/enemy_elemental.png"), UV: uv}
}
func (a *App) drawItemIcon(d *sdata.Item, x, y, size int, tint [4]float32, instances ...*proto.InventoryItem) {
	for i := range instances {
		instances[i] = withoutCloakGems(instances[i])
	}
	if file, uv, ok := ui.ItemIcon(d.Type, d.Icon); ok {
		if tex := a.assets.Texture("interface/" + file); tex != nil {
			a.sprites.Draw(tex, float32(x), float32(y), float32(size), float32(size), uv.U0, uv.V0, uv.U1, uv.V1, tint)
			var instance *proto.InventoryItem
			if len(instances) > 0 {
				instance = instances[0]
			}
			a.itemBadge(d, instance).Draw(a.sprites, x, y, size, tint)
		}
	}
}

func (a *App) drawItemDetails(def *sdata.Item, instance *proto.InventoryItem, count uint16, left, top, w, h int, price string) {
	a.setTooltip(a.itemTooltip(def, instance, count, price))
}

// Tooltip data comes from the instance for mutable state and from SData for
// names and base effects.
func (a *App) itemTooltip(def *sdata.Item, instance *proto.InventoryItem, count uint16, price string) tooltipContent {
	instance = withoutCloakGems(instance)
	// A socketed elemental lapis changes the item's effective tooltip element.
	// Keep the cached definition immutable; intrinsic/first-socket priority wins.
	display := *def
	value, inactive := a.itemElementState(def, instance)
	display.Attribute = value
	// Item tooltips describe native and active socket bonuses together. Do not
	// feed these locally derived values into the player's authoritative speeds.
	var lapisAbsorption uint32
	var bonuses lapisTooltipBonuses
	if a.state != nil {
		bonuses.enchantRules = a.state.EnchantRules
	}
	if instance != nil && itemtype.Real(def.Type) >= 1 && itemtype.Real(def.Type) <= 25 {
		move, attackSpeed := int(def.Speed), int(def.AttackTime)
		attackMin, attackAdd := uint32(def.Attack), uint32(def.AttackAdd)
		hp, sp, mp := uint32(def.RecoveryHP), uint32(def.RecoverySP), uint32(def.RecoveryMP)
		var stats [6]uint32
		for k := range 6 {
			stats[k] = uint32(def.Stats[k])
		}
		for i, id := range instance.Gems {
			if id == 0 || inactive[i] {
				continue
			}
			if g := a.itemsItem(30, id); g != nil {
				lapisAbsorption += uint32(g.ExpAbsorb)
				bonuses.def += uint32(g.Def)
				bonuses.resist += uint32(g.Resist)
				move += int(g.Speed)
				attackSpeed += int(g.AttackTime)
				attackMin += uint32(g.Attack)
				attackAdd += uint32(g.AttackAdd)
				hp += uint32(g.RecoveryHP)
				sp += uint32(g.RecoverySP)
				mp += uint32(g.RecoveryMP)
				for k := range 6 {
					stats[k] += uint32(g.Stats[k])
				}
			}
		}
		display.Speed = uint8(min(move, 255))
		display.AttackTime = uint8(min(attackSpeed, 255))
		display.Attack = uint16(min(attackMin, 0xffff))
		display.AttackAdd = uint16(min(attackAdd, 0xffff))
		if !itemHasRestoration(def.Type) {
			display.RecoveryHP = uint16(min(hp, 0xffff))
			display.RecoverySP = uint16(min(sp, 0xffff))
			display.RecoveryMP = uint16(min(mp, 0xffff))
		}
		for k := range 6 {
			display.Stats[k] = uint16(min(stats[k], 0xffff))
		}
		// Native highlights only when linked lapis raise the minimum attack.
		bonuses.attackChanged = display.Attack != def.Attack
	}
	lines := itemDetailLinesWithBonuses(&display, instance, count, price, func(id uint8) string {
		if gem := a.itemsItem(30, id); gem != nil {
			return gem.Name
		}
		return ""
	}, bonuses, a.msg)

	if a.showCatalogIDs {
		lines = slices.Insert(lines, min(1, len(lines)), itemDetailLine{fmt.Sprintf("ID: %d %d", def.Type, def.TypeID), white})
	}

	if warning, restricted := a.itemClassWarning(def); restricted {
		lines = slices.Insert(lines, min(1, len(lines)), warning)
	}

	if lapisAbsorption > 0 || a.state != nil && a.state.EnchantRules != nil {
		label := a.tr(485)
		base := itemAbsorption(def, instance)
		total := base + lapisAbsorption
		if a.state != nil && a.state.EnchantRules != nil {
			total = itemAbsorption(def, instance, a.state.EnchantRules) + lapisAbsorption
		}
		row := itemDetailLine{fmt.Sprintf("%s %d", label, total), white}
		if base > 0 {
			for i := range lines {
				if lines[i].text == fmt.Sprintf("%s %d", label, base) {
					if total > 0 {
						lines[i] = row
					} else {
						lines = slices.Delete(lines, i, i+1)
					}
					break
				}
			}
		} else if total > 0 {
			lines = slices.Insert(lines, 1, row)
		}
	}

	if rt := itemtype.Real(def.Type); rt >= 1 && rt <= 15 && display.AttackTime != def.AttackTime {
		label := a.msg.Get(5192 + int(attackspeed.Tier(display.AttackTime, 0)))
		if label == "" {
			label = a.tr(12256)
		}
		for i := range lines {
			if lines[i].text == label {
				lines[i].color = [4]float32{1, 1, 0, 1}
			}
		} // Native socket-enhanced row.
	}

	// Description is its own RichText block; sockets follow it in Phoenix.
	description := strings.ReplaceAll(def.Description, `\n`, "\n")
	countDescription := 0
	for _, line := range strings.Split(description, "\n") {
		if strings.TrimSpace(line) != "" {
			countDescription++
		}
	}
	if countDescription > 0 {
		lines = lines[:len(lines)-countDescription]
	}
	rows := lines[:0]
	for _, row := range lines {
		if !strings.HasPrefix(row.text, "Lapis: ") {
			rows = append(rows, row)
		}
	}
	if own, ok := a.selfSlot(); ok {
		for i := range rows {
			if strings.HasPrefix(rows[i].text, a.msg.Get(5211)+" ") && own.Level < def.MinLevel {
				rows[i].color = [4]float32{1, 0, 0, 1}
			}
		}
	}
	content := tooltipContent{tooltip: rows, tooltipItem: true, tooltipDescription: description}
	rt := itemtype.Real(def.Type)
	effects, immunities := a.itemEffects(def, instance)
	if itemtype.IsAntiDebuffLapis(def.Type, def.TypeID) && len(immunities) == 1 {
		content.tooltipDescription = "Blocks " + immunities[0].Name + " while linked to equipped gear."
	}
	content.tooltipExtra = tooltipExtras{forgeInfo: itemForgeInfo(def), effects: effects, immunities: immunities, effectChance: def.Type == 30 || rt >= 1 && rt <= 15, effectsLabel: a.tr(12264), immuneLabel: a.tr(12265)}
	if rt := itemtype.Real(def.Type); rt >= 1 && rt <= 25 && def.Server > 0 && def.RequiredStats[4] > 0 {
		content.tooltipExtra.maxOJ = min(def.RequiredStats[4], 99)
		pool := uint8(9)
		if rt <= 15 || rt == 22 || rt == 23 || rt == 25 {
			pool = 6
		}
		content.tooltipExtra.maxOJStats = min(def.Server, pool)
		if id := def.RequiredStats[5]; id > 0 {
			content.tooltipExtra.material = a.itemsItem(uint8(id/1000), uint8(id%1000))
			if content.tooltipExtra.material != nil {
				content.tooltipDescription = strings.TrimSpace(nativeEnhancementDescription.ReplaceAllString(content.tooltipDescription, ""))
			}
		}
	}
	sockets := 0
	if rt := itemtype.Real(def.Type); rt >= 1 && rt <= 25 {
		sockets = min(int(def.Slot), 6)
	}
	if instance != nil && itemtype.Real(def.Type) >= 1 && itemtype.Real(def.Type) <= 25 {
		for i, id := range instance.Gems {
			if id != 0 {
				sockets = max(sockets, i+1)
			}
		}
	}
	for i := 0; i < sockets; i++ {
		var gem *sdata.Item
		if instance != nil && instance.Gems[i] != 0 {
			gem = a.itemsItem(30, instance.Gems[i])
		}
		content.tooltipSockets = append(content.tooltipSockets, tooltipSocket{definition: gem, inactive: inactive[i]})
	}
	return content
}

var nativeEnhancementDescription = regexp.MustCompile(`(?i)(?:\{c[0-9]+\})?\[Enhancement:[^\]]*\](?:\{/c\})?`)

// itemEffects resolves only integrated attack/debuff lapis in the NPC namespace.
func (a *App) itemEffects(def *sdata.Item, instance *proto.InventoryItem) (effects, immunities []tooltipItemEffect) {
	if def == nil {
		return nil, nil
	}
	seenEffects, seenImmunities := map[uint16]bool{}, map[uint16]bool{}
	add := func(gem *sdata.Item, defensive bool) {
		if gem == nil || gem.Special == 0 || gem.Special == 16 {
			return
		}
		id := uint16(gem.Special)
		sk, ok := a.skillDefinition(id, 100)
		if !ok || (sk.Effect != 1 && sk.Effect != 3) {
			return
		}
		if itemtype.IsAntiDebuffLapis(gem.Type, gem.TypeID) {
			if !seenImmunities[id] {
				counterpart := a.itemsItem(gem.Type, gem.TypeID-1)
				if counterpart == nil || counterpart.Special != gem.Special {
					counterpart = gem
				}
				immunities = append(immunities, tooltipItemEffect{Skill: sk, fallback: counterpart})
				seenImmunities[id] = true
			}
		} else if (!defensive || gem.Special > 15) && !seenEffects[id] {
			effects = append(effects, tooltipItemEffect{Skill: sk})
			seenEffects[id] = true
		}
	}
	rt := itemtype.Real(def.Type)
	if def.Type == 30 {
		add(def, false)
	}
	if instance != nil && rt >= 1 && rt <= 25 {
		_, inactive := a.itemElementState(def, instance)
		for i, id := range instance.Gems {
			if id != 0 && !inactive[i] {
				add(a.itemsItem(30, id), rt > 15)
			}
		}
	}
	return effects, immunities
}
