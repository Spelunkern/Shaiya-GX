package scene

import (
	"math"
	"os"
	"strings"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/data/model"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

func TestRealSkinBoundsEncloseWeightedVertices(t *testing.T) {
	dir := os.Getenv("SHAIYA_DATA")
	if dir == "" {
		t.Skip("requires original assets")
	}
	a, err := archive.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	meshes, vertices, fallback, unparsed := 0, 0, 0, 0
	a.Walk(func(file string, f *archive.File) {
		if !strings.HasSuffix(strings.ToLower(file), ".3dc") {
			return
		}
		raw, e := a.Read(f)
		if e != nil {
			t.Fatal(e)
		}
		m, e := model.Parse3DC(raw)
		if e != nil {
			unparsed++
			return
		}
		v := make([]render.SkinVertex, len(m.Vertices))
		for i, x := range m.Vertices {
			v[i] = render.SkinVertex{Pos: render.V3(x.Pos), Weight: x.Weight, Bone0: float32(x.Bones[0]), Bone1: float32(x.Bones[1])}
		}
		bounds, ok := makeBoneBounds(v, len(m.Bones))
		if !ok {
			fallback++
			return
		}
		transforms := make([]render.M4, len(bounds))
		lo, hi := render.V3{}, render.V3{}
		has := false
		for i, box := range bounds {
			angle := float64(i) * .37
			transform := render.Basis(render.V3{float32(i) * .3, float32(i % 5), float32(i % 7)}, render.V3{float32(math.Sin(angle)), 0, float32(math.Cos(angle))}, render.V3{0, 1, 0})
			if len(m.Bones) == 0 {
				transform = render.Identity()
			}
			transforms[i] = transform
			if box.used && !unionTransformedBox(&lo, &hi, &has, box.min, box.max, transform) {
				t.Fatalf("%s invalid transform", file)
			}
		}
		for _, v := range v {
			var p render.V3
			if len(m.Bones) == 0 {
				p = v.Pos
			} else {
				m0 := transforms[min(int(v.Bone0), len(transforms)-1)]
				m1 := transforms[min(int(v.Bone1), len(transforms)-1)]
				p = transformPoint(v.Pos, m0).Scale(v.Weight).Add(transformPoint(v.Pos, m1).Scale(1 - v.Weight))
			}
			for axis := range 3 {
				if p[axis] < lo[axis]-.001 || p[axis] > hi[axis]+.001 {
					t.Fatalf("%s point %v outside %v..%v", file, p, lo, hi)
				}
			}
			vertices++
		}
		meshes++
	})
	if meshes == 0 || vertices == 0 {
		t.Fatal("no original skinned geometry")
	}
	t.Logf("checked %d meshes and %d vertices; %d conservative fallbacks, %d unparsed", meshes, vertices, fallback, unparsed)
}
