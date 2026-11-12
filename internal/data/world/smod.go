package world

import (
	"fmt"
	"math"
)

// SMOD is a static model (Parsec Shaiya/Smod).
type SMOD struct {
	Center    Vec3
	Radius    float32
	ViewBox   Box
	Meshes    []SMODMesh
	Collision Box
	Colliders []CollisionMesh
}

type CollisionMesh struct {
	Bounds   Box
	Vertices []Vec3
	Faces    []Face
}

type SMODMesh struct {
	Texture  string
	Vertices []Vertex
	Faces    []Face
}

// Vertex is shared by SMOD and DG meshes; DG adds a lightmap UV.
type Vertex struct {
	Pos, Normal Vec3
	Bone        int32
	UV, UV2     [2]float32
}

func ParseSMOD(b []byte) (*SMOD, error) {
	r := &reader{b: b}
	m := &SMOD{Center: r.vec3(), Radius: r.f32(), ViewBox: r.box()}
	n := r.count(12)
	for i := 0; i < n && r.err == nil; i++ {
		mesh := SMODMesh{Texture: r.lstring()}
		nv := r.count(36)
		mesh.Vertices = make([]Vertex, nv)
		for j := range mesh.Vertices {
			mesh.Vertices[j] = Vertex{Pos: r.vec3(), Normal: r.vec3(), Bone: r.i32(), UV: r.vec2()}
		}
		mesh.Faces = r.faces()
		m.Meshes = append(m.Meshes, mesh)
	}
	m.Collision = r.box()
	n = r.count(8)
	for i := 0; i < n && r.err == nil; i++ {
		c := r.collisionMesh()
		m.Colliders = append(m.Colliders, c)
	}
	return m, r.err
}

// SMODBounds reads only the 16-byte sphere header (Parsec Smod Center/Radius).
func SMODBounds(b []byte) (Vec3, float32, error) {
	r := &reader{b: b}
	center, radius := r.vec3(), r.f32()
	if r.err != nil {
		return center, radius, r.err
	}
	for _, v := range []float32{center[0], center[1], center[2], radius} {
		if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
			return center, radius, fmt.Errorf("smod: non-finite bounds")
		}
	}
	if radius < 0 {
		return center, radius, fmt.Errorf("smod: negative radius")
	}
	return center, radius, nil
}

func (r *reader) collisionMesh() CollisionMesh {
	c := CollisionMesh{}
	nv := r.count(12)
	c.Vertices = make([]Vec3, nv)
	for i := range c.Vertices {
		p := r.vec3()
		c.Vertices[i] = p
		if i == 0 {
			c.Bounds = Box{Min: p, Max: p}
		} else {
			for axis := range 3 {
				c.Bounds.Min[axis] = min(c.Bounds.Min[axis], p[axis])
				c.Bounds.Max[axis] = max(c.Bounds.Max[axis], p[axis])
			}
		}
	}
	c.Faces = r.faces()
	return c
}
