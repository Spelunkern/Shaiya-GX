package scene

import (
	"math"
	"math/rand/v2"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/model"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

func TestBoneBoundsEncloseWeightedVertices(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	vertices := make([]render.SkinVertex, 2000)
	for i := range vertices {
		v := &vertices[i]
		v.Pos = render.V3{rng.Float32()*12 - 6, rng.Float32()*8 - 4, rng.Float32()*10 - 5}
		v.Weight = rng.Float32()
		v.Bone0 = float32(rng.IntN(12))
		v.Bone1 = float32(rng.IntN(12))
	}
	bounds, ok := makeBoneBounds(vertices, 8)
	if !ok {
		t.Fatal("valid weights rejected")
	}
	poses := make([]render.M4, 8)
	lo, hi := render.V3{}, render.V3{}
	has := false
	for i := range poses {
		angle := rng.Float32() * 6
		poses[i] = render.Basis(render.V3{rng.Float32()*20 - 10, rng.Float32() * 10, rng.Float32()*20 - 10}, render.V3{float32(math.Sin(float64(angle))), 0, float32(math.Cos(float64(angle)))}, render.V3{0, 1, 0})
		box := bounds[i]
		if box.used && !unionTransformedBox(&lo, &hi, &has, box.min, box.max, poses[i]) {
			t.Fatal("valid transform rejected")
		}
	}
	for _, v := range vertices {
		m0 := poses[min(int(v.Bone0), 7)]
		m1 := poses[min(int(v.Bone1), 7)]
		p := transformPoint(v.Pos, m0).Scale(v.Weight).Add(transformPoint(v.Pos, m1).Scale(1 - v.Weight))
		for axis := range 3 {
			if p[axis] < lo[axis]-1e-4 || p[axis] > hi[axis]+1e-4 {
				t.Fatalf("weighted point %v outside %v..%v", p, lo, hi)
			}
		}
	}
	if _, ok := makeBoneBounds([]render.SkinVertex{{Weight: 1.1}}, 1); ok {
		t.Fatal("nonconvex weight accepted")
	}
}

func TestCharacterVisibilityIncludesAttachmentsAndUpdatesPose(t *testing.T) {
	identity := render.Identity()
	verts := []render.SkinVertex{{Pos: render.V3{0, 0, 0}, Weight: 1}, {Pos: render.V3{0, 1, 0}, Weight: 1}}
	bounds, ok := makeBoneBounds(verts, 1)
	if !ok {
		t.Fatal("invalid test bounds")
	}
	p := &skinPart{bones: []render.M4{identity}, vertices: verts, bounds: bounds, bounded: true}
	clip := &Clip{ani: &model.ANI{Bones: []model.AniBone{{Parent: -1}}}, restLocal: []render.M4{identity}}
	c := &Character{parts: []*skinPart{p}, clip: clip, World: identity}
	// A bounded weapon extends from x=0 to x=20 and must keep the actor visible.
	weapon := &Part{bounded: true, boundMin: render.V3{}, boundMax: render.V3{20, 0, 0}}
	c.attachments = []attachment{{part: weapon, bone: 0, extra: identity}}
	frustum := render.ViewFrustum(render.LookAtLH(render.V3{19, 1, -5}, render.V3{19, 1, 0}, render.V3{0, 1, 0}), render.PerspectiveFovLH(math.Pi/4, 1, .1, 100))
	if !c.Visible(frustum, [4]float32{}) {
		t.Fatal("visible attachment omitted")
	}
	if len(p.pose) != 1 {
		t.Fatal("culling did not update pose for labels/shadows")
	}
	c.Mounted = true
	if c.Visible(frustum, [4]float32{}) {
		t.Fatal("mounted actor incorrectly retained hidden equipment")
	}
	c.Mounted = false
	c.World[12] = 100
	if c.Visible(frustum, [4]float32{}) {
		t.Fatal("offscreen actor remained visible after moving")
	}
	c.World = identity
	weapon.bounded = false
	c.boundReady = false
	if !c.Visible(frustum, [4]float32{}) {
		t.Fatal("unknown attachment must fall back to drawing")
	}
}

func TestSortedKeyframeSamplingMatchesLinear(t *testing.T) {
	rots := make([]model.RotKey, 20)
	trans := make([]model.TransKey, 20)
	for i := range rots {
		frame := uint32(i * 3)
		if i == 7 {
			frame = rots[i-1].Frame // duplicate frame edge case
		}
		ang := float64(i) * 0.12
		rots[i] = model.RotKey{Frame: frame, Q: [4]float32{0, float32(math.Sin(ang)), 0, float32(math.Cos(ang))}}
		trans[i] = model.TransKey{Frame: frame, T: [3]float32{float32(i), float32(i) * 0.5, -float32(i)}}
	}
	if !rotKeysSorted(rots) || !transKeysSorted(trans) {
		t.Fatal("monotonic keyframes should be marked sorted")
	}
	unsorted := append([]model.RotKey(nil), rots...)
	unsorted[10].Frame = 1
	if rotKeysSorted(unsorted) {
		t.Fatal("unsorted keyframes must not use binary search")
	}
	samples := []float32{-5, 0, 1.5, 18, 18.01, 30, 57, 100}
	for _, f := range samples {
		if got, want := sampleRotSorted(rots, f), sampleRot(rots, f); got != want {
			t.Fatalf("rot at %v: got %v, want %v", f, got, want)
		}
		if got, want := sampleTransSorted(trans, f), sampleTrans(trans, f); got != want {
			t.Fatalf("trans at %v: got %v, want %v", f, got, want)
		}
	}
}
