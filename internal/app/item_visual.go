package app

import (
	"strings"

	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
)

func (a *App) drawItemIcon(d *sdata.Item, x, y, size int, tint [4]float32) {
	if file, uv, ok := ui.ItemIcon(d.Type, d.Icon); ok {
		if tex := a.assets.Texture("interface/" + file); tex != nil {
			a.sprites.Draw(tex, float32(x), float32(y), float32(size), float32(size), uv.U0, uv.V0, uv.U1, uv.V1, tint)
		}
	}
}

// Tooltip data comes from the instance for mutable state and from SData for
// names and base effects. Clamp its dynamically sized frame to the viewport.
func (a *App) drawItemDetails(def *sdata.Item, instance *proto.InventoryItem, count uint8, left, top, w, h int, price string) {
	lines := itemDetailLines(def, instance, count, price, func(id uint8) string {
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
	a.tooltip = rows
	a.tooltipItem = true
	a.tooltipDescription = description
	a.tooltipSockets = nil
	sockets := min(int(def.Slot), 6)
	if instance != nil {
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
		a.tooltipSockets = append(a.tooltipSockets, tooltipSocket{gem})
	}

}
