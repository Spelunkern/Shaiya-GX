package scene

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/world"
)

func TestDungeonShadowReceiversAreSpatialAndDeduplicated(t *testing.T) {
	root := &world.DGNode{Groups: []world.DGGroup{{Meshes: []world.DGMesh{{
		Vertices: []world.Vertex{{Pos: world.Vec3{-40, 0, -40}}, {Pos: world.Vec3{40, 0, -40}}, {Pos: world.Vec3{0, 0, 40}}, {Pos: world.Vec3{0, 100, 0}}, {Pos: world.Vec3{1, 100, 0}}, {Pos: world.Vec3{0, 100, 1}}},
		Faces:    []world.Face{{0, 1, 2}, {3, 4, 5}},
	}}}}}
	index := buildDungeonShadowIndex(root)
	vertices, faces := index.query(-35, -35, 35, 35, 1)
	if len(vertices) != 3 || len(faces) != 3 {
		t.Fatalf("nearby floor returned %d vertices and %d indices", len(vertices), len(faces))
	}
	if _, faces = index.query(100, 100, 110, 110, 1); len(faces) != 0 {
		t.Fatal("distant triangle included")
	}
	if _, faces = index.query(-1, -1, 1, 1, 100); len(faces) != 3 {
		t.Fatal("height filtering selected wrong floor")
	}
}
