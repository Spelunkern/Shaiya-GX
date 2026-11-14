package scene

import "github.com/Spelunkern/Shaiya-GX/internal/render"

// Cell meshes retain section UVs: masks/lightmaps must never restart at each cell.
func outTerrain(w *World, x0, z0, nx, nz, sx, sz int) ([]render.Vertex, []uint32) {
	vertices, indices := (nx+1)*(nz+1), nx*nz*6
	if w.terrainFan && x0%4 == 0 && z0%4 == 0 && nx%4 == 0 && nz%4 == 0 {
		vertices, indices = (nx/4+1)*(nz/4+1)+(nx/4)*(nz/4), (nx/4)*(nz/4)*12
	}
	return appendTerrain(make([]render.Vertex, 0, vertices), make([]uint32, 0, indices), w, x0, z0, nx, nz, sx, sz)
}

// appendTerrain writes directly into a reusable batch and offsets the indices.
func appendTerrain(v []render.Vertex, idx []uint32, w *World, x0, z0, nx, nz, sx, sz int) ([]render.Vertex, []uint32) {
	if w.terrainFan && x0%4 == 0 && z0%4 == 0 && nx%4 == 0 && nz%4 == 0 {
		return appendTerrainFans(v, idx, w, x0, z0, nx, nz, sx, sz)
	}
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
			if w.terrainFan && ((x0+x)%4 < 2) == ((z0+z)%4 < 2) {
				idx = append(idx, a, b, d, a, d, c)
			} else {
				idx = append(idx, a, b, c, b, d, c)
			}
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

// Use the original finest fan topology rather than rendering every stored
// height sample. Corners are shared across tiles; streamed cells share XYZ/UV.
func appendTerrainFans(v []render.Vertex, idx []uint32, w *World, x0, z0, nx, nz, sx, sz int) ([]render.Vertex, []uint32) {
	section := (w.side - 1) / w.sectionsPerSide()
	base := uint32(len(v))
	cols, rows := nx/4+1, nz/4+1
	add := func(gx, gz int) {
		px, pz := float32(gx)*w.step, float32(gz)*w.step
		v = append(v, render.Vertex{Pos: render.V3{px, w.heights[gz*w.side+gx], pz}, Normal: render.V3{0, 1, 0}, UV: [2]float32{float32(gx-sx*section) / float32(section), float32(gz-sz*section) / float32(section)}, UV2: [2]float32{px, pz}})
	}
	for z := 0; z < rows; z++ {
		for x := 0; x < cols; x++ {
			add(x0+x*4, z0+z*4)
		}
	}
	for z := 0; z < rows-1; z++ {
		for x := 0; x < cols-1; x++ {
			a := base + uint32(z*cols+x)
			b, c := a+1, a+uint32(cols)
			d := c + 1
			m := uint32(len(v))
			add(x0+x*4+2, z0+z*4+2)
			idx = append(idx, a, b, m, b, d, m, d, c, m, c, a, m)
		}
	}
	return v, idx
}
