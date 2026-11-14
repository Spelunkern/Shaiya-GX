package scene

import (
	"encoding/json"
	"math"
	"os"
	"slices"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/world"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

func TestTerrainSurfaceMatchesNativeHeightOracle(t *testing.T) {
	var oracle struct {
		Fixtures []struct {
			Heights []uint16
			Queries [][3]float32
		}
	}
	raw, e := os.ReadFile("testdata/terrain_native.json")
	if e != nil {
		t.Fatal(e)
	}
	if e = json.Unmarshal(raw, &oracle); e != nil {
		t.Fatal(e)
	}
	for i, fixture := range oracle.Fixtures {
		source := slices.Clone(fixture.Heights)
		w := &World{WLD: &world.WLD{MapSize: 32, Heights: fixture.Heights}}
		if !w.heightsOnly() {
			t.Fatal("invalid fixture")
		}
		for _, q := range fixture.Queries {
			if got := w.Height(q[0], q[1]); math.Abs(float64(got-q[2])) > .001 {
				t.Fatalf("fixture%d at%v: got%v, native%v", i, q, got, q[2])
			}
		}
		if !slices.Equal(source, w.WLD.Heights) {
			t.Fatal("source heights modified")
		}
	}
}

func TestTerrainMeshMatchesNativeFinestTopology(t *testing.T) {
	var oracle struct {
		Heights  []uint16
		Vertices [][3]float32
		Indices  []uint32
	}
	raw, e := os.ReadFile("testdata/terrain_mesh_native.json")
	if e != nil {
		t.Fatal(e)
	}
	if e = json.Unmarshal(raw, &oracle); e != nil {
		t.Fatal(e)
	}
	w := &World{WLD: &world.WLD{MapSize: 32, Heights: oracle.Heights}}
	w.heightsOnly()
	v, idx := outTerrain(w, 0, 0, 16, 16, 0, 0)
	if len(idx) != len(oracle.Indices) || len(v) != len(oracle.Vertices) {
		t.Fatal("native mesh size differs", len(v), len(idx))
	}
	// Native layers cache vertices in traversal order; compare actual triangles.
	key := func(a, b, c render.V3) [3]render.V3 {
		p := []render.V3{a, b, c}
		slices.SortFunc(p, func(a, b render.V3) int {
			for k := range a {
				if a[k] < b[k] {
					return -1
				}
				if a[k] > b[k] {
					return 1
				}
			}
			return 0
		})
		return [3]render.V3{p[0], p[1], p[2]}
	}
	wanted := map[[3]render.V3]int{}
	for i := 0; i < len(oracle.Indices); i += 3 {
		var p [3]render.V3
		for j := range p {
			n := oracle.Vertices[oracle.Indices[i+j]]
			p[j] = render.V3{n[0] * 2, (n[1] - 10000) / 50, n[2] * 2}
		}
		wanted[key(p[0], p[1], p[2])]++
	}
	for i := 0; i < len(idx); i += 3 {
		k := key(v[idx[i]].Pos, v[idx[i+1]].Pos, v[idx[i+2]].Pos)
		if wanted[k] == 0 {
			t.Fatal("triangle absent from original", k)
		}
		wanted[k]--
	}
	for _, n := range wanted {
		if n != 0 {
			t.Fatal("native triangle missing")
		}
	}
	// Source (1,1) spike disappears; authored (2,2) centre remains untouched.
	if w.Height(2, 2) != 40 || w.Height(4, 4) != 80 {
		t.Fatal("wrong ignored/retained sample", w.Height(2, 2), w.Height(4, 4))
	}
	checkMeshHeights(t, w, v, idx)
	sv, si := w.shadowReceiver(0, 0, 32, 32, 0)
	checkMeshHeights(t, w, sv, si)
}

func checkMeshHeights(t *testing.T, w *World, v []render.Vertex, idx []uint32) {
	t.Helper()
	for i := 0; i < len(idx); i += 3 {
		a, b, c := v[idx[i]].Pos, v[idx[i+1]].Pos, v[idx[i+2]].Pos
		for _, weights := range [][3]float32{{.25, .25, .5}, {.5, .25, .25}, {.25, .5, .25}} {
			p := a.Scale(weights[0]).Add(b.Scale(weights[1])).Add(c.Scale(weights[2]))
			if h := w.Height(p[0], p[2]); math.Abs(float64(h-p[1])) > .0001 {
				t.Fatalf("mesh and height differ: %v != %v", p, h)
			}
		}
	}
}

func TestTerrainFanEdgesClampAndKeepPlanarSlopes(t *testing.T) {
	h := make([]uint16, 17*17)
	for z := 0; z < 17; z++ {
		for x := 0; x < 17; x++ {
			h[z*17+x] = uint16(10000 + 50*x + 100*z)
		}
	}
	w := &World{WLD: &world.WLD{MapSize: 32, Heights: h}}
	w.heightsOnly()
	for _, p := range [][2]float32{{0, 0}, {32, 32}, {32, 12}, {12, 32}, {-.1, 5}, {36, 36}, {7.25, 18.75}} {
		want := min(max(p[0], 0), 32)/2 + min(max(p[1], 0), 32)
		if got := w.Height(p[0], p[1]); got != want {
			t.Fatal(p, got, want)
		}
	}
}
