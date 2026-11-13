package scene

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/world"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"math"
	"reflect"
	"testing"
)

func TestStaticLODReducesDenseMeshWithoutMutatingOriginal(t *testing.T) {
	mesh := world.SMODMesh{}
	const side = 41
	for z := 0; z < side; z++ {
		for x := 0; x < side; x++ {
			mesh.Vertices = append(mesh.Vertices, world.Vertex{Pos: world.Vec3{float32(x) / 40, 0, float32(z) / 40}, Normal: world.Vec3{0, 1, 0}})
		}
	}
	for z := 0; z < side-1; z++ {
		for x := 0; x < side-1; x++ {
			a := uint16(z*side + x)
			mesh.Faces = append(mesh.Faces, world.Face{a, a + 1, a + side}, world.Face{a + 1, a + side + 1, a + side})
		}
	}
	original := append([]world.Vertex(nil), mesh.Vertices...)
	lod := simplifyStatic(mesh, 16)
	if len(lod.indices) == 0 || len(lod.indices) >= len(mesh.Faces)*3 || lod.error <= 0 || lod.error > float32(math.Sqrt(2)/16) {
		t.Fatal("invalid reduction/error", len(lod.indices), lod.error)
	}
	if !reflect.DeepEqual(original, mesh.Vertices) {
		t.Fatal("LOD changed original geometry")
	}
	for _, i := range lod.indices {
		if int(i) >= len(lod.vertices) {
			t.Fatal("out-of-range LOD index")
		}
	}
}
func TestStaticLODUsesScreenErrorAndExcludesOtherPasses(t *testing.T) {
	model := &Model{lodAvailable: [2]bool{true, true}, lodErrors: [2]float32{.025, .05}}
	e := fieldDrawEntry{object: &cellObject{scale: 1}, batch: &fieldModelBatch{model: model}}
	s := &render.Scene3D{StaticObjectLOD: true, ViewportW: 1280, ViewportH: 720, View: render.Identity(), Proj: render.PerspectiveFovLH(math.Pi/4, 16.0/9, .1, 1000)}
	if n := e.chooseLOD(s, render.V3{0, 0, 300}, 1); n != 2 {
		t.Fatal("distant mesh did not reduce", n)
	}
	if n := e.chooseLOD(s, render.V3{0, 0, 10}, 1); n != 0 {
		t.Fatal("near mesh lost detail", n)
	}
	s.Shadow = true
	if e.chooseLOD(s, render.V3{0, 0, 300}, 1) != 0 {
		t.Fatal("shadow used display LOD")
	}
	s.Shadow = false
	s.ClipPlane = [4]float32{0, 1, 0, 0}
	if e.chooseLOD(s, render.V3{0, 0, 300}, 1) != 0 {
		t.Fatal("reflection inherited display LOD")
	}
}
