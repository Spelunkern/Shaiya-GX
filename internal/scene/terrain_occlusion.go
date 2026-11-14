package scene

import "github.com/Spelunkern/Shaiya-GX/internal/render"

// PrepareTerrainOcclusion uses only loaded, drawable terrain. The flat lower
// envelope of each resident cell is below every point of its triangle mesh.
// With the eye above terrain, a ray reaching that envelope must have crossed
// the real opaque terrain first. Unknown/underground cameras fail open.
func (w *World) PrepareTerrainOcclusion(s *render.Scene3D, eye render.V3) {
	s.Occlusion = nil
	if w != nil && !w.HasTerrain() && len(w.dg) > 0 {
		w.prepareDungeonOcclusion(s, eye)
		return
	}
	if w == nil || !w.HasTerrain() || w.stream == nil || len(w.stream.pending) != 0 || w.WLD == nil || eye[0] < 0 || eye[2] < 0 || eye[0] > float32(w.WLD.MapSize) || eye[2] > float32(w.WLD.MapSize) || eye[1] <= w.Height(eye[0], eye[2])+.1 {
		return
	}
	if w.occlusion != nil && w.occlusionRevision == w.stream.revision && w.occlusionEye == eye && w.occlusion.Matches(s.View, s.VisibilityProjection()) {
		w.occlusion.Culled = 0
		s.Occlusion = w.occlusion
		return
	}
	// The complete near plane must also be above terrain. Otherwise the
	// real surface could be near-clipped while its lower envelope is not.
	inverse := s.View.Mul(s.VisibilityProjection()).Inverse()
	minY, minX, minZ := float32(1e30), float32(1e30), float32(1e30)
	maxX, maxZ := float32(-1e30), float32(-1e30)
	for _, xy := range [4][2]float32{{-1, -1}, {1, -1}, {-1, 1}, {1, 1}} {
		x, y := xy[0], xy[1]
		ww := x*inverse[3] + y*inverse[7] + inverse[15]
		if ww <= 0 {
			return
		}
		p := render.V3{(x*inverse[0] + y*inverse[4] + inverse[12]) / ww, (x*inverse[1] + y*inverse[5] + inverse[13]) / ww, (x*inverse[2] + y*inverse[6] + inverse[14]) / ww}
		minY = min(minY, p[1])
		minX = min(minX, p[0])
		maxX = max(maxX, p[0])
		minZ = min(minZ, p[2])
		maxZ = max(maxZ, p[2])
	}
	for z := max(0, int(minZ/w.step)-1); z <= min(w.side-1, int(maxZ/w.step)+2); z++ {
		for x := max(0, int(minX/w.step)-1); x <= min(w.side-1, int(maxX/w.step)+2); x++ {
			if w.heights[z*w.side+x]+.02 >= minY {
				return
			}
		}
	}
	// All resident patches must actually write terrain depth; an unsupported
	// material is treated as an opening, not as an invisible occluder.
	for _, key := range w.stream.order {
		if c := w.stream.cells[key]; c != nil {
			for _, p := range c.parts {
				drawable := false
				for _, l := range p.terrain.Layers {
					if l.Mask != nil && l.Tile != nil {
						drawable = true
						break
					}
				}
				if !drawable {
					return
				}
			}
		}
	}
	if w.occlusion == nil {
		w.occlusion = render.NewTerrainOcclusion(160, 90)
	}
	o := w.occlusion
	o.Reset(s.View, s.VisibilityProjection())

	for _, key := range w.stream.order {
		if cell := w.stream.cells[key]; cell != nil {
			for _, p := range cell.parts {

				drawable := false
				for _, l := range p.terrain.Layers {
					if l.Mask != nil && l.Tile != nil {
						drawable = true
						break
					}
				}
				if !drawable {
					continue
				}
				lo, hi := p.minimum, p.maximum
				o.FlatQuad(lo, hi, lo[1]-.05)
				// A stepped lower envelope needs its vertical joins as well:
				// a hill taller than the camera can hide objects without a
				// ray ever crossing the horizontal top of the lower prism.
				for _, nk := range []cellKey{key, {key.x + 1, key.z}, {key.x, key.z + 1}} {
					if neighbor := w.stream.cells[nk]; neighbor != nil {
						for _, q := range neighbor.parts {
							low, high := min(p.minimum[1], q.minimum[1])-.05, max(p.minimum[1], q.minimum[1])-.05
							if high-low < .001 {
								continue
							}
							if q.minimum[0] == p.maximum[0] {
								z0, z1 := max(p.minimum[2], q.minimum[2]), min(p.maximum[2], q.maximum[2])
								if z1 > z0 {
									x := p.maximum[0]
									o.PlanarQuad(render.V3{x, low, z0}, render.V3{x, low, z1}, render.V3{x, high, z1}, render.V3{x, high, z0})
								}
							}
							if q.minimum[2] == p.maximum[2] {
								x0, x1 := max(p.minimum[0], q.minimum[0]), min(p.maximum[0], q.maximum[0])
								if x1 > x0 {
									z := p.maximum[2]
									o.PlanarQuad(render.V3{x0, low, z}, render.V3{x1, low, z}, render.V3{x1, high, z}, render.V3{x0, high, z})
								}
							}
						}
					}
				}

			}
		}
	}
	w.occlusionRevision, w.occlusionEye = w.stream.revision, eye
	s.Occlusion = o
}
