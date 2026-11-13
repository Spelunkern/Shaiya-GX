package render

import "math"

// TerrainOcclusion is a conservative CPU depth mask for the CURRENT camera.
// A pixel is written only when a complete coarse pixel is covered by an
// occluder triangle. Depth is the farthest reciprocal W across that pixel.
// The scene supplies terrain lower envelopes or actual opaque dungeon triangles.
// There is no previous-frame visibility, GPU readback or query wait.
type TerrainOcclusion struct {
	Width, Height        int
	view, projection, vp M4
	depth                []float32
	covered              bool
	Culled               int
}

func NewTerrainOcclusion(w, h int) *TerrainOcclusion {
	return &TerrainOcclusion{Width: w, Height: h, depth: make([]float32, w*h)}
}
func (o *TerrainOcclusion) Reset(view, projection M4) {
	o.view, o.projection, o.vp = view, projection, view.Mul(projection)
	clear(o.depth)
	o.covered = false
	o.Culled = 0
}

type occlusionPoint struct{ x, y, z float32 }

func (o *TerrainOcclusion) project(p V3) (occlusionPoint, bool) {
	m := o.vp
	w := p[0]*m[3] + p[1]*m[7] + p[2]*m[11] + m[15]
	if w <= .1 || math.IsNaN(float64(w)) || math.IsInf(float64(w), 0) {
		return occlusionPoint{}, false
	}
	x := (p[0]*m[0] + p[1]*m[4] + p[2]*m[8] + m[12]) / w
	y := (p[0]*m[1] + p[1]*m[5] + p[2]*m[9] + m[13]) / w
	if math.IsNaN(float64(x)) || math.IsNaN(float64(y)) || math.IsInf(float64(x), 0) || math.IsInf(float64(y), 0) {
		return occlusionPoint{}, false
	}
	return occlusionPoint{(x + 1) * .5 * float32(o.Width), (1 - y) * .5 * float32(o.Height), 1 / w}, true
}
func cross2(a, b, c occlusionPoint) float32     { return (b.x-a.x)*(c.y-a.y) - (b.y-a.y)*(c.x-a.x) }
func (o *TerrainOcclusion) Triangle(a, b, c V3) { o.polygon([]V3{a, b, c}) }

// FlatQuad avoids coverage cracks along the diagonal between coplanar triangles.
func (o *TerrainOcclusion) PlanarQuad(a, b, c, d V3) { o.polygon([]V3{a, b, c, d}) }

