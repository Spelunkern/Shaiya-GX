package scene

import (
	"fmt"

	"github.com/Spelunkern/Shaiya-GX/internal/data/world"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

// GroundModel is one of the shared 3DOs used by the original for dropped items.
type GroundModel struct {
	Part     Part
	Radius   float32
	vertices []render.V3
	indices  []uint32
}

// GroundItemModel maps the server item type to the original client's model.
func GroundItemModel(typ uint8) int {
	switch typ {
	case 26:
		return 3
	case 22, 23, 24, 30, 37, 39, 40, 94, 96, 97:
		return 1
	case 25, 27, 28, 29, 38, 41, 44, 78, 79, 80, 99, 100, 101, 102, 103:
		return 2
	default:
		return 0
	}
}

// GroundModel loads each original mesh and texture at most once per session.
func (s *Assets) GroundModel(kind int) *GroundModel {
	if kind < 0 || kind >= len(s.groundModels) {
		return nil
	}
	if s.groundAttempted[kind] {
		return s.groundModels[kind]
	}
	s.groundAttempted[kind] = true
	names := [...]string{"drop_item001", "drop_item002", "drop_item003", "gold"}
	name := names[kind]
	file := "Item/3DO/" + name + ".3DO"
	b, err := s.Arc.ReadFile(file)
	if err != nil {
		s.Missing[file] = true
		return nil
	}
	obj, err := world.Parse3DO(b)
	if err != nil {
		s.Missing[fmt.Sprintf("%s: %v", file, err)] = true
		return nil
	}
	tex := s.modelTexture("Item/DDS/" + name + ".DDS")
	if tex == nil {
		return nil
	}
	vertices := make([]render.Vertex, len(obj.Vertices))
	var radius float32
	for i, v := range obj.Vertices {
		pos := render.V3(v.Pos)
		radius = max(radius, pos.Len())
		vertices[i] = render.Vertex{Pos: pos, Normal: render.V3(v.Normal), UV: v.UV}
	}
	indices := make([]uint32, 0, len(obj.Faces)*3)
	for _, f := range obj.Faces {
		if int(f[0]) < len(vertices) && int(f[1]) < len(vertices) && int(f[2]) < len(vertices) {
			indices = append(indices, uint32(f[0]), uint32(f[1]), uint32(f[2]))
		}
	}
	if len(indices) == 0 {
		s.Missing[file+": empty mesh"] = true
		return nil
	}
	points := make([]render.V3, len(vertices))
	for i := range vertices {
		points[i] = vertices[i].Pos
	}
	m := &GroundModel{Part: Part{Mesh: render.NewMesh(vertices, indices), Tex: tex}, Radius: radius, vertices: points, indices: indices}
	s.groundModels[kind] = m
	return m
}

// RayIntersection returns the closest hit in local space. Directions are
// normalized by the caller; the original ground models are only translated.
func (m *GroundModel) RayIntersection(origin, direction render.V3) (float32, bool) {
	best := float32(0)
	for i := 0; i+2 < len(m.indices); i += 3 {
		a, b, c := m.vertices[m.indices[i]], m.vertices[m.indices[i+1]], m.vertices[m.indices[i+2]]
		e1, e2 := b.Sub(a), c.Sub(a)
		p := direction.Cross(e2)
		det := e1.Dot(p)
		if det > -1e-7 && det < 1e-7 {
			continue
		}
		inv := 1 / det
		t := origin.Sub(a)
		u := t.Dot(p) * inv
		if u < 0 || u > 1 {
			continue
		}
		q := t.Cross(e1)
		v := direction.Dot(q) * inv
		if v < 0 || u+v > 1 {
			continue
		}
		distance := e2.Dot(q) * inv
		if distance >= 0 && (best == 0 || distance < best) {
			best = distance
		}
	}
	return best, best > 0
}
