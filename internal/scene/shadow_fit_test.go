package scene

import (
	"slices"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/model"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

func shadowFitFixture(vertices int) *Character {
	identity := render.Identity()
	clip := &Clip{ani: &model.ANI{End: 30, Bones: []model.AniBone{{Parent: -1, Trans: []model.TransKey{{Frame: 0, T: [3]float32{}}, {Frame: 30, T: [3]float32{0, 1, 0}}}}}}, restLocal: []render.M4{identity}}
	part := &skinPart{bones: []render.M4{identity}, vertices: make([]render.SkinVertex, vertices)}
	for i := range part.vertices {
		part.vertices[i] = render.SkinVertex{Pos: render.V3{float32(i%29) * .03, float32(i%41) * .04, float32(i%31) * .02}, Weight: 1}
	}
	return &Character{clip: clip, parts: []*skinPart{part}, World: identity}
}

func TestShadowReceiverCacheTracksWorldAndBounds(t *testing.T) {
	w := &World{side: 5, step: 2, heights: []float32{0, 0, 0, 0, 0, 0, 1, 2, 1, 0, 0, 2, 4, 2, 0, 0, 1, 2, 1, 0, 0, 0, 0, 0, 0}}
	w2 := &World{side: 5, step: 2, heights: make([]float32, 25)}
	c := &Character{}
	fit := characterShadowFit{x0: 1, z0: 1, x1: 5, z1: 5}
	check := func(w *World, fit characterShadowFit) {
		t.Helper()
		gotV, gotI := c.shadowReceiver(w, fit)
		wantV, wantI := w.shadowReceiver(fit.x0, fit.z0, fit.x1, fit.z1, fit.origin[1])
		if !slices.Equal(gotV, wantV) || !slices.Equal(gotI, wantI) {
			t.Fatal("cached receiver differs from fresh geometry")
		}
	}
	check(w, fit)
	check(w, fit)
	fit.x0, fit.x1 = 3, 7
	check(w, fit)
	check(w2, fit)
	dungeon := &World{dgShadow: &dungeonShadowIndex{
		triangles: []shadowTriangle{{vertices: [3]render.V3{{0, 0, 0}, {8, 0, 0}, {0, 0, 8}}, min: render.V3{0, 0, 0}, max: render.V3{8, 0, 8}}},
		cells:     map[cellKey][]int{{0, 0}: {0}},
	}}
	c.shadowSeen = make(map[int]bool)
	check(dungeon, fit)
	fit.origin[1] = 100
	check(dungeon, fit)
}

func TestTerrainShadowReceiverReusesAlignedCells(t *testing.T) {
	w := &World{side: 5, step: 2, heights: make([]float32, 25)}
	c := &Character{}
	first := characterShadowFit{x0: 1, z0: -1, x1: 5, z1: 5}
	v, _ := c.shadowReceiver(w, first)
	if len(v) == 0 {
		t.Fatal("fixture has no receiver")
	}
	// Mark the reusable buffer to distinguish an actual hit from regeneration.
	v[0].Normal[0] = 123
	sameCells := characterShadowFit{x0: 1.5, z0: -.5, x1: 5.5, z1: 5.5, origin: render.V3{0, 10, 0}}
	if got, _ := c.shadowReceiver(w, sameCells); got[0].Normal[0] != 123 {
		t.Fatal("changes within the same terrain cells rebuilt the receiver")
	}
	v[0].Normal[0] = 0
	sameCells.x1 = 6.5
	gotV, gotI := c.shadowReceiver(w, sameCells)
	wantV, wantI := w.shadowReceiver(sameCells.x0, sameCells.z0, sameCells.x1, sameCells.z1, sameCells.origin[1])
	if !slices.Equal(gotV, wantV) || !slices.Equal(gotI, wantI) {
		t.Fatal("crossing a terrain cell did not rebuild the receiver")
	}
	for x := -3.75; x < 8; x += .25 {
		fit := characterShadowFit{x0: float32(x), z0: -1.1, x1: float32(x) + 4.1, z1: 5.1, origin: render.V3{0, float32(x), 0}}
		gotV, gotI := c.shadowReceiver(w, fit)
		wantV, wantI := w.shadowReceiver(fit.x0, fit.z0, fit.x1, fit.z1, fit.origin[1])
		if !slices.Equal(gotV, wantV) || !slices.Equal(gotI, wantI) {
			t.Fatalf("receiver mismatch at x=%v", x)
		}
	}
}

func TestTerrainShadowReceiverMatchesHeightInterpolation(t *testing.T) {
	w := &World{side: 5, step: 2, heights: []float32{-3, 2, 5, -1, 4, 1, 3, -2, 8, 6, 0, 7, 2, -5, 1, 3, 2, 4, 1, -6, 1, 3, 9, 2, 7}}
	for _, bounds := range [][4]float32{{0, 0, 8, 8}, {1, 2, 4, 5}, {-2, -2, 4, 4}, {6, 6, 10, 10}} {
		vertices, _ := w.shadowReceiver(bounds[0], bounds[1], bounds[2], bounds[3], 0)
		for _, vertex := range vertices {
			if want := w.Height(vertex.Pos[0], vertex.Pos[2]); vertex.Pos[1] != want {
				t.Fatalf("at %v: got %v, want %v", vertex.Pos, vertex.Pos[1], want)
			}
		}
	}
}

func BenchmarkTerrainShadowReceiver(b *testing.B) {
	for _, tc := range []struct {
		name string
		step float32
	}{{"lattice", 2}, {"interpolated", 2.0001}} {
		b.Run(tc.name, func(b *testing.B) {
			w := &World{side: 129, step: tc.step, heights: make([]float32, 129*129)}
			var vertices []render.Vertex
			var indices []uint32
			b.ReportAllocs()
			for range b.N {
				vertices, indices = w.shadowReceiverInto(61, 61, 67, 67, 0, vertices, indices, nil)
			}
		})
	}
}
func TestShadowFitCacheTracksPoseAndWorld(t *testing.T) {
	c := shadowFitFixture(100)
	first := c.fitShadow()
	if !first.valid {
		t.Fatal("valid pose was not fitted")
	}
	if got := c.fitShadow(); got != first {
		t.Fatal("cached fit changed")
	}
	c.shadowFitReady = false
	if got := c.fitShadow(); got != first {
		t.Fatal("cached fit differs from recomputation")
	}
	c.Time = .5
	animated := c.fitShadow()
	if animated == first {
		t.Fatal("animation did not invalidate fit")
	}
	c.World[12] = 4
	moved := c.fitShadow()
	if moved == animated || moved.origin[0] != 4 {
		t.Fatal("world move did not invalidate fit")
	}
	c.shadowFitReady = false
	if got := c.fitShadow(); got != moved {
		t.Fatal("moved fit differs from recomputation")
	}
	c.clip = nil
	if c.fitShadow().valid {
		t.Fatal("missing clip produced a fit")
	}
}
func BenchmarkShadowFitReuse(b *testing.B) {
	c := shadowFitFixture(10000)
	if !c.fitShadow().valid {
		b.Fatal("invalid fixture")
	}
	b.Run("recompute", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			c.shadowFitReady = false
			c.fitShadow()
		}
	})
	b.Run("cached", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			c.fitShadow()
		}
	})
}
