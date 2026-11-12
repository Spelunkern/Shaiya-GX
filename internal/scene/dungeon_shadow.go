package scene

import (
	"math"

	"github.com/Spelunkern/Shaiya-GX/internal/data/world"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

const shadowCellSize = 32

type shadowTriangle struct {
	vertices [3]render.V3
	min, max render.V3
}

type dungeonShadowIndex struct {
	triangles []shadowTriangle
	cells     map[cellKey][]int
	global    []int
}

func (w *World) HasShadowReceivers() bool {
	return w != nil && (w.HasTerrain() || w.dgShadow != nil && len(w.dgShadow.triangles) != 0 || len(w.Objects) != 0 || len(w.rotors) != 0 || len(w.vanis) != 0)
}

func buildDungeonShadowIndex(root *world.DGNode) *dungeonShadowIndex {
	index := &dungeonShadowIndex{cells: make(map[cellKey][]int)}
	root.Walk(func(node *world.DGNode) {
		for _, group := range node.Groups {
			for _, mesh := range group.Meshes {
				for _, face := range mesh.Faces {
					if int(face[0]) >= len(mesh.Vertices) || int(face[1]) >= len(mesh.Vertices) || int(face[2]) >= len(mesh.Vertices) {
						continue
					}
					tri := shadowTriangle{}
					valid := true
					for i, vertex := range face {
						p := render.V3(mesh.Vertices[vertex].Pos)
						for axis := range 3 {
							if math.IsNaN(float64(p[axis])) || math.IsInf(float64(p[axis]), 0) || math.Abs(float64(p[axis])) > 1e6 {
								valid = false
							}
							if i == 0 {
								tri.min[axis], tri.max[axis] = p[axis], p[axis]
							} else {
								tri.min[axis], tri.max[axis] = min(tri.min[axis], p[axis]), max(tri.max[axis], p[axis])
							}
						}
						tri.vertices[i] = p
					}
					if !valid || tri.vertices[1].Sub(tri.vertices[0]).Cross(tri.vertices[2].Sub(tri.vertices[0])).Len() < 1e-6 {
						continue
					}
					id := len(index.triangles)
					index.triangles = append(index.triangles, tri)
					x0, x1 := shadowCell(tri.min[0]), shadowCell(tri.max[0])
					z0, z1 := shadowCell(tri.min[2]), shadowCell(tri.max[2])
					if x1-x0 > 64 || z1-z0 > 64 {
						index.global = append(index.global, id)
						continue
					}
					for z := z0; z <= z1; z++ {
						for x := x0; x <= x1; x++ {
							index.cells[cellKey{x, z}] = append(index.cells[cellKey{x, z}], id)
						}
					}
				}
			}
		}
	})
	return index
}

func shadowCell(v float32) int { return int(math.Floor(float64(v / shadowCellSize))) }

func (index *dungeonShadowIndex) query(x0, z0, x1, z1, height float32) ([]render.Vertex, []uint32) {
	return index.queryInto(x0, z0, x1, z1, height, nil, nil, nil)
}

func (index *dungeonShadowIndex) queryInto(x0, z0, x1, z1, height float32, vertices []render.Vertex, indices []uint32, seen map[int]bool) ([]render.Vertex, []uint32) {
	if index == nil || x1 < x0 || z1 < z0 || x1-x0 > 256 || z1-z0 > 256 {
		return nil, nil
	}
	if seen == nil {
		seen = make(map[int]bool)
	} else {
		clear(seen)
	}
	vertices, indices = vertices[:0], indices[:0]
	add := func(id int) {
		if seen[id] {
			return
		}
		seen[id] = true
		tri := &index.triangles[id]
		if tri.max[0] < x0 || tri.min[0] > x1 || tri.max[2] < z0 || tri.min[2] > z1 || tri.max[1] < height-50 || tri.min[1] > height+8 {
			return
		}
		base := uint32(len(vertices))
		for _, p := range tri.vertices {
			vertices = append(vertices, render.Vertex{Pos: p})
		}
		indices = append(indices, base, base+1, base+2)
	}
	for _, id := range index.global {
		add(id)
	}
	for z := shadowCell(z0); z <= shadowCell(z1); z++ {
		for x := shadowCell(x0); x <= shadowCell(x1); x++ {
			for _, id := range index.cells[cellKey{x, z}] {
				add(id)
			}
		}
	}
	return vertices, indices
}
