package scene

import (
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"math"
	"sort"
)

// Actual opaque DG triangles provide a conservative depth mask for this camera.
// No collision hulls, room-sized solid boxes, stale queries or GPU readbacks.
func (w *World) prepareDungeonOcclusion(s *render.Scene3D, eye render.V3) {
	if s.Shadow || s.ClipPlane != [4]float32{} {
		return
	}
	if w.occlusion != nil && w.occlusion.Matches(s.View, s.Proj) {
		w.occlusion.Culled = 0
		s.Occlusion = w.occlusion
		return
	}
	if w.occlusion == nil {
		w.occlusion = render.NewTerrainOcclusion(160, 90)
	}
	o := w.occlusion
	o.Reset(s.View, s.Proj)
	f := render.ViewFrustum(s.View, s.Proj)
	w.dgOrder = w.dgOrder[:0]
	distance := func(i int) float32 {
		p := w.dungeonOccluders[i]
		var d float32
		for a := range 3 {
			v := max(p.minimum[a]-eye[a], eye[a]-p.maximum[a], 0)
			d += v * v
		}
		return d
	}
	for i, p := range w.dungeonOccluders {
		if len(p.triangles) > 0 && distance(i) < 128*128 && f.BoxVisible(p.minimum, p.maximum) {
			w.dgOrder = append(w.dgOrder, i)
		}
	}
	sort.Slice(w.dgOrder, func(i, j int) bool { return distance(w.dgOrder[i]) < distance(w.dgOrder[j]) })
	remaining := 512
	for _, i := range w.dgOrder {
		for _, tri := range w.dungeonOccluders[i].triangles {
			o.Triangle(tri[0], tri[1], tri[2])
			remaining--
			if remaining == 0 {
				s.Occlusion = o
				return
			}
		}
	}
	s.Occlusion = o
}

func dungeonModelVisible(s *render.Scene3D, f render.Frustum, m *Model, world render.M4) bool {
	if m == nil {
		return false
	}
	if m.radius <= 0 {
		return true
	}
	c, r := transformPoint(m.center, world), m.radius*sphereScale(world)
	return f.SphereVisible(c, r) && render.ClipSphereVisible(s.ClipPlane, c, r) && !s.OccludedSphere(c, r)
}
func dungeonMeshVisible(s *render.Scene3D, f render.Frustum, m *render.Mesh, world render.M4) bool {
	lo, hi, ok := m.Bounds()
	if !ok {
		return true
	}
	var minimum, maximum render.V3
	for i := 0; i < 8; i++ {
		p := lo
		for axis := range 3 {
			if i&(1<<axis) != 0 {
				p[axis] = hi[axis]
			}
		}
		p = transformPoint(p, world)
		if i == 0 {
			minimum, maximum = p, p
		} else {
			for axis := range 3 {
				minimum[axis] = min(minimum[axis], p[axis])
				maximum[axis] = max(maximum[axis], p[axis])
			}
		}
	}
	return f.BoxVisible(minimum, maximum) && render.ClipBoxVisible(s.ClipPlane, minimum, maximum) && !s.OccludedBox(minimum, maximum)
}

func triangleArea2(t [3]render.V3) float32 { return t[1].Sub(t[0]).Cross(t[2].Sub(t[0])).Len() }

type dungeonOccluder struct {
	minimum, maximum render.V3
	triangles        [][3]render.V3
}

// Bake static placements only. Moving rotors and animated/cutout surfaces are
// never occluders. Spatial bins stop a huge DG mesh monopolizing the budget.
func (w *World) buildDungeonOccluders() {
	bins := map[[3]int]int{}
	add := func(tri [3]render.V3) {
		center := tri[0].Add(tri[1]).Add(tri[2]).Scale(1.0 / 3)
		key := [3]int{}
		for a := range 3 {
			key[a] = int(math.Floor(float64(center[a] / 16)))
		}
		i, ok := bins[key]
		if !ok {
			i = len(w.dungeonOccluders)
			bins[key] = i
			w.dungeonOccluders = append(w.dungeonOccluders, dungeonOccluder{minimum: tri[0], maximum: tri[0]})
		}
		b := &w.dungeonOccluders[i]
		b.triangles = append(b.triangles, tri)
		for _, p := range tri {
			for a := range 3 {
				b.minimum[a] = min(b.minimum[a], p[a])
				b.maximum[a] = max(b.maximum[a], p[a])
			}
		}
	}
	for _, p := range w.dg {
		for _, tri := range p.occluders {
			add(tri)
		}
	}
	for _, object := range w.Objects {
		for _, tri := range object.Model.occluders {
			for i := range tri {
				tri[i] = transformPoint(tri[i], object.World)
			}
			add(tri)
		}
	}
	for i := range w.dungeonOccluders {
		b := &w.dungeonOccluders[i]
		sort.Slice(b.triangles, func(i, j int) bool { return triangleArea2(b.triangles[i]) > triangleArea2(b.triangles[j]) })
		if len(b.triangles) > 64 {
			b.triangles = append([][3]render.V3(nil), b.triangles[:64]...)
		}
	}
}

func (w *World) drawDungeonObjects(s *render.Scene3D, f render.Frustum) {
	for _, mani := range []bool{true, false} {
		var pending *Model
		w.dungeonInstances = w.dungeonInstances[:0]
		flush := func() {
			matrices := w.dungeonInstances
			if len(matrices) == 0 {
				return
			}
			p := pending.Parts[0]
			if len(matrices) == 1 {
				s.DrawModel(p.Mesh, p.Tex, matrices[0], render.NativeWorldAlphaRef(mani), false)
			} else {
				s.SetInstances(matrices)
				s.DrawInstanced(p.Mesh, p.Tex, render.NativeWorldAlphaRef(mani))
			}
			w.dungeonInstances = w.dungeonInstances[:0]
		}
		for _, object := range w.Objects {
			if object.Mani != mani || !dungeonModelVisible(s, f, object.Model, object.World) {
				continue
			}
			model := object.Model
			if len(model.Parts) == 1 {
				p := model.Parts[0]
				if !dungeonMeshVisible(s, f, p.Mesh, object.World) {
					continue
				}
				if pending != model {
					flush()
					pending = model
				}
				w.dungeonInstances = append(w.dungeonInstances, object.World)
			} else {
				flush()
				pending = nil
				for _, p := range model.Parts {
					if dungeonMeshVisible(s, f, p.Mesh, object.World) {
						s.DrawModel(p.Mesh, p.Tex, object.World, render.NativeWorldAlphaRef(mani), false)
					}
				}
			}
		}
		flush()
	}
}
