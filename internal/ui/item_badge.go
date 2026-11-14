package ui

import (
	"github.com/Spelunkern/Shaiya-GX/internal/element"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

// ElementUV shares the original four-symbol monster target atlas. Tier II
// retains its family symbol; the item's tooltip supplies the exact tier.
func ElementUV(value uint8) (UV, bool) {
	family := element.Family(value)
	if family == 0 {
		return UV{}, false
	}
	u := float32(family-1) / 4
	return UV{u, 0, u + .25, 1}, true
}

type ItemBadge struct {
	Tex *render.Texture
	UV  UV
}

// Upper-left placement leaves shortcut keys and lower-right quantities free.
// The insignia scales with the item instead of keeping a fixed screen size.
func ItemBadgeSize(iconSize int) int { return min(iconSize, max(1, min(14, iconSize*7/16))) }
func (b ItemBadge) Draw(s *render.Sprites, x, y, size int, tint [4]float32) {
	if b.Tex == nil || size <= 0 {
		return
	}
	n := float32(ItemBadgeSize(size))
	s.Draw(b.Tex, float32(x), float32(y), n, n, b.UV.U0, b.UV.V0, b.UV.U1, b.UV.V1, tint)
}
