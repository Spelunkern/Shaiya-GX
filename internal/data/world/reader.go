package world

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
)

// ErrTruncated reports a file shorter than its declared contents.
var ErrTruncated = errors.New("world: truncated data")

// reader is a little-endian cursor whose first error sticks.
type reader struct {
	b   []byte
	off int
	err error
}

func (r *reader) take(n int) []byte {
	if r.err != nil || n < 0 || r.off+n > len(r.b) {
		if r.err == nil {
			r.err = fmt.Errorf("%w at offset %d (+%d of %d)", ErrTruncated, r.off, n, len(r.b))
		}
		return nil
	}
	p := r.b[r.off : r.off+n]
	r.off += n
	return p
}

func (r *reader) u16() uint16 {
	if p := r.take(2); p != nil {
		return binary.LittleEndian.Uint16(p)
	}
	return 0
}
func (r *reader) u32() uint32 {
	if p := r.take(4); p != nil {
		return binary.LittleEndian.Uint32(p)
	}
	return 0
}
func (r *reader) i32() int32   { return int32(r.u32()) }
func (r *reader) f32() float32 { return math.Float32frombits(r.u32()) }
func (r *reader) vec3() Vec3   { return Vec3{r.f32(), r.f32(), r.f32()} }
func (r *reader) vec2() [2]float32 {
	return [2]float32{r.f32(), r.f32()}
}
func (r *reader) box() Box { return Box{r.vec3(), r.vec3()} }

// count reads an i32 element count bounded by the bytes left.
func (r *reader) count(minSize int) int {
	n := r.i32()
	if r.err != nil {
		return 0
	}
	if n < 0 || (minSize > 0 && int(n)*minSize > len(r.b)-r.off) {
		r.err = fmt.Errorf("world: bad count %d at offset %d", n, r.off-4)
		return 0
	}
	return int(n)
}

// fixed reads a NUL-padded string of n bytes.
func (r *reader) fixed(n int) string {
	p := r.take(n)
	for i, c := range p {
		if c == 0 {
			return string(p[:i])
		}
	}
	return string(p)
}

// lstring reads an i32-length string (terminator included).
func (r *reader) lstring() string {
	n := r.count(1)
	return r.fixedLen(n)
}

func (r *reader) fixedLen(n int) string { return r.fixed(n) }

// Vec3 and Box are the file coordinate types (Shaiya: Y up).
type Vec3 [3]float32

type Box struct{ Min, Max Vec3 }

// Face is three u16 vertex indices.
type Face [3]uint16

func (r *reader) faces() []Face {
	n := r.count(6)
	f := make([]Face, n)
	for i := range f {
		f[i] = Face{r.u16(), r.u16(), r.u16()}
	}
	return f
}
