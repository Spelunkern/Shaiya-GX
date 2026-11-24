package app

import (
	"bytes"

	img "github.com/Spelunkern/Shaiya-GX/internal/data/image"
)

// prepareGraphicsMenuArt removes the artwork from the former spare controls
// and clears the Color Bit label for the AA control.
func prepareGraphicsMenuArt(art *img.RGBA) {
	const (
		split     = 469
		addedRows = 146
	)
	if art == nil || art.W != 512 || art.H < 540 {
		return
	}

	stride := art.W * 4
	row := func(y, x0, x1 int) []byte {
		return art.Pix[y*stride+x0*4 : y*stride+x1*4]
	}

	// The previous asset patch cloned the Shadow checkbox into the vacant
	// slot beside Effects and inserted 146 rows just below it.
	expanded := art.H >= 686 && row(401, 32, 54)[3] != 0
	for y := 0; expanded && y < 20; y++ {
		expanded = bytes.Equal(row(401+y, 32, 54), row(448+y, 153, 175))
	}
	if expanded {
		for y := 448; y < 468; y++ {
			copy(row(y, 153, 175), row(y, 175, 197))
		}
		copy(art.Pix[split*stride:], art.Pix[(split+addedRows)*stride:])
		clear(art.Pix[(art.H-addedRows)*stride:])
	}

	// Tile the panel background immediately left of Color Bit.
	for y := 84; y < 101; y++ {
		var tile [16 * 4]byte
		copy(tile[:], row(y, 141, 157))
		for x := 150; x < 206; x += 16 {
			copy(row(y, x, min(x+16, 206)), tile[:])
		}
	}
}
