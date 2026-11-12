// Package image decodes the client's texture formats: TGA (raw and RLE,
// 8/24/32-bit truecolor), BMP, JPEG, and DDS (DXT1/3/5 and uncompressed RGB(A)).
// Compressed DDS data is kept as blocks for direct upload to the GPU.
package image

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// RGBA is a decoded image, top row first, 4 bytes per pixel.
type RGBA struct {
	W, H int
	Pix  []byte
}

const maxSide = 8192

var ErrFormat = errors.New("image: unsupported format")

// DecodeTGA handles types 2 (truecolor), 3 (grey), 10 and 11 (RLE).
func DecodeTGA(b []byte) (*RGBA, error) {
	if len(b) < 18 {
		return nil, ErrFormat
	}
	idLen, cmapType, typ := int(b[0]), b[1], b[2]
	w, h := int(binary.LittleEndian.Uint16(b[12:])), int(binary.LittleEndian.Uint16(b[14:]))
	bpp, desc := int(b[16]), b[17]
	if cmapType != 0 || w == 0 || h == 0 || w > maxSide || h > maxSide {
		return nil, fmt.Errorf("%w: tga cmap=%d %dx%d", ErrFormat, cmapType, w, h)
	}
	grey := typ == 3 || typ == 11
	rle := typ == 10 || typ == 11
	if !(typ == 2 || typ == 3 || rle) || (grey && bpp != 8) || (!grey && bpp != 24 && bpp != 32) {
		return nil, fmt.Errorf("%w: tga type %d bpp %d", ErrFormat, typ, bpp)
	}
	ps := bpp / 8
	if idLen > len(b)-18 {
		return nil, errors.New("image: truncated tga image ID")
	}
	src := b[18+idLen:]
	n := w * h
	raw := make([]byte, n*ps)
	if rle {
		o := 0
		for i := 0; o < len(raw); {
			if i >= len(src) {
				return nil, errors.New("image: truncated tga rle")
			}
			hdr := src[i]
			i++
			cnt := int(hdr&0x7f) + 1
			if hdr&0x80 != 0 {
				if i+ps > len(src) {
					return nil, errors.New("image: truncated tga rle")
				}
				for k := 0; k < cnt && o < len(raw); k++ {
					o += copy(raw[o:], src[i:i+ps])
				}
				i += ps
			} else {
				if i+cnt*ps > len(src) {
					return nil, errors.New("image: truncated tga rle")
				}
				o += copy(raw[o:], src[i:i+cnt*ps])
				i += cnt * ps
			}
		}
	} else {
		if len(src) < len(raw) {
			return nil, errors.New("image: truncated tga")
		}
		copy(raw, src)
	}
	img := &RGBA{W: w, H: h, Pix: make([]byte, n*4)}
	topFirst := desc&0x20 != 0
	rightFirst := desc&0x10 != 0
	for y := 0; y < h; y++ {
		dy := y
		if !topFirst {
			dy = h - 1 - y
		}
		for x := 0; x < w; x++ {
			dx := x
			if rightFirst {
				dx = w - 1 - x
			}
			s := raw[(y*w+x)*ps:]
			d := img.Pix[(dy*w+dx)*4:]
			if grey {
				d[0], d[1], d[2], d[3] = s[0], s[0], s[0], 255
				continue
			}
			d[0], d[1], d[2] = s[2], s[1], s[0]
			if ps == 4 {
				d[3] = s[3]
			} else {
				d[3] = 255
			}
		}
	}
	return img, nil
}
