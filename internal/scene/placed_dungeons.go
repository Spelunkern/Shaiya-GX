package scene

import (
	"log"
	"path"
	"strings"

	"github.com/Spelunkern/Shaiya-GX/internal/data/world"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

// FLD dungeon placements are complete DG geometry, not SMOD props. Geometry,
// lightmaps, collision and bounds all receive the authored placement transform.
func (w *World) loadPlacedDungeons(a *Assets) {
	for _, p := range w.WLD.Dungeons.Instances {
		if p.Asset < 0 || int(p.Asset) >= len(w.WLD.Dungeons.Assets) {
			continue
		}
		name := path.Base(strings.ReplaceAll(w.WLD.Dungeons.Assets[p.Asset], `\`, "/"))
		if !strings.EqualFold(path.Ext(name), ".dg") {
			continue
		}
		matrix := render.Basis(render.V3(p.Pos), render.V3(p.Fwd), render.V3(p.Up))
		if err := w.loadDGAt(a, name, matrix); err != nil {
			log.Printf("placed dungeon %s: %v", name, err)
		}
	}
}

func transformDG(root *world.DGNode, m render.M4) {
	root.Walk(func(n *world.DGNode) {
		for gi := range n.Groups {
			for mi := range n.Groups[gi].Meshes {
				mesh := &n.Groups[gi].Meshes[mi]
				for vi := range mesh.Vertices {
					v := &mesh.Vertices[vi]
					v.Pos = world.Vec3(transformPoint(render.V3(v.Pos), m))
					normal := render.V3(v.Normal)
					normal = render.V3{normal[0]*m[0] + normal[1]*m[4] + normal[2]*m[8], normal[0]*m[1] + normal[1]*m[5] + normal[2]*m[9], normal[0]*m[2] + normal[1]*m[6] + normal[2]*m[10]}.Norm()
					v.Normal = world.Vec3(normal)
				}
			}
		}
		for i := range n.Collider.Vertices {
			n.Collider.Vertices[i] = world.Vec3(transformPoint(render.V3(n.Collider.Vertices[i]), m))
		}
		if len(n.Collider.Vertices) > 0 {
			lo, hi := n.Collider.Vertices[0], n.Collider.Vertices[0]
			for _, v := range n.Collider.Vertices {
				for axis := range v {
					lo[axis] = min(lo[axis], v[axis])
					hi[axis] = max(hi[axis], v[axis])
				}
			}
			n.Collider.Bounds = world.Box{Min: lo, Max: hi}
		}
	})
}

func (w *World) appendDGShadow(other *dungeonShadowIndex) {
	if w.dgShadow == nil {
		w.dgShadow = other
		return
	}
	index := w.dgShadow
	offset := len(index.triangles)
	index.triangles = append(index.triangles, other.triangles...)
	for cell, ids := range other.cells {
		for _, id := range ids {
			index.cells[cell] = append(index.cells[cell], id+offset)
		}
	}
	for _, id := range other.global {
		index.global = append(index.global, id+offset)
	}
}
