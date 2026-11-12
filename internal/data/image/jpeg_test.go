package image

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/jpeg"
	"testing"
)

func TestJPEGContentDetectionAndLimits(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	for y := 0; y < 2; y++ {
		for x := 0; x < 2; x++ {
			src.SetNRGBA(x, y, color.NRGBA{R: 200, G: 70, B: 20, A: 255})
		}
	}
	var b bytes.Buffer
	if err := jpeg.Encode(&b, src, &jpeg.Options{Quality: 100}); err != nil {
		t.Fatal(err)
	}
	tex, err := Decode(b.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if tex.RGBA == nil || tex.RGBA.W != 2 || tex.RGBA.H != 2 || tex.RGBA.Pix[3] != 255 || tex.RGBA.Pix[0] < 190 {
		t.Fatal("JPEG pixels/dimensions", tex)
	}
	oversized := append([]byte(nil), b.Bytes()...)
	sof := bytes.Index(oversized, []byte{0xff, 0xc0})
	if sof < 0 {
		t.Fatal("fixture has no SOF")
	}
	binary.BigEndian.PutUint16(oversized[sof+5:], maxSide+1)
	if _, err := DecodeJPEG(oversized); err == nil {
		t.Fatal("oversized JPEG accepted")
	}
	if _, err := DecodeJPEG(b.Bytes()[:30]); err == nil {
		t.Fatal("truncated JPEG accepted")
	}
}
func FuzzJPEG(f *testing.F) {
	var seed bytes.Buffer
	if err := jpeg.Encode(&seed, image.NewGray(image.Rect(0, 0, 1, 1)), nil); err != nil {
		f.Fatal(err)
	}
	f.Add(seed.Bytes())
	f.Add([]byte{0xff, 0xd8, 0xff, 0xd9})
	f.Fuzz(func(t *testing.T, b []byte) { DecodeJPEG(b) })
}
