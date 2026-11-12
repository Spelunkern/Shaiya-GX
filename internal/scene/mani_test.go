package scene

import (
	"math"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

func TestManiRotatesLocallyAroundFixedPosition(t *testing.T) {
	base := render.Basis(render.V3{3, 4, 5}, render.V3{0, 0, 1}, render.V3{0, 1, 0})
	r := maniRotor{base: base, axis: render.V3{0, 2, 0}, speed: float32(math.Pi / 60)}
	m := r.transform(1)
	if math.Abs(float64(m[0])) > 1e-6 || math.Abs(float64(m[2]+1)) > 1e-6 || m[12] != 3 || m[13] != 4 || m[14] != 5 {
		t.Fatalf("wrong rotation or pivot: %v", m)
	}
	if r.transform(0) != base {
		t.Fatal("initial transform changed")
	}
}
