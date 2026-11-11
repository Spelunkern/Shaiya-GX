package image

import (
	"encoding/binary"
	"errors"
	"fmt"
)

type Format int

const (
	FormatRGBA Format = iota // uncompressed, decoded to RGBA8
	FormatDXT1
	FormatDXT3
	FormatDXT5
)

// DDS holds every declared mip level. For DXT formats Levels are the raw
// blocks; for FormatRGBA they are RGBA8 pixels.
type DDS struct {
	W, H   int
	Format Format
	Levels [][]byte
}

func blockBytes(f Format) int {
	if f == FormatDXT1 {
		return 8
	}
	return 16
}

// DecodeDDS parses the 128-byte header (DDSD/DDPF fields at the standard
// offsets) and slices the mip chain.
func DecodeDDS(b []byte) (*DDS, error) {
	if len(b) < 128 || string(b[:4]) != "DDS " {
		return nil, fmt.Errorf("%w: not dds", ErrFormat)
	}
	u32 := func(o int) uint32 { return binary.LittleEndian.Uint32(b[o:]) }
	h, w := int(u32(12)), int(u32(16))
	mips := int(u32(28))
	if mips == 0 {
		mips = 1
	}
	if w == 0 || h == 0 || w > maxSide || h > maxSide || mips > 16 {
		return nil, fmt.Errorf("%w: dds %dx%d mips %d", ErrFormat, w, h, mips)
	}
	pfFlags, fourCC, rgbBits := u32(80), string(b[84:88]), int(u32(88))
	masks := [4]uint32{u32(92), u32(96), u32(100), u32(104)}
	d := &DDS{W: w, H: h}
	switch {
	case pfFlags&4 != 0 && fourCC == "DXT1":
		d.Format = FormatDXT1
	case pfFlags&4 != 0 && (fourCC == "DXT2" || fourCC == "DXT3"):
		d.Format = FormatDXT3
	case pfFlags&4 != 0 && (fourCC == "DXT4" || fourCC == "DXT5"):
		d.Format = FormatDXT5
	case pfFlags&0x40 != 0 && (rgbBits == 24 || rgbBits == 32):
		d.Format = FormatRGBA
	default:
		return nil, fmt.Errorf("%w: dds pixel format flags=%#x fourcc=%q bits=%d", ErrFormat, pfFlags, fourCC, rgbBits)
	}
	data := b[128:]
	lw, lh := w, h
	for i := 0; i < mips; i++ {
		var size int
		if d.Format == FormatRGBA {
			size = lw * lh * rgbBits / 8
		} else {
			size = ((lw + 3) / 4) * ((lh + 3) / 4) * blockBytes(d.Format)
		}
		if size > len(data) {
			if i == 0 {
				return nil, errors.New("image: truncated dds")
			}
			break // keep the complete levels
		}
		lvl := data[:size]
		if d.Format == FormatRGBA {
			lvl = unpackRGB(lvl, lw, lh, rgbBits/8, masks, pfFlags&1 != 0)
		}
		d.Levels = append(d.Levels, lvl)
		data = data[size:]
		lw, lh = max(lw/2, 1), max(lh/2, 1)
	}
	return d, nil
}

func shiftOf(m uint32) (shift uint, bits uint) {
	if m == 0 {
		return 0, 0
	}
	for m&1 == 0 {
		m >>= 1
		shift++
	}
	for m&1 == 1 {
		m >>= 1
		bits++
	}
	return
}

func unpackRGB(src []byte, w, h, ps int, masks [4]uint32, hasAlpha bool) []byte {
	out := make([]byte, w*h*4)
	var sh, bt [4]uint
	for i, m := range masks {
		sh[i], bt[i] = shiftOf(m)
	}
	for i := 0; i < w*h; i++ {
		var v uint32
		for k := 0; k < ps; k++ {
			v |= uint32(src[i*ps+k]) << (8 * k)
		}
		for c := 0; c < 4; c++ {
			if bt[c] == 0 {
				out[i*4+c] = 255
				continue
			}
			x := (v & masks[c]) >> sh[c]
			out[i*4+c] = byte(x * 255 / (1<<bt[c] - 1))
		}
		if !hasAlpha {
			out[i*4+3] = 255
		}
	}
	return out
}
