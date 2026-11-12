package world

import (
	"fmt"
	"math"
)

// Mani describes the continuous local rotation of a WLD mesh instance.
type Mani struct {
	Rotate bool
	Axis   Vec3
	Speed  float32 // radians per 30 Hz animation tick
}

func ParseMani(b []byte) (Mani, error) {
	r := &reader{b: b}
	version := r.i32()
	r.take(56) // original fields not needed by rotation
	enabled := r.i32()
	axis, speed := r.vec3(), r.f32()
	r.take(28)
	if r.err != nil {
		return Mani{}, r.err
	}
	if version != 33 {
		return Mani{}, fmt.Errorf("world: unknown mani version %d", version)
	}
	for _, v := range []float32{axis[0], axis[1], axis[2], speed} {
		if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
			return Mani{}, fmt.Errorf("world: invalid mani rotation")
		}
	}
	return Mani{Rotate: enabled != 0, Axis: axis, Speed: speed}, nil
}
