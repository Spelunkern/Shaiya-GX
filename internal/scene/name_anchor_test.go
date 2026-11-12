package scene

import (
	"math"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

func TestNameAnchorFollowsRenderedHeadPoseAndWorldTransform(t *testing.T) {
	a, b, world := render.Identity(), render.Identity(), render.Identity()
	a[13], b[13], world[12], world[13], world[14] = 1, 3, 10, 4, 5
	head := &skinPart{min: render.V3{-1, 1, -.5}, max: render.V3{1, 2, .5},
		top: render.SkinVertex{Weight: .25, Bone0: 0, Bone1: 1}, pose: []render.M4{a, b}}
	c := &Character{head: head, World: world, parts: []*skinPart{{max: render.V3{0, 10, 0}}, head}}
	point, ok := c.NameAnchor()
	if !ok || point[0] != 10 || point[2] != 5 || math.Abs(float64(point[1]-8.65)) > .0001 {
		t.Fatal("plate ignored face/skin/world transform", point, ok)
	}
	head.pose = nil
	if _, ok := c.NameAnchor(); ok {
		t.Fatal("undrawn pose produced an arbitrary plate")
	}
}
