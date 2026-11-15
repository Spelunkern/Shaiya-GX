package collision

import (
	"math"
	"sync"
)

const cellSize = 16

// Index is an immutable uniform-grid broad phase. Queries own their scratch
// state; a prepared index can be read concurrently without render residency.
type Index struct {
	triangles []Triangle
	cells     map[[2]int][]int
	global    []int
	// marks recycles each query's visited set (see Visit); a pool keeps
	// concurrent readers independent.
	marks sync.Pool
}

// visitMarks is a visited set by generation: a triangle was seen in this
// query when its stamp equals epoch, so queries never clear or allocate.
type visitMarks struct {
	stamp []uint32
	epoch uint32
}

func NewIndex() *Index { return &Index{cells: map[[2]int][]int{}} }
func (i *Index) Len() int {
	if i == nil {
		return 0
	}
	return len(i.triangles)
}
func (i *Index) Add(t Triangle) {
	if !t.Valid() {
		return
	}
	id := len(i.triangles)
	i.triangles = append(i.triangles, t)
	b := t.Bounds()
	if !gridBox(b) {
		i.global = append(i.global, id)
		return
	}
	x0, x1, z0, z1 := cells(b)
	if (x1-x0+1)*(z1-z0+1) > 64 {
		i.global = append(i.global, id)
		return
	}
	for z := z0; z <= z1; z++ {
		for x := x0; x <= x1; x++ {
			key := [2]int{x, z}
			i.cells[key] = append(i.cells[key], id)
		}
	}
}
func cells(b Box) (int, int, int, int) {
	return int(math.Floor(b.Min[0] / cellSize)), int(math.Floor(b.Max[0] / cellSize)), int(math.Floor(b.Min[2] / cellSize)), int(math.Floor(b.Max[2] / cellSize))
}
func (i *Index) Visit(b Box, visit func(Triangle)) {
	if i == nil {
		return
	}
	// Triangles spanning several cells are listed in each; a per-query map
	// to skip repeats allocated on every call, so reuse generation marks.
	m, _ := i.marks.Get().(*visitMarks)
	if m == nil || len(m.stamp) != len(i.triangles) {
		m = &visitMarks{stamp: make([]uint32, len(i.triangles))}
	}
	m.epoch++
	if m.epoch == 0 {
		clear(m.stamp)
		m.epoch = 1
	}
	defer i.marks.Put(m)
	emit := func(id int) {
		if m.stamp[id] == m.epoch {
			return
		}
		m.stamp[id] = m.epoch
		t := i.triangles[id]
		if b.Overlaps(t.Bounds()) {
			visit(t)
		}
	}
	for _, id := range i.global {
		emit(id)
	}
	if !gridBox(b) {
		for id := range i.triangles {
			emit(id)
		}
		return
	}
	x0, x1, z0, z1 := cells(b)
	if (x1-x0+1)*(z1-z0+1) > 4096 {
		for id := range i.triangles {
			emit(id)
		}
		return
	}
	for z := z0; z <= z1; z++ {
		for x := x0; x <= x1; x++ {
			for _, id := range i.cells[[2]int{x, z}] {
				emit(id)
			}
		}
	}
}

func gridBox(b Box) bool {
	return b.Min.Finite() && b.Max.Finite() && math.Abs(b.Min[0]) < 1e9 && math.Abs(b.Max[0]) < 1e9 && math.Abs(b.Min[2]) < 1e9 && math.Abs(b.Max[2]) < 1e9
}
