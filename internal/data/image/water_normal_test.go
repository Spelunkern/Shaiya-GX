package image

import "testing"

func TestWaterNormalsWrapHeightSamples(t *testing.T) {
	height := &RGBA{W: 3, H: 1, Pix: []byte{0, 0, 0, 255, 128, 0, 0, 255, 255, 0, 0, 255}}
	n := WaterNormals(height)
	if n == nil || n.Pix[0] <= 128 || n.Pix[4] >= 128 || n.Pix[8] <= 128 {
		t.Fatalf("wrapped horizontal slopes are wrong: %v", n)
	}
	for i := 0; i < len(n.Pix); i += 4 {
		if n.Pix[i+1] != 128 || n.Pix[i+3] != 255 {
			t.Fatalf("flat vertical gradient or alpha: %v", n.Pix[i:i+4])
		}
	}
	if WaterNormals(&RGBA{W: 1, H: 1}) != nil {
		t.Fatal("truncated height map accepted")
	}
}
