package render

import "math"

// Frustum planes use the native LH clip range: -w<=x,y<=w and 0<=z<=w.
type Frustum [6][4]float32

func ViewFrustum(view, projection M4) Frustum {
	m := view.Mul(projection)
	f := Frustum{
		{m[3] + m[0], m[7] + m[4], m[11] + m[8], m[15] + m[12]},
		{m[3] - m[0], m[7] - m[4], m[11] - m[8], m[15] - m[12]},
		{m[3] + m[1], m[7] + m[5], m[11] + m[9], m[15] + m[13]},
		{m[3] - m[1], m[7] - m[5], m[11] - m[9], m[15] - m[13]},
		{m[2], m[6], m[10], m[14]},
		{m[3] - m[2], m[7] - m[6], m[11] - m[10], m[15] - m[14]},
	}
	for i, p := range f {
		length := float32(math.Sqrt(float64(p[0]*p[0] + p[1]*p[1] + p[2]*p[2])))
		if length > 0 {
			for j := range p {
				f[i][j] /= length
			}
		}
	}
	return f
}
func (f Frustum) SphereVisible(center V3, radius float32) bool {
	for _, p := range f {
		if p[0]*center[0]+p[1]*center[1]+p[2]*center[2]+p[3] < -radius {
			return false
		}
	}
	return true
}

// BoxVisible conservatively tests an axis-aligned box, including flat boxes.
// Reject it only when its most positive corner is outside a frustum plane.
func (f Frustum) BoxVisible(minimum, maximum V3) bool {
	for _, p := range f {
		corner := minimum
		for axis := 0; axis < 3; axis++ {
			if p[axis] >= 0 {
				corner[axis] = maximum[axis]
			}
		}
		if p[0]*corner[0]+p[1]*corner[1]+p[2]*corner[2]+p[3] < 0 {
			return false
		}
	}
	return true
}

// ClipBoxVisible rejects a box only if it is wholly behind a world clip plane.
// A zero plane means clipping is disabled.
func ClipBoxVisible(plane [4]float32, minimum, maximum V3) bool {
	if plane == [4]float32{} {
		return true
	}
	corner := minimum
	for axis := 0; axis < 3; axis++ {
		if plane[axis] >= 0 {
			corner[axis] = maximum[axis]
		}
	}
	return !(plane[0]*corner[0]+plane[1]*corner[1]+plane[2]*corner[2]+plane[3] < 0)
}

// ClipSphereVisible accepts touching spheres, including an unbounded radius.
func ClipSphereVisible(plane [4]float32, center V3, radius float32) bool {
	if plane == [4]float32{} {
		return true
	}
	norm := float32(math.Sqrt(float64(plane[0]*plane[0] + plane[1]*plane[1] + plane[2]*plane[2])))
	return !(plane[0]*center[0]+plane[1]*center[1]+plane[2]*center[2]+plane[3] < -radius*norm)
}
