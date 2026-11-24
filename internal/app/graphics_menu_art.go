package app

import img "github.com/Spelunkern/Shaiya-GX/internal/data/image"

// prepareGraphicsMenuArt gives compact EP5 artwork the stock GX layout. The
// expanded pack already contains the reserved checkbox rows and footer.
func prepareGraphicsMenuArt(t img.Texture) {
	var width, height int
	var levels [][]byte
	switch {
	case t.RGBA != nil:
		width, height = t.RGBA.W, t.RGBA.H
		levels = [][]byte{t.RGBA.Pix}
	case t.DDS != nil && t.DDS.Format == img.FormatRGBA:
		width, height = t.DDS.W, t.DDS.H
		levels = t.DDS.Levels
	default:
		return
	}
	if width < 308 || height < 540 || len(levels) == 0 {
		return
	}
	// The left frame is opaque at row 539 in both versions and at row 614
	// only in the extended one, including when the compact PNG has padding.
	opaque := func(y int) bool { return levels[0][4*(y*width+16)+3] != 0 }
	if !opaque(539) {
		return
	}
	compact := height < 686 || !opaque(614)
	const addedRows = 146
	targetHeight := height
	if compact && height < 686 {
		targetHeight += addedRows
	}

	// Keep every authored PNG mip. Apply the same layout in each level's
	// coordinates rather than re-generating the rest of the atlas.
	for level, pixels := range levels {
		w, h := max(width>>level, 1), max(height>>level, 1)
		if compact {
			pixels = expandGraphicsMenuArt(pixels, w, h, max(targetHeight>>level, 1), level)
		}
		// Clear Color Bit or the baked AA caption for the runtime AA label.
		left, right, sample := 150>>level, 206>>level, 141>>level
		for y := 84 >> level; y < 101>>level && sample < left; y++ {
			row := pixels[y*w*4 : (y+1)*w*4]
			for x := left; x < right; x++ {
				copy(row[x*4:(x+1)*4], row[sample*4:(sample+1)*4])
			}
		}
		levels[level] = pixels
	}
	if t.RGBA != nil {
		t.RGBA.H, t.RGBA.Pix = targetHeight, levels[0]
	} else {
		t.DDS.H = targetHeight
	}
}

func expandGraphicsMenuArt(pixels []byte, w, h, targetHeight, level int) []byte {
	const split, addedRows = 469, 146
	stride := w * 4
	start, inserted := split>>level, addedRows>>level
	if targetHeight != h {
		expanded := make([]byte, stride*targetHeight)
		copy(expanded, pixels)
		pixels = expanded
	}
	if inserted == 0 {
		return pixels
	}
	footer := start + inserted
	copy(pixels[footer*stride:], pixels[start*stride:])
	tileRows := max(1, 4>>level)
	for row := range inserted {
		tile := row % (2 * tileRows)
		tile = min(tile, 2*tileRows-tile-1)
		copy(pixels[(start+row)*stride:], pixels[(footer+tile)*stride:(footer+tile+1)*stride])
	}

	// Clone the 22x20 Shadow frame into the slot beside Effects and six rows.
	boxWidth, boxHeight := 22>>level, 20>>level
	stamp := func(x, y int) {
		src := (401>>level)*stride + (32>>level)*4
		dst := (y>>level)*stride + (x>>level)*4
		for row := range boxHeight {
			copy(pixels[dst+row*stride:], pixels[src+row*stride:src+row*stride+boxWidth*4])
		}
	}
	stamp(153, 448)
	for y := 472; y <= 592; y += 24 {
		stamp(32, y)
		stamp(153, y)
	}
	return pixels
}
