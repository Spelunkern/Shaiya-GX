package scene

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

func TestSphereScaleIncludesSkewedTransforms(t *testing.T) {
	m := render.Identity()
	if sphereScale(m) != 1 {
		t.Fatal("identity expanded")
	}
	m[4] = 1
	bound := sphereScale(m)
	for _, v := range []render.V3{{1, 0, 0}, {0, 1, 0}, {.70710677, .70710677, 0}, {-.70710677, .70710677, 0}} {
		if got := transformPoint(v, m).Len(); got > bound+1e-6 {
			t.Fatalf("transformed point %v exceeds sphere %v", got, bound)
		}
	}
}
