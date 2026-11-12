package image

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
)

// DecodeJPEG checks dimensions before the standard decoder allocates pixels.
func DecodeJPEG(b []byte) (*RGBA, error) {
	cfg, err := jpeg.DecodeConfig(bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("%w: jpeg: %v", ErrFormat, err)
	}
	if cfg.Width < 1 || cfg.Height < 1 || cfg.Width > maxSide || cfg.Height > maxSide {
		return nil, fmt.Errorf("%w: jpeg %dx%d", ErrFormat, cfg.Width, cfg.Height)
	}
	src, err := jpeg.Decode(bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("%w: jpeg: %v", ErrFormat, err)
	}
	dst := image.NewNRGBA(image.Rect(0, 0, cfg.Width, cfg.Height))
	draw.Draw(dst, dst.Bounds(), src, src.Bounds().Min, draw.Src)
	return &RGBA{W: cfg.Width, H: cfg.Height, Pix: dst.Pix}, nil
}
