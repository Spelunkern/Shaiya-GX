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

const cameraColliderBlockSize = 32

type colliderBlock struct {
	min, max render.V3
	valid    bool
}

func buildColliderBlocks(mesh *world.CollisionMesh) []colliderBlock {
	if len(mesh.Faces) <= cameraColliderBlockSize {
		return nil
	}
	blocks := make([]colliderBlock, (len(mesh.Faces)+cameraColliderBlockSize-1)/cameraColliderBlockSize)
	for blockIdx := range blocks {
		start := blockIdx * cameraColliderBlockSize
		end := min(start+cameraColliderBlockSize, len(mesh.Faces))
		lo := render.V3{math.MaxFloat32, math.MaxFloat32, math.MaxFloat32}
		hi := render.V3{-math.MaxFloat32, -math.MaxFloat32, -math.MaxFloat32}
		has := false
		for _, f := range mesh.Faces[start:end] {
			for _, vi := range f {
				if int(vi) >= len(mesh.Vertices) {
					continue
				}
				v := mesh.Vertices[vi]
				has = true
				for axis := range 3 {
					lo[axis] = min(lo[axis], v[axis])
					hi[axis] = max(hi[axis], v[axis])
				}
			}
		}
		if has {
			blocks[blockIdx] = colliderBlock{min: lo, max: hi, valid: true}
		}
	}
	return blocks
}

func colliderSegment(mesh *world.CollisionMesh, origin, delta render.V3, best float32) float32 {
	return colliderSegmentBlocks(mesh, nil, origin, delta, best)
}

func colliderSegmentBlocks(mesh *world.CollisionMesh, blocks []colliderBlock, origin, delta render.V3, best float32) float32 {
	if !segmentBoxParam(origin, delta, render.V3(mesh.Bounds.Min), render.V3(mesh.Bounds.Max), best) {
		return best
	}
	if len(blocks) == 0 {
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
	for blockIdx, blk := range blocks {
		if !blk.valid || !segmentBoxParam(origin, delta, blk.min, blk.max, best) {
			continue
		}
		start := blockIdx * cameraColliderBlockSize
		end := min(start+cameraColliderBlockSize, len(mesh.Faces))
		for _, f := range mesh.Faces[start:end] {
			if int(f[0]) >= len(mesh.Vertices) || int(f[1]) >= len(mesh.Vertices) || int(f[2]) >= len(mesh.Vertices) {
				continue
			}
			t, hit := segmentTriangle(origin, delta, render.V3(mesh.Vertices[f[0]]), render.V3(mesh.Vertices[f[1]]), render.V3(mesh.Vertices[f[2]]))
			if hit && t < best {
				best = t
			}
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
		if m.colliderBlocks == nil {
			m.colliderBlocks = make([][]colliderBlock, len(m.colliders))
			for i := range m.colliders {
				m.colliderBlocks[i] = buildColliderBlocks(&m.colliders[i])
			}
		}
		inv := matrix.Inverse()
		start := transformPoint(origin, inv)
		end := transformPoint(eye, inv)
		localDelta := end.Sub(start)
		for i := range m.colliders {
			best = colliderSegmentBlocks(&m.colliders[i], m.colliderBlocks[i], start, localDelta, best)
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
			if o.vani || o.mani || o.rotor != nil {
				return
			}
			matrix := o.world
			if o.rotor != nil {
				matrix = o.rotor.transform(w.DynamicTime())
			}
			if o.radius > 0 {
				center, radius := o.drawCenter, o.drawRadius
				if !o.drawBoundsValid {
					center = transformPoint(o.center, o.world)
					radius = o.radius * o.scale
				}
				if !segmentSphere(origin, delta, center, radius+0.05) {
					return
				}
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
			if o.Mani {
				continue
			}
			project(o.Model, o.World)
		}
	}
	if len(w.dgColliders) > 0 && len(w.dgColliderBlocks) != len(w.dgColliders) {
		w.dgColliderBlocks = make([][]colliderBlock, len(w.dgColliders))
		for i := range w.dgColliders {
			w.dgColliderBlocks[i] = buildColliderBlocks(&w.dgColliders[i])
		}
	}
	for i := range w.dgColliders {
		best = colliderSegmentBlocks(&w.dgColliders[i], w.dgColliderBlocks[i], origin, delta, best)
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

func segmentSphere(origin, delta, center render.V3, radius float32) bool {
	if radius <= 0 {
		return true
	}
	oc := origin.Sub(center)
	dd := delta.Dot(delta)
	if dd <= 1e-12 {
		return oc.Dot(oc) <= radius*radius
	}
	t := min(max(-oc.Dot(delta)/dd, float32(0)), float32(1))
	closest := oc.Add(delta.Scale(t))
	return closest.Dot(closest) <= radius*radius
}

func segmentBox(origin, delta, minimum, maximum render.V3) bool {
	return segmentBoxParam(origin, delta, minimum, maximum, 1)
}

func segmentBoxParam(origin, delta, minimum, maximum render.V3, maxT float32) bool {
	lo, hi := float32(0), maxT
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
