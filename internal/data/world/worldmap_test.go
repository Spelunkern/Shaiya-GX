package world

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRealWorldMapRegions(t *testing.T) {
	dir := os.Getenv("SHAIYA_DATA")
	if dir == "" {
		t.Skip("resources")
	}
	b, err := os.ReadFile(filepath.Join(dir, "WorldMap.cfg"))
	if err != nil {
		t.Fatal(err)
	}
	regions, err := ParseWorldMap(b)
	if err != nil {
		t.Fatal(err)
	}
	if len(regions) != 19 || regions[0].Map != 19 || regions[1].Map != 1 || regions[2].Top != 208 {
		t.Fatal("native rows or decimal zero prefix lost", regions)
	}
	for _, r := range regions {
		if !r.Contains(float32(r.Left+r.Right)/2, float32(r.Top+r.Bottom)/2) {
			t.Fatal("authored region not hittable", r)
		}
		if r.Country == 1 && r.Available(true) || r.Country == 2 && r.Available(false) {
			t.Fatal("opposing faction preview allowed", r)
		}
	}
}
