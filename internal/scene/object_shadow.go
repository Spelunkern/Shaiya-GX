package scene

import (
	"math"

	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

// projectObjectShadows uses only resident SMODs intersecting the shadow area.
// Alpha-cutout materials remain masked, so the shadow cannot fill holes in a
// fence, leaf or other textured receiver.
func (w *World) projectObjectShadows(target *render.ShadowMap, s *render.Scene3D, projector render.M4, x0, z0, x1, z1, height float32) {
	if x1 < x0 || z1 < z0 || x1-x0 > 256 || z1-z0 > 256 {
		return
	}
	vertices := w.objectShadowVertices[:0]
	indices := w.objectShadowIndices[:0]
	project := func(model *Model, matrix render.M4, mani bool) {
		if model == nil {
			return
		}
		center := transformPoint(model.center, matrix)
		scale := sphereScale(matrix)
		r := model.radius * scale
		if center[0]+r < x0 || center[0]-r > x1 || center[2]+r < z0 || center[2]-r > z1 || center[1]+r < height-50 || center[1]-r > height+8 {
			return
		}
		for _, part := range model.Parts {
			partCenter := transformPoint(part.shadowCenter, matrix)
			partRadius := part.shadowRadius * scale
			if partCenter[0]+partRadius < x0 || partCenter[0]-partRadius > x1 || partCenter[2]+partRadius < z0 || partCenter[2]-partRadius > z1 || partCenter[1]+partRadius < height-50 || partCenter[1]-partRadius > height+8 {
				continue
			}
			vertices, indices = vertices[:0], indices[:0]
			for i := 0; i+2 < len(part.shadowIndices); i += 3 {
				var tri [3]render.Vertex
				var lo, hi render.V3
				for j := range 3 {
					source := part.shadowVertices[part.shadowIndices[i+j]]
					tri[j] = render.Vertex{Pos: source.Pos, UV: source.UV}
					tri[j].Pos = transformPoint(tri[j].Pos, matrix)
					for axis := range 3 {
						if j == 0 {
							lo[axis], hi[axis] = tri[j].Pos[axis], tri[j].Pos[axis]
						} else {
							lo[axis], hi[axis] = min(lo[axis], tri[j].Pos[axis]), max(hi[axis], tri[j].Pos[axis])
						}
					}
				}
				if hi[0] < x0 || lo[0] > x1 || hi[2] < z0 || lo[2] > z1 || hi[1] < height-50 || lo[1] > height+8 {
					continue
				}
				base := uint32(len(vertices))
				vertices = append(vertices, tri[:]...)
				indices = append(indices, base, base+1, base+2)
			}
			if len(indices) > 0 {
				target.ProjectWorldMasked(s, projector, vertices, indices, .45, part.Tex, mani)
			}
		}
	}
	projectVani := func(asset *vaniAsset, matrix render.M4) {
		if asset == nil || asset.count <= 0 {
			return
		}
		center, radius := transformPoint(asset.center, matrix), asset.radius*sphereScale(matrix)
		if center[0]+radius < x0 || center[0]-radius > x1 || center[2]+radius < z0 || center[2]-radius > z1 || center[1]+radius < height-50 || center[1]-radius > height+8 {
			return
		}
		frame := int(math.Mod(math.Floor(w.elapsed*12), float64(asset.count)))
		for _, part := range asset.parts {
			vertices, indices = vertices[:0], indices[:0]
			for _, face := range part.faces {
				var tri [3]render.Vertex
				var lo, hi render.V3
				for j, id := range face {
					source := part.frames[frame][id]
					tri[j] = render.Vertex{Pos: transformPoint(render.V3(source.Pos), matrix), UV: source.UV}
					for axis := range 3 {
						if j == 0 {
							lo[axis], hi[axis] = tri[j].Pos[axis], tri[j].Pos[axis]
						} else {
							lo[axis], hi[axis] = min(lo[axis], tri[j].Pos[axis]), max(hi[axis], tri[j].Pos[axis])
						}
					}
				}
				if hi[0] < x0 || lo[0] > x1 || hi[2] < z0 || lo[2] > z1 || hi[1] < height-50 || lo[1] > height+8 {
					continue
				}
				base := uint32(len(vertices))
				vertices = append(vertices, tri[:]...)
				indices = append(indices, base, base+1, base+2)
			}
			if len(indices) > 0 {
				target.ProjectWorldMasked(s, projector, vertices, indices, .45, part.texture, false)
			}
		}
	}
	if stream := w.stream; stream != nil {
		if w.objectShadowSeen == nil {
			w.objectShadowSeen = make(map[int]bool)
		} else {
			clear(w.objectShadowSeen)
		}
		seen := w.objectShadowSeen
		add := func(id int) {
			if seen[id] {
				return
			}
			seen[id] = true
			o := stream.objects[id]
			if o.vani {
				projectVani(stream.vanis[o.file], o.world)
				return
			}
			matrix := o.world
			if o.rotor != nil {
				matrix = o.rotor.transform(w.elapsed)
			}
			project(stream.assets.models[o.file], matrix, o.mani)
		}
		for _, id := range stream.global {
			add(id)
		}
		size := stream.size()
		for z := int(math.Floor(float64(z0 / size))); z <= int(math.Floor(float64(z1/size))); z++ {
			for x := int(math.Floor(float64(x0 / size))); x <= int(math.Floor(float64(x1/size))); x++ {
				for _, id := range stream.index[cellKey{x, z}] {
					add(id)
				}
			}
		}
		w.objectShadowVertices, w.objectShadowIndices = vertices, indices
		return
	}
	for _, o := range w.Objects {
		project(o.Model, o.World, o.Mani)
	}
	for _, rotor := range w.rotors {
		project(rotor.model, rotor.transform(w.elapsed), true)
	}
	for _, asset := range w.vanis {
		for _, matrix := range asset.worlds {
			projectVani(asset, matrix)
		}
	}
	w.objectShadowVertices, w.objectShadowIndices = vertices, indices
}
