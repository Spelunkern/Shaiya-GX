package scene

import (
	"math"

	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

type characterShadowFit struct {
	view, projection render.M4
	origin           render.V3
	x0, z0, x1, z1   float32
	valid            bool
}

// fitShadow is camera-independent; reflection and main passes use the same
// actor pose and world matrix. Preserve the exact per-vertex fit across them.
func (c *Character) fitShadow() characterShadowFit {
	if c == nil || c.clip == nil {
		return characterShadowFit{}
	}
	if c.shadowFitReady && c.shadowFitClip == c.clip && c.shadowFitTime == c.Time && c.shadowFitWorld == c.World {
		return c.shadowFit
	}
	c.preparePose()
	c.shadowFitClip, c.shadowFitTime, c.shadowFitWorld = c.clip, c.Time, c.World
	c.shadowFitReady = true
	c.shadowFit = characterShadowFit{}
	origin := render.V3{c.World[12], c.World[13], c.World[14]}
	view := render.LookAtLH(origin.Add(render.V3{0, 50, 20}), origin, render.V3{0, 1, 0})
	lo, hi := render.V3{math.MaxFloat32, math.MaxFloat32, math.MaxFloat32}, render.V3{-math.MaxFloat32, -math.MaxFloat32, -math.MaxFloat32}
	x0, z0, x1, z1 := float32(math.MaxFloat32), float32(math.MaxFloat32), float32(-math.MaxFloat32), float32(-math.MaxFloat32)
	count := 0
	for _, part := range c.parts {
		if cap(c.shadowScratch) < len(part.pose) {
			c.shadowScratch = make([]render.M4, len(part.pose))
		}
		matrices := c.shadowScratch[:len(part.pose)]
		for i, pose := range part.pose {
			matrices[i] = pose.Mul(c.World)
		}
		for _, v := range part.vertices {
			var point render.V3
			valid := true
			for i, weight := range [2]float32{v.Weight, 1 - v.Weight} {
				if weight == 0 {
					continue
				}
				bone := int(v.Bone0)
				if i == 1 {
					bone = int(v.Bone1)
				}
				if bone < 0 || bone >= len(part.pose) {
					valid = false
					break
				}
				m := matrices[bone]
				point = point.Add(transformPoint(v.Pos, m).Scale(weight))
			}
			if !valid {
				continue
			}
			p := transformPoint(point, view)
			for axis := range 3 {
				lo[axis] = min(lo[axis], p[axis])
				hi[axis] = max(hi[axis], p[axis])
			}
			z := point[2] - (point[1]-origin[1])*.4
			x0, x1 = min(x0, point[0]), max(x1, point[0])
			z0, z1 = min(z0, z), max(z1, z)
			count++
		}
	}
	if count == 0 {
		return c.shadowFit
	}
	// Border also covers attached weapons not present in the body skin bounds.
	pad := max(.5, (hi[0]-lo[0])*.25, (hi[1]-lo[1])*.25)
	projection := orthoBounds(lo[0]-pad, hi[0]+pad, lo[1]-pad, hi[1]+pad, max(.1, lo[2]-10), hi[2]+10)
	c.shadowFit = characterShadowFit{view: view, projection: projection, origin: origin, x0: x0 - pad, z0: z0 - pad, x1: x1 + pad, z1: z1 + pad, valid: true}
	return c.shadowFit
}

// DrawShadow follows the native eye offset (0,50,20), fits the posed silhouette,
// then projects it onto nearby terrain or dungeon geometry.
func (c *Character) DrawShadow(target *render.ShadowMap, k *render.Skinner, s *render.Scene3D, w *World) {
	if c == nil || w == nil || !w.HasShadowReceivers() {
		return
	}
	fit := c.fitShadow()
	if !fit.valid {
		return
	}
	capture := *s
	capture.View, capture.Proj, capture.Shadow = fit.view, fit.projection, true
	capture.ClipPlane = [4]float32{}
	capture.FogStart, capture.FogEnd = 0, 0
	target.Capture(func() { capture.Begin(); c.Draw(k, &capture) })
	s.Begin()
	if !w.HasTerrain() && c.shadowSeen == nil {
		c.shadowSeen = make(map[int]bool)
	}
	vertices, indices := c.shadowReceiver(w, fit)
	projector := fit.view.Mul(fit.projection)
	if len(indices) != 0 {
		target.Project(s, projector, vertices, indices, .45)
	}
	s.Begin()
}

// Only terrain and dungeon floor geometry receive the local character shadow.
func (c *Character) shadowReceiver(w *World, fit characterShadowFit) ([]render.Vertex, []uint32) {
	bounds := [5]float32{fit.x0, fit.z0, fit.x1, fit.z1, fit.origin[1]}
	if w.HasTerrain() {
		// Terrain receivers use only the aligned lattice; their geometry does not
		// depend on actor height. Small pose changes often stay within these cells.
		bounds = [5]float32{
			float32(math.Floor(float64(fit.x0)/2)) * 2,
			float32(math.Floor(float64(fit.z0)/2)) * 2,
			float32(math.Ceil(float64(fit.x1)/2)) * 2,
			float32(math.Ceil(float64(fit.z1)/2)) * 2,
			0,
		}
	}
	if !c.shadowReceiverReady || c.shadowReceiverWorld != w || c.shadowReceiverBounds != bounds {
		c.shadowVertices, c.shadowIndices = w.shadowReceiverInto(bounds[0], bounds[1], bounds[2], bounds[3], bounds[4], c.shadowVertices, c.shadowIndices, c.shadowSeen)
		c.shadowReceiverWorld, c.shadowReceiverBounds, c.shadowReceiverReady = w, bounds, true
	}
	return c.shadowVertices, c.shadowIndices
}

func (w *World) shadowReceiver(x0, z0, x1, z1, height float32) ([]render.Vertex, []uint32) {
	return w.shadowReceiverInto(x0, z0, x1, z1, height, nil, nil, nil)
}

// Each character can reuse field receiver buffers after Project uploads them.
func (w *World) shadowReceiverInto(x0, z0, x1, z1, height float32, vertices []render.Vertex, indices []uint32, seen map[int]bool) ([]render.Vertex, []uint32) {
	if !w.HasTerrain() {
		return w.dgShadow.queryInto(x0, z0, x1, z1, height, vertices, indices, seen)
	}
	// Align the receiver to the field lattice; equal diagonals avoid gaps on slopes.
	x0 = float32(math.Floor(float64(x0)/2)) * 2
	z0 = float32(math.Floor(float64(z0)/2)) * 2
	x1 = float32(math.Ceil(float64(x1)/2)) * 2
	z1 = float32(math.Ceil(float64(z1)/2)) * 2
	nx, nz := int((x1-x0)/2), int((z1-z0)/2)
	if nx <= 0 || nz <= 0 || nx > 128 || nz > 128 {
		return nil, nil
	}
	if cap(vertices) < (nx+1)*(nz+1) {
		vertices = make([]render.Vertex, 0, (nx+1)*(nz+1))
	} else {
		vertices = vertices[:0]
	}
	// Loaded field grids have a two-unit step. Inside the map every receiver
	// point lands exactly on a stored height, so interpolation is unnecessary.
	direct := w.step == 2 && x0 >= 0 && z0 >= 0 && x1 <= float32((w.side-1)*2) && z1 <= float32((w.side-1)*2) && len(w.heights) >= w.side*w.side
	baseX, baseZ := int(x0/2), int(z0/2)
	for z := 0; z <= nz; z++ {
		for x := 0; x <= nx; x++ {
			px, pz := x0+float32(x*2), z0+float32(z*2)
			var height float32
			if direct {
				height = w.heights[(baseZ+z)*w.side+baseX+x]
			} else {
				height = w.Height(px, pz)
			}
			vertices = append(vertices, render.Vertex{Pos: render.V3{px, height, pz}})
		}
	}
	if cap(indices) < nx*nz*6 {
		indices = make([]uint32, 0, nx*nz*6)
	} else {
		indices = indices[:0]
	}
	for z := 0; z < nz; z++ {
		for x := 0; x < nx; x++ {
			a := uint32(z*(nx+1) + x)
			b := a + 1
			cc := a + uint32(nx+1)
			d := cc + 1
			indices = append(indices, a, b, cc, b, d, cc)
		}
	}
	return vertices, indices
}

func transformPoint(v render.V3, m render.M4) render.V3 {
	return render.V3{v[0]*m[0] + v[1]*m[4] + v[2]*m[8] + m[12], v[0]*m[1] + v[1]*m[5] + v[2]*m[9] + m[13], v[0]*m[2] + v[1]*m[6] + v[2]*m[10] + m[14]}
}
func orthoBounds(left, right, bottom, top, near, far float32) render.M4 {
	return render.M4{2 / (right - left), 0, 0, 0, 0, 2 / (top - bottom), 0, 0, 0, 0, 1 / (far - near), 0, -(left + right) / (right - left), -(top + bottom) / (top - bottom), -near / (far - near), 1}
}
