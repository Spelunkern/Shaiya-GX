package scene

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/world"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"testing"
)

func TestCameraUsesAuthoredColliderAndNearestHit(t *testing.T) {
	mesh := world.CollisionMesh{Vertices: []world.Vec3{{-2, 0, -2}, {2, 0, -2}, {0, 4, -2}}, Faces: []world.Face{{0, 1, 2}}, Bounds: world.Box{Min: world.Vec3{-2, 0, -2}, Max: world.Vec3{2, 4, -2}}}
	m := &Model{colliders: []world.CollisionMesh{mesh}}
	w := &World{Objects: []Instance{{Model: m, World: render.Identity()}}}
	origin, eye := render.V3{0, 1, 0}, render.V3{0, 1, -5}
	if p, hit := w.CameraIntersection(origin, eye); !hit || p.Sub(render.V3{0, 1, -2}).Len() > .0001 {
		t.Fatal("camera crossed the wall", p, hit)
	}
	matrix := render.Identity()
	matrix[12] = 20
	w.Objects[0].World = matrix
	if p, hit := w.CameraIntersection(origin, eye); hit || p != eye {
		t.Fatal("camera hit a translated-away collider", p, hit)
	}
	w.Objects = []Instance{{Model: &Model{}, World: render.Identity()}}
	if _, hit := w.CameraIntersection(origin, eye); hit {
		t.Fatal("render geometry substituted for a missing collision mesh")
	}
	w.Objects = []Instance{{Model: m, World: render.Identity(), Mani: true}}
	if _, hit := w.CameraIntersection(origin, eye); hit {
		t.Fatal("static-looking MANI blocked camera")
	}
	w.Objects = nil
	w.rotors = []maniRotor{{model: m, base: render.Identity()}}
	if _, hit := w.CameraIntersection(origin, eye); hit {
		t.Fatal("decorative MANI blocked the camera")
	}
	w.dgColliders = []world.CollisionMesh{mesh}
	if _, hit := w.CameraIntersection(origin, eye); !hit {
		t.Fatal("dungeon collider was ignored")
	}
}

func TestSegmentSphereAndColliderBlocksMatchBruteForce(t *testing.T) {
	if !segmentSphere(render.V3{0, 0, 0}, render.V3{1, 0, 0}, render.V3{0, 0, 0}, 0) {
		t.Fatal("zero radius should fail open")
	}
	if segmentSphere(render.V3{0, 5, 0}, render.V3{10, 0, 0}, render.V3{5, 0, 0}, 1) {
		t.Fatal("disjoint ray hit sphere")
	}
	if !segmentSphere(render.V3{0, 0, 0}, render.V3{10, 0, 0}, render.V3{5, 0.5, 0}, 1) {
		t.Fatal("intersecting ray missed sphere")
	}

	const triCount = 110
	vertices := make([]world.Vec3, 0, triCount*3)
	faces := make([]world.Face, 0, triCount)
	minB := world.Vec3{1e9, 1e9, 1e9}
	maxB := world.Vec3{-1e9, -1e9, -1e9}
	for i := range triCount {
		z := -float32(i+1) * 0.25
		x := float32((i%5)-2) * 1.5
		base := uint16(len(vertices))
		tri := [3]world.Vec3{{x - 1, 0, z}, {x + 1, 0, z}, {x, 3, z}}
		for _, v := range tri {
			vertices = append(vertices, v)
			for a := range 3 {
				minB[a] = min(minB[a], v[a])
				maxB[a] = max(maxB[a], v[a])
			}
		}
		faces = append(faces, world.Face{base, base + 1, base + 2})
	}
	mesh := world.CollisionMesh{Vertices: vertices, Faces: faces, Bounds: world.Box{Min: minB, Max: maxB}}
	blocks := buildColliderBlocks(&mesh)
	if len(blocks) != 4 {
		t.Fatalf("got %d blocks, want 4", len(blocks))
	}

	queries := []struct {
		origin, eye render.V3
		best        float32
	}{
		{render.V3{0, 1, 2}, render.V3{0, 1, -40}, 1},
		{render.V3{-1.5, 1, 2}, render.V3{1.5, 1, -40}, 1},
		{render.V3{25, 1, 2}, render.V3{25, 1, -40}, 1},
		{render.V3{0, 1, 2}, render.V3{0, 1, -40}, 0.1},
	}
	for _, q := range queries {
		delta := q.eye.Sub(q.origin)
		want := colliderSegment(&mesh, q.origin, delta, q.best)
		got := colliderSegmentBlocks(&mesh, blocks, q.origin, delta, q.best)
		if got != want {
			t.Fatalf("query %v->%v (best=%v): got %v, want %v", q.origin, q.eye, q.best, got, want)
		}
	}
}
