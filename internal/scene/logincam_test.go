package scene

import (
	"math"
	"testing"
)

// Phoenix scripts/login_camera.gd at its current ENTRY_PROGRESS_RATE = 0.01.
func TestLoginCameraTrajectory(t *testing.T) {
	want := map[int][2]float64{60: {442.0887, 377.7878}, 240: {444.5252, 256.8980}, 300: {444.6504, 255.6047}, 600: {444.7572, 255.2666}, 1200: {444.7623, 255.1319}}
	c := NewLoginCamera()
	for f := 1; f <= 1200; f++ {
		c.step(1.0 / 60)
		if w, ok := want[f]; ok {
			t.Logf("frame %d: eye x %.4f z %.4f (want %.4f %.4f)", f, c.Eye[0], c.Eye[2], w[0], w[1])
			if math.Abs(float64(c.Eye[2])-w[1]) > 0.01 || math.Abs(float64(c.Eye[0])-w[0]) > 0.01 {
				t.Errorf("frame %d z %.2f, want %.2f", f, c.Eye[2], w[1])
			}
		}
	}
}
