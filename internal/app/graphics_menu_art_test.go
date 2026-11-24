package app

import (
	"bytes"
	"testing"

	img "github.com/Spelunkern/Shaiya-GX/internal/data/image"
)

func TestGraphicsMenuArtCompactsLegacyFrame(t *testing.T) {
	const w, h = 512, 1024
	for _, expanded := range []bool{false, true} {
		art := &img.RGBA{W: w, H: h, Pix: make([]byte, w*h*4)}
		pixels := func(y, left, right int) []byte {
			return art.Pix[(y*w+left)*4 : (y*w+right)*4]
		}
		for y := 401; y < 421; y++ {
			copy(pixels(y, 32, 33), []byte{1, 2, 3, 255})
		}
		footer := 539
		if expanded {
			footer += 146
			for y := 0; y < 20; y++ {
				copy(pixels(448+y, 153, 175), pixels(401+y, 32, 54))
			}
			pixels(475, 35, 36)[0] = 5
		}
		pixels(footer, 0, 1)[0] = 7
		pixels(90, 141, 142)[0] = 9
		pixels(90, 150, 151)[0] = 10
		prepareGraphicsMenuArt(art)
		if pixels(539, 0, 1)[0] != 7 || pixels(685, 0, 1)[0] != 0 {
			t.Fatalf("expanded=%t: footer was not restored", expanded)
		}
		if !bytes.Equal(pixels(448, 153, 175), make([]byte, 22*4)) || pixels(475, 35, 36)[0] != 0 {
			t.Fatalf("expanded=%t: spare checkbox pixels remain", expanded)
		}
		if pixels(401, 32, 33)[0] != 1 || pixels(90, 150, 151)[0] != 9 {
			t.Fatalf("expanded=%t: shadow box or antialias label slot is wrong", expanded)
		}
	}
}
