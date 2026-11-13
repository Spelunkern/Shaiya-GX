package scene

import (
	"fmt"
	"log"
	"math"
	"sort"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/data/world"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

const terrainCellSamples = 32
const cellsPerFrame = 8

type cellKey struct{ x, z int }
type cellObject struct {
	resourceID      int
	drawCenter      [3]float32
	drawRadius      float32
	drawBoundsValid bool
	drawLOD         int
	file            string
	world           render.M4
	rotor           *maniRotor
	vani            bool
	mani            bool
	center          render.V3
	radius          float32
	scale           float32
}
type cellPart struct {
	center               render.V3
	minimum              render.V3
	maximum              render.V3
	radius               float32
	section              int
	terrain              Section
	x, z, nx, nz, sx, sz int
	batch                terrainBatchKey
	rangeInBatch         render.TerrainRange
}
type terrainBatchKey struct{ x, z, section int }
type terrainBatch struct {
	mesh     *render.Mesh
	material Section
	ranges   []render.TerrainRange
}
type fieldCell struct {
	parts   []cellPart
	objects []int
}
type modelBatchKey struct {
	file string
	mani bool
}

type fieldDrawEntry struct {
	lod    int
	sector int
	object *cellObject
	batch  *fieldModelBatch
	vani   *vaniAsset
	center render.V3
	radius float32
}
type fieldModelBatch struct {
	prepared  uint64
	lodWorlds [2][]render.M4
	key       modelBatchKey
	model     *Model
	worlds    []render.M4
}

type fieldStream struct {
	resources                          []fieldResource
	resourceIndex                      map[string]int
	resourceObjectCount                int
	objectMarks, resourceMarks         []bool
	batchPreparation                   uint64
	lastPrepare, lastTerrain, lastTrim time.Duration
	lodInstances                       int
	submittedTriangles                 int
	drawSectors                        []fieldDrawSector
	objectTests                        int
	disableSectors                     bool // reference path for image/performance comparisons
	loader                             *fieldLoader
	revision                           uint64
	drawEntries                        []fieldDrawEntry
	modelBatches                       []*fieldModelBatch
	batches                            map[modelBatchKey]*fieldModelBatch
	world                              *World
	assets                             *Assets // owns static field resources, independent of avatars/login scenes
	stem                               string
	index                              map[cellKey][]int
	objects                            []cellObject
	vanis                              map[string]*vaniAsset
	global                             []int
	cells                              map[cellKey]*fieldCell
	materials                          map[int]Section
	terrainBatches                     map[terrainBatchKey]*terrainBatch
	visibleTerrain                     []*terrainBatch
	terrainVertices                    []render.Vertex
	terrainIndices                     []uint32
	order, pending                     []cellKey
	drawObjects                        []int
	center                             cellKey
	radius                             float32
	available                          []cellKey
	planned                            bool
}

func loadStreamedField(parent *Assets, stem string) (*World, error) {
	b, err := parent.Arc.ReadFile("world/" + stem + ".wld")
	if err != nil {
		return nil, err
	}
	data, err := world.ParseWLD(b)
	if err != nil {
		return nil, err
	}
	w := &World{WLD: data, FogColor: render.V3{data.FogColor[0] / 255, data.FogColor[1] / 255, data.FogColor[2] / 255}, FogStart: data.FogStart, FogEnd: data.FogEnd}
	if !w.heightsOnly() {
		return nil, fmt.Errorf("field %s: invalid height grid", stem)
	}
	s := &fieldStream{world: w, assets: NewAssets(parent.Arc), stem: stem, index: map[cellKey][]int{}, cells: map[cellKey]*fieldCell{}, materials: map[int]Section{}, vanis: map[string]*vaniAsset{}, terrainBatches: map[terrainBatchKey]*terrainBatch{}}
	w.stream = s
	s.indexObjects()
	s.indexResources()
	w.loadMapEffects(parent)
	available := map[cellKey]bool{}
	side := (w.side-2)/terrainCellSamples + 1
	for z := 0; z < side; z++ {
		for x := 0; x < side; x++ {
			available[cellKey{x, z}] = true
		}
	}
	for key := range s.index {
		available[key] = true
	}
	for key := range available {
		s.available = append(s.available, key)
	}
	w.Sky = LoadSky(s.assets, data)
	w.water = loadWater(parent, w)
	return w, nil
}
func (s *fieldStream) size() float32 { return s.world.step * terrainCellSamples }

// desiredCells includes a full-cell margin so motion within the camera cell
// cannot expose missing geometry before the next residency update.
func desiredCells(available []cellKey, eye render.V3, radius, size float32) []cellKey {
	cx, cz := int(math.Floor(float64(eye[0]/size))), int(math.Floor(float64(eye[2]/size)))
	keys := make([]cellKey, 0, len(available))
	limit := radius + size*1.5
	for _, key := range available {
		dx, dz := float32(key.x-cx)*size, float32(key.z-cz)*size
		if dx*dx+dz*dz <= limit*limit {
			keys = append(keys, key)
		}
	}
	sort.Slice(keys, func(i, j int) bool {
		a, b := keys[i], keys[j]
		da, db := (a.x-cx)*(a.x-cx)+(a.z-cz)*(a.z-cz), (b.x-cx)*(b.x-cx)+(b.z-cz)*(b.z-cz)
		if da != db {
			return da < db
		}
		if a.z != b.z {
			return a.z < b.z
		}
		return a.x < b.x
	})
	return keys
}

// UpdateVisible runs only on the GL owner thread, after camera setup.
func (w *World) UpdateVisible(eye render.V3, radius float32) {
	s := w.stream
	if s == nil {
		return
	}
	center := cellKey{int(math.Floor(float64(eye[0] / s.size()))), int(math.Floor(float64(eye[2] / s.size())))}
	changed := !s.planned || center != s.center || radius != s.radius
	dirtyTerrain := make(map[terrainBatchKey]bool)
	if changed {
		if s.loader != nil {
			s.loader.reset()
		}
		s.center, s.radius, s.planned = center, radius, true
		s.order, s.pending = nil, nil
		wanted := map[cellKey]bool{}
		for _, key := range desiredCells(s.available, eye, radius, s.size()) {
			inside := key.x >= 0 && key.z >= 0 && key.x*terrainCellSamples < w.side-1 && key.z*terrainCellSamples < w.side-1
			if !inside && len(s.index[key]) == 0 {
				continue
			}
			wanted[key] = true
			s.order = append(s.order, key)
			if s.cells[key] == nil {
				s.pending = append(s.pending, key)
			}
		}
		for key, c := range s.cells {
			if !wanted[key] {
				for _, p := range c.parts {
					dirtyTerrain[p.batch] = true
				}
				delete(s.cells, key)
			}
		}
	}
	workStart := time.Now()
	deadline := workStart.Add(2 * time.Millisecond)
	if s.loader != nil {
		s.pollAssets(deadline)
	}
	for i := 0; i < cellsPerFrame && len(s.pending) > 0; i++ {
		if s.loader != nil && (!s.prepareCellAssets(s.pending[0], deadline) || !time.Now().Before(deadline)) {
			break
		}
		key := s.pending[0]
		s.pending = s.pending[1:]
		s.cells[key] = s.loadCell(key)
		for _, p := range s.cells[key].parts {
			dirtyTerrain[p.batch] = true
		}
		changed = true
	}
	s.lastPrepare = time.Since(workStart)
	if changed {
		s.revision++
		t := time.Now()
		s.rebuildTerrainBatches(dirtyTerrain)
		s.lastTerrain = time.Since(t)
		t = time.Now()
		s.trim()
		s.lastTrim = time.Since(t)
	}
	for file := range s.assets.Missing {
		log.Printf("field %s: missing %s", s.stem, file)
		delete(s.assets.Missing, file)
	}
}
func (w *World) Ready() bool {
	return w == nil || w.stream == nil || w.stream.planned && len(w.stream.pending) == 0
}

func (s *fieldStream) loadCell(key cellKey) *fieldCell {
	w := s.world
	c := &fieldCell{objects: s.index[key]}
	x0, z0 := key.x*terrainCellSamples, key.z*terrainCellSamples
	x1, z1 := min(x0+terrainCellSamples, w.side-1), min(z0+terrainCellSamples, w.side-1)
	n := w.sectionsPerSide()
	section := (w.side - 1) / n
	if x0 >= 0 && z0 >= 0 && x0 < x1 && z0 < z1 {
		for sz := 0; sz < n; sz++ {
			for sx := 0; sx < n; sx++ {
				ax, az, bx, bz := max(x0, sx*section), max(z0, sz*section), min(x1, (sx+1)*section), min(z1, (sz+1)*section)
				if ax >= bx || az >= bz {
					continue
				}
				id := sz*n + sx
				mat, ok := s.materials[id]
				if !ok {
					mat = s.loadMaterial(sx, sz)
					s.materials[id] = mat
				}
				lo := render.V3{float32(ax) * w.step, w.heights[az*w.side+ax], float32(az) * w.step}
				hi := render.V3{float32(bx) * w.step, lo[1], float32(bz) * w.step}
				for z := az; z <= bz; z++ {
					for x := ax; x <= bx; x++ {
						height := w.heights[z*w.side+x]
						lo[1], hi[1] = min(lo[1], height), max(hi[1], height)
					}
				}
				c.parts = append(c.parts, cellPart{section: id, terrain: mat, minimum: lo, maximum: hi, center: lo.Add(hi).Scale(.5), radius: hi.Sub(lo).Len() * .5, x: ax, z: az, nx: bx - ax, nz: bz - az, sx: sx, sz: sz, batch: terrainBatchKey{key.x / 4, key.z / 4, id}})
			}
		}
	}
	for _, id := range append(append([]int(nil), c.objects...), s.global...) {
		object := s.objects[id]
		file := object.file
		if object.vani {
			if _, ok := s.vanis[file]; !ok {
				asset, err := loadVani(s.assets, file)
				if err != nil {
					log.Printf("field vani %s: %v", file, err)
				}
				s.vanis[file] = asset
			}
			continue
		}
		if _, ok := s.assets.models[file]; ok {
			continue
		}
		if _, err := s.assets.SMOD(file); err != nil {
			log.Printf("field model %s: %v", file, err)
			s.assets.models[file] = nil
		}
	}
	return c
}
