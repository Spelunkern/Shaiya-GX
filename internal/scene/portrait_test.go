package scene

import (
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"math"
	"testing"
)

func TestPortraitFrameFitsTallAndWideBodies(t *testing.T) {
	for _, bounds := range [][2]render.V3{{{-1, 0, -.2}, {1, 3, .2}}, {{-2, 0, -.4}, {2, 2, .4}}} {
		c := &Character{parts: []*skinPart{{min: bounds[0], max: bounds[1]}}}
		const aspect, fov, scale = float32(.65), float32(math.Pi / 5), float32(1.04)
		center, distance := c.PortraitFrame(aspect, fov, scale)
		tangent := float32(math.Tan(float64(fov) / 2))
		for _, p := range bounds {
			depth := distance - float32(math.Abs(float64(p[2])))
			sx := float32(math.Abs(float64(p[0]))) / (depth * tangent)
			sy := float32(math.Abs(float64(p[1]*scale-center))) / (depth * tangent)
			if depth <= 0 || sx > aspect || sy > 1 {
				t.Fatal("portrait clips body", bounds, center, distance, sx, sy)
			}
		}
	}
}

func TestInventoryFrameDoesNotZoomOutForBindPoseArms(t *testing.T) {
	narrow := &Character{parts: []*skinPart{{min: render.V3{-.3, 0, -.1}, max: render.V3{.3, 2, .1}}}}
	wide := &Character{parts: []*skinPart{{min: render.V3{-2, 0, -.1}, max: render.V3{2, 2, .1}}}}
	c1, d1 := narrow.InventoryFrame(math.Pi/5, 1)
	c2, d2 := wide.InventoryFrame(math.Pi/5, 1)
	if c1 != 1 || c1 != c2 || d1 != d2 {
		t.Fatal("bind-pose arms changed inventory scale", c1, d1, c2, d2)
	}
	_, tall := wide.InventoryFrame(math.Pi/5, 1.04)
	if tall <= d2 {
		t.Fatal("height did not preserve full-body framing")
	}
}
