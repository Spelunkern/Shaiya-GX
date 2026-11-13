package image

import (
	"bytes"
	"encoding/binary"
	"fmt"
	stdimage "image"
	"image/color"
	"image/draw"
	"image/png"
)

var pngSignature = []byte{137, 80, 78, 71, 13, 10, 26, 10}

func pngPixels(b []byte) (*RGBA, error) {
	cfg, err := png.DecodeConfig(bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("image: png: %w", err)
	}
	if cfg.Width < 1 || cfg.Height < 1 || cfg.Width > maxSide || cfg.Height > maxSide {
		return nil, fmt.Errorf("image: png dimensions %dx%d", cfg.Width, cfg.Height)
	}
	src, err := png.Decode(bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	dst := stdimage.NewNRGBA(stdimage.Rect(0, 0, cfg.Width, cfg.Height))
	if palette, ok := src.(*stdimage.Paletted); ok {
		// Preserve RGB even under alpha zero: bilinear filtering can sample
		// those values at UI edges. Premultiplied draw paths discard them.
		colors := make([]color.NRGBA, len(palette.Palette))
		for i, c := range palette.Palette {
			colors[i] = color.NRGBAModel.Convert(c).(color.NRGBA)
		}
		for y := 0; y < cfg.Height; y++ {
			for x := 0; x < cfg.Width; x++ {
				c := colors[palette.Pix[y*palette.Stride+x]]
				o := y*dst.Stride + x*4
				dst.Pix[o], dst.Pix[o+1], dst.Pix[o+2], dst.Pix[o+3] = c.R, c.G, c.B, c.A
			}
		}
	} else {
		draw.Draw(dst, dst.Bounds(), src, src.Bounds().Min, draw.Src)
	}
	return &RGBA{W: cfg.Width, H: cfg.Height, Pix: dst.Pix}, nil
}

// DecodePNG reads standard lossless PNG pixels. The optional private ancillary
// gxMP chunk stores the original UI atlas mipmaps, each also encoded as PNG.
// Ordinary PNG tools can still display the base image without this extension.
func DecodePNG(b []byte) (Texture, error) {
	base, err := pngPixels(b)
	if err != nil {
		return Texture{}, err
	}
	result := Texture{RGBA: base}
	seen := false
	for off := 8; off+12 <= len(b); {
		n := int(binary.BigEndian.Uint32(b[off:]))
		if n > len(b)-off-12 {
			return Texture{}, fmt.Errorf("image: truncated PNG chunk")
		}
		if string(b[off+4:off+8]) == "gxMP" {
			if seen {
				return Texture{}, fmt.Errorf("image: duplicate PNG mipmaps")
			}
			seen = true
			payload := b[off+8 : off+8+n]
			if len(payload) < 4 {
				return Texture{}, fmt.Errorf("image: missing PNG mip count")
			}
			count := int(binary.LittleEndian.Uint32(payload))
			payload = payload[4:]
			if count < 1 || count > 15 {
				return Texture{}, fmt.Errorf("image: invalid PNG mip count")
			}
			levels := [][]byte{base.Pix}
			w, h := base.W, base.H
			for i := 0; i < count; i++ {
				if len(payload) < 4 {
					return Texture{}, fmt.Errorf("image: truncated PNG mipmap")
				}
				size := int(binary.LittleEndian.Uint32(payload))
				payload = payload[4:]
				if size > len(payload) {
					return Texture{}, fmt.Errorf("image: truncated PNG mipmap")
				}
				mip, e := pngPixels(payload[:size])
				if e != nil {
					return Texture{}, e
				}
				payload = payload[size:]
				w, h = max(1, w/2), max(1, h/2)
				if mip.W != w || mip.H != h {
					return Texture{}, fmt.Errorf("image: invalid PNG mip dimensions")
				}
				levels = append(levels, mip.Pix)
			}
			if len(payload) != 0 {
				return Texture{}, fmt.Errorf("image: trailing PNG mip bytes")
			}
			result = Texture{DDS: &DDS{W: base.W, H: base.H, Format: FormatRGBA, Levels: levels}}
		}
		off += 12 + n
	}
	return result, nil
}
