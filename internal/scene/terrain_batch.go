package scene

import "github.com/Spelunkern/Shaiya-GX/internal/render"

// Small spatial batches limit rebuild work as cells stream in and out while
// allowing each camera to submit only its visible cell ranges.
func (s *fieldStream) rebuildTerrainBatches(dirty map[terrainBatchKey]bool) {
	for group := range dirty {
		s.terrainVertices, s.terrainIndices = s.terrainVertices[:0], s.terrainIndices[:0]
		var material Section
		for _, key := range s.order {
			cell := s.cells[key]
			if cell == nil {
				continue
			}
			for i := range cell.parts {
				p := &cell.parts[i]
				if p.batch != group {
					continue
				}
				material = p.terrain
				p.rangeInBatch = render.TerrainRange{First: uint32(len(s.terrainIndices)), Count: uint32(p.nx * p.nz * 6)}
				s.terrainVertices, s.terrainIndices = appendTerrain(s.terrainVertices, s.terrainIndices, s.world, p.x, p.z, p.nx, p.nz, p.sx, p.sz)
			}
		}
		if old := s.terrainBatches[group]; old != nil {
			old.mesh.Delete()
			delete(s.terrainBatches, group)
		}
		if len(s.terrainIndices) > 0 {
			s.terrainBatches[group] = &terrainBatch{mesh: render.NewMesh(s.terrainVertices, s.terrainIndices), material: material}
		}
	}
}

func drawTerrainBatch(s *render.Scene3D, b *terrainBatch) {
	if b == nil || len(b.ranges) == 0 {
		return
	}
	s.SetTerrainRanges(b.ranges)
	first := true
	for _, layer := range b.material.Layers {
		if layer.Mask == nil || layer.Tile == nil {
			continue
		}
		s.DrawTerrainRanges(b.mesh, layer.Mask, layer.Tile, b.material.Lightmap, layer.TileSize, first, len(b.ranges))
		first = false
	}
}
