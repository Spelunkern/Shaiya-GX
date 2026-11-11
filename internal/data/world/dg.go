package world

import "errors"

// DG is dungeon geometry: an octree of textured, lightmapped meshes
// (Parsec Shaiya/Dg). Lightmaps are <dir>/<name>/<name>_L<i>.dds.
type DG struct {
	Bounds    Box
	Textures  []string
	Lightmaps int32
	Root      *DGNode
}

type DGNode struct {
	Center   Vec3
	ViewBox  Box
	CollBox  Box
	Groups   []DGGroup
	Children []*DGNode
}

type DGGroup struct {
	Texture int32
	Meshes  []DGMesh
}

type DGMesh struct {
	Lightmap int32
	Vertices []Vertex
	Faces    []Face
}

var errDepth = errors.New("world: dg octree too deep")

func ParseDG(b []byte) (*DG, error) {
	r := &reader{b: b}
	d := &DG{Bounds: r.box(), Textures: r.names(), Lightmaps: r.i32()}
	if r.i32() > 0 {
		d.Root = r.dgNode(0)
	}
	return d, r.err
}

func (r *reader) dgNode(depth int) *DGNode {
	if depth > 32 {
		r.err = errDepth
		return nil
	}
	n := &DGNode{Center: r.vec3(), ViewBox: r.box(), CollBox: r.box()}
	ng := r.count(8)
	for i := 0; i < ng && r.err == nil; i++ {
		g := DGGroup{Texture: r.i32()}
		nm := r.count(12)
		for j := 0; j < nm && r.err == nil; j++ {
			m := DGMesh{Lightmap: r.i32()}
			nv := r.count(44)
			m.Vertices = make([]Vertex, nv)
			for k := range m.Vertices {
				m.Vertices[k] = Vertex{Pos: r.vec3(), Normal: r.vec3(), Bone: r.i32(), UV: r.vec2(), UV2: r.vec2()}
			}
			m.Faces = r.faces()
			g.Meshes = append(g.Meshes, m)
		}
		n.Groups = append(n.Groups, g)
	}
	if r.i32() == 1 { // collision mesh
		nv := r.count(12)
		r.take(12 * nv)
		r.faces()
	}
	for i := 0; i < 8 && r.err == nil; i++ {
		if r.i32() > 0 {
			n.Children = append(n.Children, r.dgNode(depth+1))
		}
	}
	return n
}

// Walk visits every node.
func (n *DGNode) Walk(fn func(*DGNode)) {
	if n == nil {
		return
	}
	fn(n)
	for _, c := range n.Children {
		c.Walk(fn)
	}
}