func (o *TerrainOcclusion) FlatQuad(lo, hi V3, y float32) {
	o.polygon([]V3{{lo[0], y, lo[2]}, {hi[0], y, lo[2]}, {hi[0], y, hi[2]}, {lo[0], y, hi[2]}})
}
func (o *TerrainOcclusion) polygon(vertices []V3) {
	var p [4]occlusionPoint
	n := len(vertices)
	for i, v := range vertices {
		q, ok := o.project(v)
		if !ok {
			return
		}
		p[i] = q
	}
	area := cross2(p[0], p[1], p[2])
	if area < 0 {
		for i := 0; i < n/2; i++ {
			p[i], p[n-1-i] = p[n-1-i], p[i]
		}
		area = cross2(p[0], p[1], p[2])
	}
	if area < .0001 {
		return
	}
	loX, loY := p[0].x, p[0].y
	hiX, hiY := loX, loY
	var edges [4][3]float32
	for i := 0; i < n; i++ {
		a, b := p[i], p[(i+1)%n]
		dx, dy := b.x-a.x, b.y-a.y
		edges[i] = [3]float32{-dy, dx, dy*a.x - dx*a.y + min(-dy, 0) + min(dx, 0)}
		loX = min(loX, a.x)
		hiX = max(hiX, a.x)
		loY = min(loY, a.y)
		hiY = max(hiY, a.y)
	}
	// Reciprocal W varies affinely across a planar projected polygon.
	dzdx := ((p[1].z-p[0].z)*(p[2].y-p[0].y) - (p[2].z-p[0].z)*(p[1].y-p[0].y)) / area
	dzdy := ((p[1].x-p[0].x)*(p[2].z-p[0].z) - (p[2].x-p[0].x)*(p[1].z-p[0].z)) / area
	base := p[0].z - dzdx*p[0].x - dzdy*p[0].y + min(dzdx, 0) + min(dzdy, 0)
	x0, x1 := max(0, int(math.Floor(float64(loX)))), min(o.Width-1, int(math.Ceil(float64(hiX))))
	y0, y1 := max(0, int(math.Floor(float64(loY)))), min(o.Height-1, int(math.Ceil(float64(hiY))))
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			inside := true
			for i := 0; i < n; i++ {
				e := edges[i]
				if e[0]*float32(x)+e[1]*float32(y)+e[2] < .0001 {
					inside = false
					break
				}
			}
			if inside {
				far := base + dzdx*float32(x) + dzdy*float32(y)
				if far > 0 {
					o.depth[y*o.Width+x] = max(o.depth[y*o.Width+x], far)
					o.covered = true
				}
			}
		}
	}
}
func (o *TerrainOcclusion) Box(lo, hi V3) bool {
	if o == nil || !o.covered {
		return false
	}
	minX, minY := float32(math.MaxFloat32), float32(math.MaxFloat32)
	maxX, maxY, near := float32(-math.MaxFloat32), float32(-math.MaxFloat32), float32(0)
	for i := 0; i < 8; i++ {
		p := lo
		for axis := 0; axis < 3; axis++ {
			if i&(1<<axis) != 0 {
				p[axis] = hi[axis]
			}
		}
		q, ok := o.project(p)
		if !ok {
			return false
		}
		minX = min(minX, q.x)
		maxX = max(maxX, q.x)
		minY = min(minY, q.y)
		maxY = max(maxY, q.y)
		near = max(near, q.z)
	}
	return o.rectangle(minX, minY, maxX, maxY, near)
}
func (o *TerrainOcclusion) rectangle(minX, minY, maxX, maxY, near float32) bool {
	// A one-cell guard includes projection/animation and rasterization margins.
	x0, y0 := int(math.Floor(float64(minX)))-1, int(math.Floor(float64(minY)))-1
	x1, y1 := int(math.Ceil(float64(maxX)))+1, int(math.Ceil(float64(maxY)))+1
	if x0 < 0 || y0 < 0 || x1 >= o.Width || y1 >= o.Height {
		return false
	}
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			if o.depth[y*o.Width+x] <= near*1.0001+.000001 {
				return false
			}
		}
	}
	o.Culled++
	return true
}
func (s *Scene3D) OccludedBox(lo, hi V3) bool {
	o := s.Occlusion
	if o == nil || s.Shadow || s.ClipPlane != [4]float32{} || s.View != o.view || s.VisibilityProjection() != o.projection {
		return false
	}
	return o.Box(lo, hi)
}
func (s *Scene3D) OccludedSphere(c V3, r float32) bool {
	if math.IsNaN(float64(r)) || math.IsInf(float64(r), 0) || r < 0 {
		return false
	}
	d := V3{r, r, r}
	return s.OccludedBox(c.Sub(d), c.Add(d))
}
func (s *Scene3D) meshOccluded(m *Mesh, world M4) bool {
	if s.Occlusion == nil || m == nil || !m.boundsValid {
		return false
	}
	lo, hi := V3{math.MaxFloat32, math.MaxFloat32, math.MaxFloat32}, V3{-math.MaxFloat32, -math.MaxFloat32, -math.MaxFloat32}
	for i := 0; i < 8; i++ {
		p := m.minimum
		for axis := 0; axis < 3; axis++ {
			if i&(1<<axis) != 0 {
				p[axis] = m.maximum[axis]
			}
		}
		for axis := 0; axis < 3; axis++ {
			v := p[0]*world[axis] + p[1]*world[4+axis] + p[2]*world[8+axis] + world[12+axis]
			lo[axis] = min(lo[axis], v)
			hi[axis] = max(hi[axis], v)
		}
	}
	return s.OccludedBox(lo, hi)
}
func (s *Scene3D) visibleEffectInstances(m *Mesh, worlds []M4, colors [][4]float32) ([]M4, [][4]float32) {
	if s.Occlusion == nil {
		return worlds, colors
	}
	first := -1
	for i, w := range worlds {
		if s.meshOccluded(m, w) {
			first = i
			break
		}
	}
	if first < 0 {
		return worlds, colors
	}
	s.occlusionWorlds = append(s.occlusionWorlds[:0], worlds[:first]...)
	s.occlusionColors = append(s.occlusionColors[:0], colors[:first]...)
	for i := first + 1; i < len(worlds); i++ {
		if !s.meshOccluded(m, worlds[i]) {
			s.occlusionWorlds = append(s.occlusionWorlds, worlds[i])
			s.occlusionColors = append(s.occlusionColors, colors[i])
		}
	}
	return s.occlusionWorlds, s.occlusionColors
}

// OccludedLabel tests the entire screen-space label, including outline padding.
// Its depth is constant just as in the name sprite shader.
func (s *Scene3D) OccludedLabel(anchor V3, x, y, width, height, viewportW, viewportH int) bool {
	o := s.Occlusion
	if o == nil || !o.covered || viewportW <= 0 || viewportH <= 0 || s.Shadow || s.ClipPlane != [4]float32{} || s.View != o.view || s.VisibilityProjection() != o.projection {
		return false
	}
	p, ok := o.project(anchor)
	if !ok {
		return false
	}
	sx, sy := float32(o.Width)/float32(viewportW), float32(o.Height)/float32(viewportH)
	// Include the Unicode raster texture's generous overhang area, not just
	// advances: combining marks and italic/script glyphs can extend beyond it.
	pad := height*3 + 3
	return o.rectangle(float32(x-pad)*sx, float32(y-pad)*sy, float32(x+width+pad)*sx, float32(y+height+pad)*sy, p.z)
}

// Matches permits reuse only for an identical current camera, never stale visibility.
func (o *TerrainOcclusion) Matches(view, projection M4) bool {
	return o != nil && o.view == view && o.projection == projection
}
