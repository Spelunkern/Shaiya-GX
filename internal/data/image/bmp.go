package image

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// DecodeBMP handles BI_RGB 8 (palette), 24 and 32-bit images.
func DecodeBMP(b []byte) (*RGBA, error) {
	if len(b) < 54 || string(b[:2]) != "BM" {
		return nil, fmt.Errorf("%w: not bmp", ErrFormat)
	}
	off := int(binary.LittleEndian.Uint32(b[10:]))
	hdr := int(binary.LittleEndian.Uint32(b[14:]))
	w := int(int32(binary.LittleEndian.Uint32(b[18:])))
	h := int(int32(binary.LittleEndian.Uint32(b[22:])))
	bpp := int(binary.LittleEndian.Uint16(b[28:]))
	comp := binary.LittleEndian.Uint32(b[30:])
	topFirst := h < 0
	if topFirst {
		h = -h
	}
	if comp != 0 || w <= 0 || h <= 0 || w > maxSide || h > maxSide || (bpp != 8 && bpp != 24 && bpp != 32) {
		return nil, fmt.Errorf("%w: bmp %dx%d bpp %d comp %d", ErrFormat, w, h, bpp, comp)
	}
	var pal []byte
	if bpp == 8 {
		pal = b[14+hdr : min(off, len(b))]
	}
	stride := (w*bpp/8 + 3) &^ 3
	if off < 0 || off+stride*h > len(b) {
		return nil, errors.New("image: truncated bmp")
	}
	img := &RGBA{W: w, H: h, Pix: make([]byte, w*h*4)}
	for y := 0; y < h; y++ {
		row := b[off+y*stride:]
		dy := h - 1 - y
		if topFirst {
			dy = y
		}
		for x := 0; x < w; x++ {
			d := img.Pix[(dy*w+x)*4:]
			switch bpp {
			case 8:
				i := int(row[x]) * 4
				if i+2 < len(pal) {
					d[0], d[1], d[2] = pal[i+2], pal[i+1], pal[i]
				}
				d[3] = 255
			case 24:
				d[0], d[1], d[2], d[3] = row[x*3+2], row[x*3+1], row[x*3], 255
			case 32:
				d[0], d[1], d[2], d[3] = row[x*4+2], row[x*4+1], row[x*4], 255
			}
		}
	}
	return img, nil
}

// Texture is a decoded texture: either RGBA pixels or a DDS chain.
type Texture struct {
	RGBA *RGBA
	DDS  *DDS
}

// Decode picks the decoder by content, not by file extension: the archive
// has BMP and TGA data under .dds names.
func Decode(b []byte) (Texture, error) {
	switch {
	case len(b) >= 4 && string(b[:4]) == "DDS ":
		d, err := DecodeDDS(b)
		return Texture{DDS: d}, err
	case len(b) >= 2 && b[0] == 0xff && b[1] == 0xd8:
		i, err := DecodeJPEG(b)
		return Texture{RGBA: i}, err
	case len(b) >= 2 && string(b[:2]) == "BM":
		i, err := DecodeBMP(b)
		return Texture{RGBA: i}, err
	default:
		i, err := DecodeTGA(b)
		return Texture{RGBA: i}, err
	}
}
