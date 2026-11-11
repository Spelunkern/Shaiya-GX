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

// Inverse returns the inverse of a general 4x4 matrix (identity when singular).
func (m M4) Inverse() M4 {
	var inv M4
	a := m
	inv[0] = a[5]*a[10]*a[15] - a[5]*a[11]*a[14] - a[9]*a[6]*a[15] + a[9]*a[7]*a[14] + a[13]*a[6]*a[11] - a[13]*a[7]*a[10]
	inv[4] = -a[4]*a[10]*a[15] + a[4]*a[11]*a[14] + a[8]*a[6]*a[15] - a[8]*a[7]*a[14] - a[12]*a[6]*a[11] + a[12]*a[7]*a[10]
	inv[8] = a[4]*a[9]*a[15] - a[4]*a[11]*a[13] - a[8]*a[5]*a[15] + a[8]*a[7]*a[13] + a[12]*a[5]*a[11] - a[12]*a[7]*a[9]
	inv[12] = -a[4]*a[9]*a[14] + a[4]*a[10]*a[13] + a[8]*a[5]*a[14] - a[8]*a[6]*a[13] - a[12]*a[5]*a[10] + a[12]*a[6]*a[9]
	inv[1] = -a[1]*a[10]*a[15] + a[1]*a[11]*a[14] + a[9]*a[2]*a[15] - a[9]*a[3]*a[14] - a[13]*a[2]*a[11] + a[13]*a[3]*a[10]
	inv[5] = a[0]*a[10]*a[15] - a[0]*a[11]*a[14] - a[8]*a[2]*a[15] + a[8]*a[3]*a[14] + a[12]*a[2]*a[11] - a[12]*a[3]*a[10]
	inv[9] = -a[0]*a[9]*a[15] + a[0]*a[11]*a[13] + a[8]*a[1]*a[15] - a[8]*a[3]*a[13] - a[12]*a[1]*a[11] + a[12]*a[3]*a[9]
	inv[13] = a[0]*a[9]*a[14] - a[0]*a[10]*a[13] - a[8]*a[1]*a[14] + a[8]*a[2]*a[13] + a[12]*a[1]*a[10] - a[12]*a[2]*a[9]
	inv[2] = a[1]*a[6]*a[15] - a[1]*a[7]*a[14] - a[5]*a[2]*a[15] + a[5]*a[3]*a[14] + a[13]*a[2]*a[7] - a[13]*a[3]*a[6]
	inv[6] = -a[0]*a[6]*a[15] + a[0]*a[7]*a[14] + a[4]*a[2]*a[15] - a[4]*a[3]*a[14] - a[12]*a[2]*a[7] + a[12]*a[3]*a[6]
	inv[10] = a[0]*a[5]*a[15] - a[0]*a[7]*a[13] - a[4]*a[1]*a[15] + a[4]*a[3]*a[13] + a[12]*a[1]*a[7] - a[12]*a[3]*a[5]
	inv[14] = -a[0]*a[5]*a[14] + a[0]*a[6]*a[13] + a[4]*a[1]*a[14] - a[4]*a[2]*a[13] - a[12]*a[1]*a[6] + a[12]*a[2]*a[5]
	inv[3] = -a[1]*a[6]*a[11] + a[1]*a[7]*a[10] + a[5]*a[2]*a[11] - a[5]*a[3]*a[10] - a[9]*a[2]*a[7] + a[9]*a[3]*a[6]
	inv[7] = a[0]*a[6]*a[11] - a[0]*a[7]*a[10] - a[4]*a[2]*a[11] + a[4]*a[3]*a[10] + a[8]*a[2]*a[7] - a[8]*a[3]*a[6]
	inv[11] = -a[0]*a[5]*a[11] + a[0]*a[7]*a[9] + a[4]*a[1]*a[11] - a[4]*a[3]*a[9] - a[8]*a[1]*a[7] + a[8]*a[3]*a[5]
	inv[15] = a[0]*a[5]*a[10] - a[0]*a[6]*a[9] - a[4]*a[1]*a[10] + a[4]*a[2]*a[9] + a[8]*a[1]*a[6] - a[8]*a[2]*a[5]
	det := a[0]*inv[0] + a[1]*inv[4] + a[2]*inv[8] + a[3]*inv[12]
	if det == 0 {
		return Identity()
	}
	for i := range inv {
		inv[i] /= det
	}
	return inv
}

