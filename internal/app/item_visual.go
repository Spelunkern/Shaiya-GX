package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/element"
	"github.com/Spelunkern/Shaiya-GX/internal/itemtype"
	"regexp"
	"strings"

	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

// effectiveItemElement is shared by the tooltip and every icon presentation.
func (a *App) itemElementState(d *sdata.Item, instance *proto.InventoryItem) (uint8, [6]bool) {
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

// Tooltip data comes from the instance for mutable state and from SData for
// names and base effects. Clamp its dynamically sized frame to the viewport.
func (a *App) drawItemDetails(def *sdata.Item, instance *proto.InventoryItem, count uint16, left, top, w, h int, price string) {
	// A socketed elemental lapis changes the item's effective tooltip element.
	// Keep the cached definition immutable; intrinsic/first-socket priority wins.
	display := *def
	value, inactive := a.itemElementState(def, instance)
	display.Attribute = value
	lines := itemDetailLines(&display, instance, count, price, func(id uint8) string {
		if gem := a.itemsItem(30, id); gem != nil {
			return gem.Name
		}
		return ""
	}, a.msg)

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
	a.tooltipMuted = false
	a.tooltip = rows
	a.tooltipItem = true
	a.tooltipDescription = description
	a.tooltipExtra = tooltipExtras{forgeInfo: itemForgeInfo(def)}
	if rt := itemtype.Real(def.Type); rt >= 1 && rt <= 25 && def.Server > 0 && def.RequiredStats[4] > 0 {
		a.tooltipExtra.maxOJ = min(def.RequiredStats[4], 99)
		if id := def.RequiredStats[5]; id > 0 {
			a.tooltipExtra.material = a.itemsItem(uint8(id/1000), uint8(id%1000))
			if a.tooltipExtra.material != nil {
				a.tooltipDescription = strings.TrimSpace(nativeEnhancementDescription.ReplaceAllString(a.tooltipDescription, ""))
			}
		}
	}
	a.tooltipSockets = nil
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
		a.tooltipSockets = append(a.tooltipSockets, tooltipSocket{definition: gem, inactive: inactive[i]})
	}

}

var nativeEnhancementDescription = regexp.MustCompile(`(?i)(?:\{c[0-9]+\})?\[Enhancement:[^\]]*\](?:\{/c\})?`)
