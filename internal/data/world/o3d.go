package world

// O3D is a 3DO static object: one textured mesh (Parsec Shaiya/3do).
type O3D struct {
	Texture  string
	Vertices []Vertex
	Faces    []Face
}

func Parse3DO(b []byte) (*O3D, error) {
	r := &reader{b: b}
	o := &O3D{Texture: r.lstring()}
	n := r.count(32)
	o.Vertices = make([]Vertex, n)
	for i := range o.Vertices {
		o.Vertices[i] = Vertex{Pos: r.vec3(), Normal: r.vec3(), UV: r.vec2()}
	}
	o.Faces = r.faces()
	return o, r.err
}
