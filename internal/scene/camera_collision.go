package scene

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/world"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"math"
)

// segmentTriangle uses the authored collision triangles, never alpha-cutout
// render geometry. Its parameter is a fraction along the complete segment.
func segmentTriangle(origin, delta, a, b, c render.V3) (float32, bool) {
	e1, e2 := b.Sub(a), c.Sub(a)
	p := delta.Cross(e2)
	det := e1.Dot(p)
	if math.Abs(float64(det)) < 1e-7 {
		return 0, false
	}
	t := origin.Sub(a)
	u := t.Dot(p) / det
	if u < 0 || u > 1 {
		return 0, false
	}
	q := t.Cross(e1)
	v := delta.Dot(q) / det
	if v < 0 || u+v > 1 {
		return 0, false
	}
	d := e2.Dot(q) / det
	return d, d > 1e-5 && d <= 1
}

func colliderSegment(mesh *world.CollisionMesh, origin, delta render.V3, best float32) float32 {
	if !segmentBox(origin, delta, render.V3(mesh.Bounds.Min), render.V3(mesh.Bounds.Max)) {
		return best
	}
	for _, f := range mesh.Faces {
		if int(f[0]) >= len(mesh.Vertices) || int(f[1]) >= len(mesh.Vertices) || int(f[2]) >= len(mesh.Vertices) {
			continue
		}
		t, hit := segmentTriangle(origin, delta, render.V3(mesh.Vertices[f[0]]), render.V3(mesh.Vertices[f[1]]), render.V3(mesh.Vertices[f[2]]))
		if hit && t < best {
			best = t
		}
	}
	return best
}

// CameraIntersection follows the native character-to-desired-eye segment
// (0x550090 -> 0x5a4f00 / 0x5a7450). Broad phase reuses the map cell index.
func (w *World) CameraIntersection(origin, eye render.V3) (render.V3, bool) {
	if w == nil {
		return eye, false
	}
	delta := eye.Sub(origin)
	best := float32(1)
	project := func(m *Model, matrix render.M4) {
		if m == nil || len(m.colliders) == 0 {
			return
		}
		inv := matrix.Inverse()
		start := transformPoint(origin, inv)
		end := transformPoint(eye, inv)
		for i := range m.colliders {
			best = colliderSegment(&m.colliders[i], start, end.Sub(start), best)
		}
	}
	if s := w.stream; s != nil {
		seen := w.cameraSeen
		if seen == nil {
			seen = map[int]bool{}
			w.cameraSeen = seen
		} else {
			clear(seen)
		}
		add := func(id int) {
			if seen[id] {
				return
			}
			seen[id] = true
			o := s.objects[id]
			if o.vani {
				return
			}
			matrix := o.world
			if o.rotor != nil {
				matrix = o.rotor.transform(w.elapsed)
			}
			project(s.assets.models[o.file], matrix)
		}
		for _, id := range s.global {
			add(id)
		}
		size := s.size()
		for z := int(math.Floor(float64(min(origin[2], eye[2]) / size))); z <= int(math.Floor(float64(max(origin[2], eye[2])/size))); z++ {
			for x := int(math.Floor(float64(min(origin[0], eye[0]) / size))); x <= int(math.Floor(float64(max(origin[0], eye[0])/size))); x++ {
				for _, id := range s.index[cellKey{x, z}] {
					add(id)
				}
			}
		}
	} else {
		for _, o := range w.Objects {
			project(o.Model, o.World)
		}
		for _, o := range w.rotors {
			project(o.model, o.transform(w.elapsed))
		}
	}
	for i := range w.dgColliders {
		best = colliderSegment(&w.dgColliders[i], origin, delta, best)
	}
	// Terrain is a continuous surface; bracket a crossing along the short camera ray.
	if w.HasTerrain() {
		steps := max(1, int(delta.Len()/.2))
		previous := float32(0)
		for i := 1; i <= steps; i++ {
			t := float32(i) / float32(steps)
			if t >= best {
				break
			}
			p := origin.Add(delta.Scale(t))
			if p[1] < w.Height(p[0], p[2]) {
				lo, hi := previous, t
				for range 10 {
					mid := (lo + hi) * .5
					p = origin.Add(delta.Scale(mid))
					if p[1] < w.Height(p[0], p[2]) {
						hi = mid
					} else {
						lo = mid
					}
				}
				best = hi
				break
			}
			previous = t
		}
	}
	return origin.Add(delta.Scale(best)), best < 1
}

func segmentBox(origin, delta, minimum, maximum render.V3) bool {
	lo, hi := float32(0), float32(1)
	for axis := range 3 {
		if math.Abs(float64(delta[axis])) < 1e-8 {
			if origin[axis] < minimum[axis] || origin[axis] > maximum[axis] {
				return false
			}
			continue
		}
		a, b := (minimum[axis]-origin[axis])/delta[axis], (maximum[axis]-origin[axis])/delta[axis]
		lo, hi = max(lo, min(a, b)), min(hi, max(a, b))
		if lo > hi {
			return false
		}
	}
	return true
}
