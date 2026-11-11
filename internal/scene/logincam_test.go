package scene

import (
	"math"
	"testing"
)

// Reference trajectory sampled from the native routine (Phoenix
// ep5_login.json trajectory60Hz).
func TestLoginCameraTrajectory(t *testing.T) {
	want := map[int][2]float64{60: {441.8087, 415.2540}, 300: {443.5152, 334.7174}, 600: {444.6011, 265.8323}, 1200: {444.7440, 256.3700}}
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
