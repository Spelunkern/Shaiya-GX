package ui

import (
	"math"
	"testing"
)

func TestMinimapProjectionAndClampedCrop(t *testing.T) {
	for _, corner := range []struct{ x, z, u, v float32 }{{0, 0, 0, 1}, {2048, 0, 1, 1}, {0, 2048, 0, 0}, {2048, 2048, 1, 0}} {
		u, v, ok := MapPosition(corner.x, corner.z, 2048)
		if !ok || u != corner.u || v != corner.v {
			t.Fatal("map axes do not match source calibration", corner, u, v)
		}
	}
	if _, _, ok := MapPosition(100, 100, 0); ok {
		t.Fatal("dungeon without calibration invented a position")
	}
	for _, pos := range [][2]float32{{0, 0}, {1, 1}, {.5, .5}} {
		view := MapWindow(pos[0], pos[1], .4)
		if view.U0 < 0 || view.V0 < 0 || view.U1 > 1 || view.V1 > 1 || math.Abs(float64(view.U1-view.U0-.4)) > .00001 {
			t.Fatal("minimap stretched or sampled beyond image edge", view)
		}
		x, y, ok := MapPoint(pos[0], pos[1], view)
		if !ok || x < 0 || x > 1 || y < 0 || y > 1 {
			t.Fatal("marker lost when clamping image", pos, view, x, y)
		}
	}
}

func TestFullMapFrameFitsWindowWithoutChangingImageAspect(t *testing.T) {
	for _, size := range [][2]int{{640, 480}, {800, 600}, {1024, 768}, {1920, 1080}} {
		x, y, scale := FullMapRect(size[0], size[1])
		if x < 0 || y < 0 || x+int(544*scale) > size[0] || y+int(579*scale) > size[1] || scale > 1 {
			t.Fatal("full map clipped or upscaled native artwork", size, x, y, scale)
		}
	}
}
