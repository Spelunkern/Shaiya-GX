package image

import "math"

// WaterNormals derives a wrapped normal map from the red height channel, as
// the original water setup does for WaterBumps1/2. The input is not modified.
func WaterNormals(height *RGBA) *RGBA {
	if height == nil || height.W <= 0 || height.H <= 0 || len(height.Pix) < height.W*height.H*4 {
		return nil
	}
	out := &RGBA{W: height.W, H: height.H, Pix: make([]byte, height.W*height.H*4)}
	red := func(x, y int) float64 { return float64(height.Pix[(y*height.W+x)*4]) }
	for y := 0; y < height.H; y++ {
		for x := 0; x < height.W; x++ {
			left, right := (x+height.W-1)%height.W, (x+1)%height.W
			up, down := (y+height.H-1)%height.H, (y+1)%height.H
			nx := (red(left, y) - red(right, y)) * 10 / 255
			ny := (red(x, up) - red(x, down)) * 10 / 255
			inv := 1 / math.Sqrt(nx*nx+ny*ny+1)
			p := out.Pix[(y*height.W+x)*4:]
			p[0], p[1], p[2], p[3] = byte((nx*inv*.5+.5)*255+.5), byte((ny*inv*.5+.5)*255+.5), byte((inv*.5+.5)*255+.5), 255
		}
	}
	return out
}
