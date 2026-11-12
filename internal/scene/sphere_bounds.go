package scene

import (
	"math"

	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

// sphereScale bounds the largest stretch of the 3x3 transform using the
// maximum absolute row sum of its Gram matrix. It is exactly one for an
// orthonormal placement and conservative for skewed/scaled placements.
func sphereScale(m render.M4) float32 {
	rows := [3]render.V3{{m[0], m[1], m[2]}, {m[4], m[5], m[6]}, {m[8], m[9], m[10]}}
	var bound float32
	for i := range 3 {
		var sum float32
		for j := range 3 {
			sum += float32(math.Abs(float64(rows[i].Dot(rows[j]))))
		}
		bound = max(bound, sum)
	}
	return float32(math.Sqrt(float64(bound)))
}
