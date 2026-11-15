package collision

import (
	"math/rand"
	"sync"
	"testing"
)

func visitFixture() *Index {
	r := rand.New(rand.NewSource(7))
	idx := NewIndex()
	for range 4000 {
		x, z := r.Float64()*512, r.Float64()*512
		w := 1 + r.Float64()*40 // many triangles span several cells
		idx.Add(Triangle{{x, 0, z}, {x + w, 1, z}, {x, 2, z + w}})
	}
	idx.Add(Triangle{{-1e12, 0, 0}, {1e12, 0, 0}, {0, 0, 1e12}}) // global
	return idx
}

func bruteVisit(idx *Index, b Box) map[Triangle]int {
	out := map[Triangle]int{}
	for _, t := range idx.triangles {
		if b.Overlaps(t.Bounds()) {
			out[t]++
		}
	}
	return out
}

func TestVisitReportsEachOverlapOnceAcrossGoroutines(t *testing.T) {
	idx := visitFixture()
	boxes := []Box{{Min: Vec{0, -10, 0}, Max: Vec{64, 10, 64}}, {Min: Vec{200, -10, 300}, Max: Vec{260, 10, 330}}, {Min: Vec{-1, -1, -1}, Max: Vec{1, 1, 1}}}
	var wg sync.WaitGroup
	for g := range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for n := range 200 {
				b := boxes[(g+n)%len(boxes)]
				got := map[Triangle]int{}
				idx.Visit(b, func(t Triangle) { got[t]++ })
				want := bruteVisit(idx, b)
				if len(got) != len(want) {
					t.Errorf("box %v: %d triangles, want %d", b, len(got), len(want))
					return
				}
				for tri, c := range got {
					if c != 1 || want[tri] == 0 {
						t.Errorf("triangle %v visited %d times (want once, overlapping)", tri, c)
						return
					}
				}
			}
		}()
	}
	wg.Wait()
}

func BenchmarkVisitSmallBox(b *testing.B) {
	idx := visitFixture()
	box := Box{Min: Vec{100, -10, 100}, Max: Vec{104, 10, 104}}
	n := 0
	b.ReportAllocs()
	for b.Loop() {
		idx.Visit(box, func(Triangle) { n++ })
	}
}
