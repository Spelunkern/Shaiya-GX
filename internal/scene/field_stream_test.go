package scene

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/world"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"testing"
)

func TestTerrainCellKeepsSectionUVAndSharedBorder(t *testing.T) {
	w := &World{WLD: &world.WLD{MapSize: 128, Heights: make([]uint16, 65*65)}}
	for z := 0; z < 65; z++ {
		for x := 0; x < 65; x++ {
			w.WLD.Heights[z*65+x] = 10000 + uint16(x+z)*10
		}
	}
	if !w.heightsOnly() {
		t.Fatal("height fixture")
	}
	left, li := outTerrain(w, 0, 0, 32, 32, 0, 0)
	right, ri := outTerrain(w, 32, 0, 32, 32, 0, 0)
	if len(li) != 32*32*6 || len(ri) != len(li) {
		t.Fatal("cell triangle count")
	}
	for z := 0; z <= 32; z++ {
		a, b := left[z*33+32], right[z*33]
		if a.Pos != b.Pos || a.UV != b.UV || a.UV2 != b.UV2 || a.UV[0] != .5 {
			t.Fatal("cell seam", z, a, b)
		}
		if w.Height(a.Pos[0], a.Pos[2]) != a.Pos[1] {
			t.Fatal("movement height differs from streamed terrain")
		}
	}
}
func TestVisibleCellsMoveAndRemainBounded(t *testing.T) {
	var available []cellKey
	for z := -2; z < 32; z++ {
		for x := -2; x < 32; x++ {
			available = append(available, cellKey{x, z})
		}
	}
	a := desiredCells(available, render.V3{600, 0, 1700}, 500, 64)
	b := desiredCells(available, render.V3{1800, 0, 300}, 500, 64)
	if len(a) == 0 || len(a) >= len(available)/2 || a[0] == b[0] {
		t.Fatal("residency did not follow camera", len(a), len(b))
	}
	// A very wide viewport must enumerate available cells, never an unbounded grid.
	if got := desiredCells(available, render.V3{}, 1e9, 64); len(got) != len(available) {
		t.Fatal("wide viewport coverage")
	}
}
