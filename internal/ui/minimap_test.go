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

func TestExpandedMapHoverCoordinatesRespectCropAndNorth(t *testing.T) {
	for _, view := range []UV{{U0: 0, V0: 0, U1: 1, V1: 1}, MapWindow(.72, .25, .25), MapWindow(0, 1, .5)} {
		for _, p := range [][2]float32{{0, 0}, {1, 1}, {.25, .8}, {.5, .5}} {
			x, z, ok := MapCoordinates(view, p[0], p[1], 2048)
			if !ok {
				t.Fatal("valid hover rejected")
			}
			u, v, _ := MapPosition(x, z, 2048)
			fx, fy, ok := MapPoint(u, v, view)
			if !ok || math.Abs(float64(fx-p[0])) > .00001 || math.Abs(float64(fy-p[1])) > .00001 {
				t.Fatal("hover axes/crop mismatch", view, p, x, z, fx, fy)
			}
		}
	}
	if _, _, ok := MapCoordinates(MapWindow(.5, .5, 1), -.01, .5, 2048); ok {
		t.Fatal("outside image accepted")
	}
}
