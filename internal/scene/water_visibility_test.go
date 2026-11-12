package scene

import (
	"math"
	"os"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/data/world"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

func dryWaterFixture() (*World, render.M4, render.M4) {
	w := &World{WLD: &world.WLD{MapSize: 128}, side: 65, step: 2, heights: make([]float32, 65*65)}
	for i := range w.heights {
		w.heights[i] = 10
	}
	w.stream = &fieldStream{world: w, planned: true, radius: 1000, center: cellKey{1, 1}, materials: map[int]Section{0: {Layers: []TerrainLayer{{Mask: &render.Texture{}, Tile: &render.Texture{}}}}}}
	w.water = &waterSurface{}
	w.water.indexTerrain(w)
	eye := render.V3{64, 12, 64}
	view := render.LookAtLH(eye, render.V3{64, 0, 90}, render.V3{0, 1, 0})
	proj := render.PerspectiveFovLH(math.Pi/4, 1, .1, 100)
	return w, view, proj
}
func TestWaterVisibilityCullsOnlyConservatively(t *testing.T) {
	w, view, proj := dryWaterFixture()
	if !render.ViewFrustum(view, proj).BoxVisible(render.V3{}, render.V3{128, 0, 128}) {
		t.Fatal("fixture does not intersect water plane")
	}
	if w.WaterVisible(view, proj) {
		t.Fatal("completely dry terrain did not occlude water")
	}
	w.stream.pending = []cellKey{{0, 0}}
	if !w.WaterVisible(view, proj) {
		t.Fatal("culled while terrain is still loading")
	}
	w.stream.pending = nil
	w.stream.radius = 1
	if !w.WaterVisible(view, proj) {
		t.Fatal("culled with insufficient resident radius")
	}
	w.stream.radius = 1000
	w.stream.materials[0] = Section{}
	if !w.WaterVisible(view, proj) {
		t.Fatal("culled without an opaque terrain layer")
	}
	w.stream.materials[0] = Section{Layers: []TerrainLayer{{Mask: &render.Texture{}, Tile: &render.Texture{}}}}
	below := render.LookAtLH(render.V3{64, 9, 64}, render.V3{64, 0, 90}, render.V3{0, 1, 0})
	if !w.WaterVisible(below, proj) {
		t.Fatal("culled from below terrain")
	}
	for i := range w.heights {
		w.heights[i] = -1
	}
	w.water.indexTerrain(w)
	if !w.WaterVisible(view, proj) {
		t.Fatal("visible wet terrain was culled")
	}
}
func TestWaterIndexIncludesSharedVertex(t *testing.T) {
	w, _, _ := dryWaterFixture()
	w.heights[32*w.side+32] = 0
	w.water.indexTerrain(w)
	if len(w.water.wetCells) != 4 {
		t.Fatalf("shared wet corner indexed in %d cells", len(w.water.wetCells))
	}
}

var waterVisibilityResult bool

func BenchmarkWaterVisibilityReal(b *testing.B) {
	dir := os.Getenv("SHAIYA_DATA")
	if dir == "" {
		b.Skip("requires original assets")
	}
	arc, err := archive.Open(dir)
	if err != nil {
		b.Fatal(err)
	}
	defer arc.Close()
	raw, err := arc.ReadFile("world/1.wld")
	if err != nil {
		b.Fatal(err)
	}
	data, err := world.ParseWLD(raw)
	if err != nil {
		b.Fatal(err)
	}
	w := &World{WLD: data, FogEnd: data.FogEnd}
	if !w.heightsOnly() {
		b.Fatal("invalid terrain")
	}
	w.stream = &fieldStream{world: w, planned: true, radius: 10000, materials: map[int]Section{0: {Layers: []TerrainLayer{{Mask: &render.Texture{}, Tile: &render.Texture{}}}}}}
	w.water = &waterSurface{}
	w.water.indexTerrain(w)
	proj := render.PerspectiveFovLH(math.Pi/4, 1, .1, max(400, w.FogEnd))
	dry := render.V3{128, w.Height(128, 128) + 3, 128}
	wet := render.V3{600, 2, 620}
	for _, tc := range []struct {
		name     string
		eye, dir render.V3
		want     bool
	}{{"dry", dry, render.V3{0, -1, 1}, false}, {"wet", wet, render.V3{0, 0, 1}, true}} {
		b.Run(tc.name, func(b *testing.B) {
			w.stream.center = cellKey{int(tc.eye[0] / w.stream.size()), int(tc.eye[2] / w.stream.size())}
			view := render.LookAtLH(tc.eye, tc.eye.Add(tc.dir), render.V3{0, 1, 0})
			if got := w.WaterVisible(view, proj); got != tc.want {
				b.Fatalf("unexpected water visibility: %v", got)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				waterVisibilityResult = w.WaterVisible(view, proj)
			}
		})
	}
}
