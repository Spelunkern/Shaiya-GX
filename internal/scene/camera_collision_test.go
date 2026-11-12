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
	w.dgColliders = []world.CollisionMesh{mesh}
	if _, hit := w.CameraIntersection(origin, eye); !hit {
		t.Fatal("dungeon collider was ignored")
	}
}
