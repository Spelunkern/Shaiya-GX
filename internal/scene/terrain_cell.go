package scene

import "github.com/Spelunkern/Shaiya-GX/internal/render"

// Cell meshes retain section UVs: masks/lightmaps must never restart at each cell.
func outTerrain(w *World, x0, z0, nx, nz, sx, sz int) ([]render.Vertex, []uint32) {
	return appendTerrain(make([]render.Vertex, 0, (nx+1)*(nz+1)), make([]uint32, 0, nx*nz*6), w, x0, z0, nx, nz, sx, sz)
}

// appendTerrain writes directly into a reusable batch and offsets the indices.
func appendTerrain(v []render.Vertex, idx []uint32, w *World, x0, z0, nx, nz, sx, sz int) ([]render.Vertex, []uint32) {
	section := (w.side - 1) / w.sectionsPerSide()
	base := uint32(len(v))
	for z := 0; z <= nz; z++ {
		for x := 0; x <= nx; x++ {
			gx, gz := x0+x, z0+z
			px, pz := float32(gx)*w.step, float32(gz)*w.step
			v = append(v, render.Vertex{Pos: render.V3{px, w.heights[gz*w.side+gx], pz}, Normal: render.V3{0, 1, 0},
				UV: [2]float32{float32(gx-sx*section) / float32(section), float32(gz-sz*section) / float32(section)}, UV2: [2]float32{px, pz}})
		}
	}
	for z := 0; z < nz; z++ {
		for x := 0; x < nx; x++ {
			a, b := base+uint32(z*(nx+1)+x), base+uint32(z*(nx+1)+x+1)
			c, d := base+uint32((z+1)*(nx+1)+x), base+uint32((z+1)*(nx+1)+x+1)
			idx = append(idx, a, b, c, b, d, c)
		}
	}
	return v, idx
}
func drawSection(s *render.Scene3D, sec Section) {
	first := true
	for _, l := range sec.Layers {
		if l.Mask == nil || l.Tile == nil || sec.Mesh == nil {
			continue
		}
		s.DrawTerrainLayer(sec.Mesh, l.Mask, l.Tile, sec.Lightmap, l.TileSize, first)
		first = false
	}
}
