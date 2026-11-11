package world

// SMOD is a static model (Parsec Shaiya/Smod).
type SMOD struct {
	Center    Vec3
	Radius    float32
	ViewBox   Box
	Meshes    []SMODMesh
	Collision Box
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
	// Collision meshes follow; the client does not need them for drawing.
	return m, r.err
}