// QuatMatrix is the row-vector rotation of q (x,y,z,w) plus translation t
// (D3DXMatrixRotationQuaternion layout).
func QuatMatrix(q [4]float32, t [3]float32) M4 {
	x, y, z, w := q[0], q[1], q[2], q[3]
	return M4{
		1 - 2*(y*y+z*z), 2 * (x*y + z*w), 2 * (x*z - y*w), 0,
		2 * (x*y - z*w), 1 - 2*(x*x+z*z), 2 * (y*z + x*w), 0,
		2 * (x*z + y*w), 2 * (y*z - x*w), 1 - 2*(x*x+y*y), 0,
		t[0], t[1], t[2], 1,
	}
}

// MatrixQuat extracts the rotation quaternion and translation of a rigid
// row-vector matrix (inverse of QuatMatrix).
func MatrixQuat(m M4) (q [4]float32, t [3]float32) {
	t = [3]float32{m[12], m[13], m[14]}
	// Standard R is the transpose of the row-vector block.
	r00, r01, r02 := m[0], m[4], m[8]
	r10, r11, r12 := m[1], m[5], m[9]
	r20, r21, r22 := m[2], m[6], m[10]
	tr := r00 + r11 + r22
	var x, y, z, w float64
	switch {
	case tr > 0:
		s := math.Sqrt(float64(tr)+1) * 2
		w, x, y, z = s/4, float64(r21-r12)/s, float64(r02-r20)/s, float64(r10-r01)/s
	case r00 > r11 && r00 > r22:
		s := math.Sqrt(1+float64(r00-r11-r22)) * 2
		w, x, y, z = float64(r21-r12)/s, s/4, float64(r01+r10)/s, float64(r02+r20)/s
	case r11 > r22:
		s := math.Sqrt(1+float64(r11-r00-r22)) * 2
		w, x, y, z = float64(r02-r20)/s, float64(r01+r10)/s, s/4, float64(r12+r21)/s
	default:
		s := math.Sqrt(1+float64(r22-r00-r11)) * 2
		w, x, y, z = float64(r10-r01)/s, float64(r02+r20)/s, float64(r12+r21)/s, s/4
	}
	n := math.Sqrt(x*x + y*y + z*z + w*w)
	if n == 0 {
		return [4]float32{0, 0, 0, 1}, t
	}
	return [4]float32{float32(x / n), float32(y / n), float32(z / n), float32(w / n)}, t
}

// Slerp interpolates unit quaternions along the shortest arc.
func Slerp(a, b [4]float32, t float32) [4]float32 {
	d := a[0]*b[0] + a[1]*b[1] + a[2]*b[2] + a[3]*b[3]
	if d < 0 {
		d, b = -d, [4]float32{-b[0], -b[1], -b[2], -b[3]}
	}
	var s0, s1 float32
	if d > 0.9995 {
		s0, s1 = 1-t, t
	} else {
		th := math.Acos(float64(d))
		sn := math.Sin(th)
		s0 = float32(math.Sin((1-float64(t))*th) / sn)
		s1 = float32(math.Sin(float64(t)*th) / sn)
	}
	r := [4]float32{a[0]*s0 + b[0]*s1, a[1]*s0 + b[1]*s1, a[2]*s0 + b[2]*s1, a[3]*s0 + b[3]*s1}
	n := float32(math.Sqrt(float64(r[0]*r[0] + r[1]*r[1] + r[2]*r[2] + r[3]*r[3])))
	return [4]float32{r[0] / n, r[1] / n, r[2] / n, r[3] / n}
}
