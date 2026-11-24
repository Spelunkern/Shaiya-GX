package app

import (
	"bytes"
	"testing"

	img "github.com/Spelunkern/Shaiya-GX/internal/data/image"
)

func TestGraphicsMenuArtPreservesReservedCheckboxes(t *testing.T) {
	for _, tc := range []struct {
		name        string
		w, h, wantH int
		expanded    bool
	}{
		{"padded-compact", 512, 1024, 1024, false},
		{"unpadded-compact", 311, 548, 694, false},
		{"stock-expanded", 512, 1024, 1024, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			art := &img.RGBA{W: tc.w, H: tc.h, Pix: make([]byte, tc.w*tc.h*4)}
			pixel := func(x, y int) []byte { return art.Pix[(y*art.W+x)*4:][:4] }
			frame := []byte{13, 24, 35, 255}
			copy(pixel(32, 401), frame)
			pixel(16, 539)[3] = 255
			footer := 539
			if tc.expanded {
				pixel(16, 614)[3] = 255
				footer += 146
				// An authored expanded frame may style the spare boxes differently.
				pixel(153, 448)[0], pixel(32, 592)[0] = 71, 93
			}
			pixel(8, footer)[0] = 7
			pixel(141, 90)[0], pixel(150, 90)[0] = 9, 10
			prepareGraphicsMenuArt(img.Texture{RGBA: art})
			if art.H != tc.wantH || pixel(8, 685)[0] != 7 {
				t.Fatalf("height %d or footer pixel %v", art.H, pixel(8, 685))
			}
			if tc.expanded {
				if pixel(153, 448)[0] != 71 || pixel(32, 592)[0] != 93 {
					t.Fatal("authored checkbox pixels changed")
				}
			} else {
				positions := [][2]int{{153, 448}}
				for y := 472; y <= 592; y += 24 {
					positions = append(positions, [2]int{32, y}, [2]int{153, y})
				}
				for _, p := range positions {
					if !bytes.Equal(pixel(p[0], p[1]), frame) {
						t.Fatalf("reserved checkbox frame missing at %v", p)
					}
				}
			}
			if !bytes.Equal(pixel(32, 401), frame) || pixel(150, 90)[0] != 9 {
				t.Fatal("shadow frame or AA label background is wrong")
			}
		})
	}
}

func TestGraphicsMenuArtKeepsAuthoredMipLevels(t *testing.T) {
	art := &img.DDS{W: 311, H: 548, Format: img.FormatRGBA}
	for level := range 3 {
		w, h := art.W>>level, art.H>>level
		pixels := make([]byte, w*h*4)
		pixels[4*((539>>level)*w+(16>>level))+3] = 255
		pixels[4*((80>>level)*w+(64>>level))] = byte(level + 40)
		art.Levels = append(art.Levels, pixels)
	}
	prepareGraphicsMenuArt(img.Texture{DDS: art})
	if art.H != 694 || len(art.Levels) != 3 {
		t.Fatalf("dimensions or mip count changed incorrectly: %d, %d", art.H, len(art.Levels))
	}
	for level, pixels := range art.Levels {
		w, h := art.W>>level, art.H>>level
		if got := pixels[4*((80>>level)*w+(64>>level))]; len(pixels) != w*h*4 || got != byte(level+40) {
			t.Fatalf("mip %d lost its authored pixels or dimensions", level)
		}
	}
}
