package image

import (
	"bytes"
	"encoding/binary"
	"fmt"
	stdimage "image"
	"image/png"
)

// Cursor holds native pixel dimensions and the click hotspot from a Windows
// .cur directory. AND masks supply transparency for legacy RGB/paletted images.
type Cursor struct {
	Image      *RGBA
	HotX, HotY int
}

func DecodeCUR(b []byte) (*Cursor, error) {
	bad := func() (*Cursor, error) { return nil, fmt.Errorf("%w: invalid cursor", ErrFormat) }
	if len(b) < 22 || binary.LittleEndian.Uint16(b) != 0 || binary.LittleEndian.Uint16(b[2:]) != 2 {
		return bad()
	}
	count := int(binary.LittleEndian.Uint16(b[4:]))
	if count < 1 || count > (len(b)-6)/16 {
		return bad()
	}
	entry := b[6:22]
	width, height := int(entry[0]), int(entry[1])
	if width == 0 {
		width = 256
	}
	if height == 0 {
		height = 256
	}
	hx, hy := int(binary.LittleEndian.Uint16(entry[4:])), int(binary.LittleEndian.Uint16(entry[6:]))
	size, off := uint64(binary.LittleEndian.Uint32(entry[8:])), uint64(binary.LittleEndian.Uint32(entry[12:]))
	if hx >= width || hy >= height || off < uint64(6+16*count) || size > uint64(len(b)) || off+size > uint64(len(b)) {
		return bad()
	}
	raw := b[off : off+size]
	var out *RGBA
	if bytes.HasPrefix(raw, []byte("\x89PNG\r\n\x1a\n")) {
		cfg, err := png.DecodeConfig(bytes.NewReader(raw))
		if err != nil || cfg.Width != width || cfg.Height != height {
			return bad()
		}
		src, err := png.Decode(bytes.NewReader(raw))
		if err != nil {
			return bad()
		}
		dst := stdimage.NewNRGBA(stdimage.Rect(0, 0, width, height))
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				dst.Set(x, y, src.At(x, y))
			}
		}
		out = &RGBA{W: width, H: height, Pix: dst.Pix}
	} else {
		if len(raw) < 40 {
			return bad()
		}
		hdr := int(binary.LittleEndian.Uint32(raw))
		w, h := int(int32(binary.LittleEndian.Uint32(raw[4:]))), int(int32(binary.LittleEndian.Uint32(raw[8:])))
		bits := int(binary.LittleEndian.Uint16(raw[14:]))
		compression := binary.LittleEndian.Uint32(raw[16:])
		if hdr < 40 || hdr > len(raw) || w != width || h != height*2 || compression != 0 || (bits != 1 && bits != 4 && bits != 8 && bits != 24 && bits != 32) {
			return bad()
		}
		colors := 0
		if bits <= 8 {
			colors = int(binary.LittleEndian.Uint32(raw[32:]))
			if colors == 0 {
				colors = 1 << bits
			}
			if colors > 1<<bits {
				return bad()
			}
		}
		pixelOffset := hdr + colors*4
		stride := ((width*bits + 31) / 32) * 4
		maskStride := ((width + 31) / 32) * 4
		maskOffset := pixelOffset + stride*height
		if pixelOffset > len(raw) || maskOffset+maskStride*height > len(raw) {
			return bad()
		}
		out = &RGBA{W: width, H: height, Pix: make([]byte, width*height*4)}
		alpha := false
		for y := 0; y < height; y++ {
			row := raw[pixelOffset+(height-1-y)*stride:]
			for x := 0; x < width; x++ {
				p := out.Pix[(y*width+x)*4:]
				if bits <= 8 {
					index := int(row[x*bits/8]>>(8-bits-(x*bits)%8)) & ((1 << bits) - 1)
					if index >= colors {
						return bad()
					}
					c := raw[hdr+index*4:]
					p[0], p[1], p[2], p[3] = c[2], c[1], c[0], 255
				} else {
					c := row[x*(bits/8):]
					p[0], p[1], p[2], p[3] = c[2], c[1], c[0], 255
					if bits == 32 {
						p[3] = c[3]
						alpha = alpha || c[3] != 0
					}
				}
			}
		}
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				p := out.Pix[(y*width+x)*4:]
				if bits == 32 && !alpha {
					p[3] = 255
				}
				if raw[maskOffset+(height-1-y)*maskStride+x/8]&(0x80>>uint(x%8)) != 0 {
					p[3] = 0
				}
			}
		}
	}
	return &Cursor{Image: out, HotX: hx, HotY: hy}, nil
}
