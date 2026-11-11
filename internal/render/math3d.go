package render

import "math"

// V3 is a raw Shaiya/D3D vector (left-handed, Y up).
type V3 [3]float32

func (a V3) Add(b V3) V3        { return V3{a[0] + b[0], a[1] + b[1], a[2] + b[2]} }
func (a V3) Sub(b V3) V3        { return V3{a[0] - b[0], a[1] - b[1], a[2] - b[2]} }
func (a V3) Scale(s float32) V3 { return V3{a[0] * s, a[1] * s, a[2] * s} }
func (a V3) Dot(b V3) float32   { return a[0]*b[0] + a[1]*b[1] + a[2]*b[2] }
func (a V3) Cross(b V3) V3 {
	return V3{a[1]*b[2] - a[2]*b[1], a[2]*b[0] - a[0]*b[2], a[0]*b[1] - a[1]*b[0]}
}
func (a V3) Len() float32 { return float32(math.Sqrt(float64(a.Dot(a)))) }
func (a V3) Norm() V3 {
	l := a.Len()
	if l == 0 {
		return a
	}
	return a.Scale(1 / l)
}

// M4 is a D3D-style matrix (row vectors, v' = v*M) stored row-major.
// Uploaded untransposed, GLSL reads it as M^T, so `mat * vec4(v,1)` in a
// shader computes exactly the D3D product v*M.
type M4 [16]float32

func Identity() M4 { return M4{1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1} }

// Mul returns a*b in D3D order (apply a, then b).
func (a M4) Mul(b M4) M4 {
	var r M4
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			var s float32
			for k := 0; k < 4; k++ {
				s += a[i*4+k] * b[k*4+j]
			}
			r[i*4+j] = s
		}
	}
	return r
}

// LookAtLH is D3DXMatrixLookAtLH.
func LookAtLH(eye, at, up V3) M4 {
	z := at.Sub(eye).Norm()
	x := up.Cross(z).Norm()
	y := z.Cross(x)
	return M4{
		x[0], y[0], z[0], 0,
		x[1], y[1], z[1], 0,
		x[2], y[2], z[2], 0,
		-x.Dot(eye), -y.Dot(eye), -z.Dot(eye), 1,
	}
}

// PerspectiveFovLH is D3DXMatrixPerspectiveFovLH (depth 0..1; the GL
// context uses glClipControl ZERO_TO_ONE).
func PerspectiveFovLH(fovY, aspect, zn, zf float32) M4 {
	ys := float32(1 / math.Tan(float64(fovY)/2))
	xs := ys / aspect
	q := zf / (zf - zn)
	return M4{
		xs, 0, 0, 0,
		0, ys, 0, 0,
		0, 0, q, 1,
		0, 0, -zn * q, 0,
	}
}

// Basis builds a world matrix from the WLD forward/up vectors and position
// (rows: right, up, forward, position — left-handed).
func Basis(pos, fwd, up V3) M4 {
	f := fwd.Norm()
	u := up.Norm()
	r := u.Cross(f).Norm()
	return M4{r[0], r[1], r[2], 0, u[0], u[1], u[2], 0, f[0], f[1], f[2], 0, pos[0], pos[1], pos[2], 1}
}
