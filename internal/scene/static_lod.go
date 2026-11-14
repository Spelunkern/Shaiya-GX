package scene

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/world"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"math"
)

type staticLODCPU struct {
	vertices []render.Vertex
	indices  []uint32
	error    float32
}
type staticMeshLOD struct {
	mesh  *render.Mesh
	error float32
}
type lodVertexKey struct {
	position [3]int32
	normal   [3]int32
	uv       [2]int32
}

// Cluster only vertices with closely matching normals/UVs. Track actual maximum
// displacement, keep original representatives, and discard collapsed triangles.
// Selection is limited by projected error, never by an arbitrary world distance.
func simplifyStatic(mesh world.SMODMesh, grid int) staticLODCPU {
	if len(mesh.Faces) < 64 || len(mesh.Vertices) == 0 {
		return staticLODCPU{}
	}
	lo, hi := render.V3(mesh.Vertices[0].Pos), render.V3(mesh.Vertices[0].Pos)
	for _, v := range mesh.Vertices {
		for a, p := range v.Pos {
			lo[a] = min(lo[a], p)
			hi[a] = max(hi[a], p)
		}
	}
	step := max(hi[0]-lo[0], hi[1]-lo[1], hi[2]-lo[2]) / float32(grid)
	if step <= 0 || math.IsNaN(float64(step)) || math.IsInf(float64(step), 0) {
		return staticLODCPU{}
	}
	result := staticLODCPU{}
	groups := map[lodVertexKey]uint32{}
	remap := make([]uint32, len(mesh.Vertices))
	for i, v := range mesh.Vertices {
		key := lodVertexKey{}
		for a := range 3 {
			key.position[a] = int32(math.Floor(float64((v.Pos[a] - lo[a]) / step)))
			key.normal[a] = int32(math.Round(float64(v.Normal[a] * 16)))
		}
		for a := range 2 {
			key.uv[a] = int32(math.Round(float64(v.UV[a] * 32)))
		}
		j, ok := groups[key]
		if !ok {
			j = uint32(len(result.vertices))
			groups[key] = j
			result.vertices = append(result.vertices, render.Vertex{Pos: render.V3(v.Pos), Normal: render.V3(v.Normal), UV: v.UV})
		}
		remap[i] = j
		result.error = max(result.error, render.V3(v.Pos).Sub(result.vertices[j].Pos).Len())
	}
	for _, f := range mesh.Faces {
		if int(f[0]) >= len(remap) || int(f[1]) >= len(remap) || int(f[2]) >= len(remap) {
			continue
		}
		a, b, c := remap[f[0]], remap[f[1]], remap[f[2]]
		if a != b && b != c && a != c {
			result.indices = append(result.indices, a, b, c)
		}
	}
	if len(result.indices) == 0 || len(result.indices)*100 > len(mesh.Faces)*3*90 {
		return staticLODCPU{}
	}
	return result
}
func buildStaticLODs(sm *world.SMOD) [][]staticLODCPU {
	result := make([][]staticLODCPU, len(sm.Meshes))
	for i, m := range sm.Meshes {
		result[i] = []staticLODCPU{simplifyStatic(m, 32), simplifyStatic(m, 16)}
	}
	return result
}
func (m *Model) delete() {
	for _, p := range m.Parts {
		p.Mesh.Delete()
		for _, lod := range p.lods {
			if lod.mesh != nil {
				lod.mesh.Delete()
			}
		}
	}
}
func (p *Part) meshAtLOD(level int) *render.Mesh {
	if level > 0 && level <= len(p.lods) && p.lods[level-1].mesh != nil {
		return p.lods[level-1].mesh
	}
	return p.Mesh
}

func projectedLODError(error float32, center render.V3, radius float32, s *render.Scene3D) float32 {
	v := s.View
	z := center[0]*v[2] + center[1]*v[6] + center[2]*v[10] + v[14] - radius
	if z <= error || s.ViewportW <= 0 || s.ViewportH <= 0 {
		return float32(math.Inf(1))
	}
	x := center[0]*v[0] + center[1]*v[4] + center[2]*v[8] + v[12]
	y := center[0]*v[1] + center[1]*v[5] + center[2]*v[9] + v[13]
	fx := float32(math.Abs(float64(s.Proj[0]))) * float32(s.ViewportW) * .5 * (1 + (float32(math.Abs(float64(x)))+radius)/z)
	fy := float32(math.Abs(float64(s.Proj[5]))) * float32(s.ViewportH) * .5 * (1 + (float32(math.Abs(float64(y)))+radius)/z)
	return error / (z - error) * max(fx, fy)
}
func (e *fieldDrawEntry) chooseLOD(s *render.Scene3D, center render.V3, radius float32) int {
	if !s.StaticObjectLOD || s.Shadow || (s.ClipPlane != [4]float32{} && s.DetailBias == 0) || e.object.rotor != nil {
		return 0
	}
	m := e.batch.model
	for level := 2; level > 0; level-- {
		if !m.lodAvailable[level-1] {
			continue
		}
		limit := float32(.35)
		if s.DetailBias >= 2 {
			limit = 3
		} else if s.DetailBias >= 1 {
			limit = 1.25
		}
		if e.object.drawLOD == level {
			if s.DetailBias < 1 {
				limit = .45
			} else {
				limit *= .45 / .35
			}
		} // hysteresis at subpixel error
		if projectedLODError(m.lodErrors[level-1]*e.object.scale, center, radius, s) <= limit {
			return level
		}
	}
	return 0
}
