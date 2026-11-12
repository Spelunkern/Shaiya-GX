//go:build integration

package scene

import "github.com/Spelunkern/Shaiya-GX/internal/render"

func drawTerrainCell(s *render.Scene3D, b *terrainBatch, cell render.TerrainRange) {
	if b == nil {
		return
	}
	first := true
	for _, layer := range b.material.Layers {
		if layer.Mask == nil || layer.Tile == nil {
			continue
		}
		s.DrawTerrainRange(b.mesh, layer.Mask, layer.Tile, b.material.Lightmap, layer.TileSize, first, cell)
		first = false
	}
}
