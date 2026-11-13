package image

import "testing"

func TestOccluderOpacityChecksEveryMipAndCompressedAlpha(t *testing.T) {
	if Opaque(Texture{}) || Opaque(Texture{RGBA: &RGBA{W: 1, H: 1, Pix: []byte{255, 255, 255, 0}}}) {
		t.Fatal("unknown/transparent texture occludes")
	}
	if !Opaque(Texture{RGBA: &RGBA{W: 1, H: 1, Pix: []byte{1, 2, 3, 255}}}) {
		t.Fatal("opaque RGBA rejected")
	}
	dxt1 := []byte{255, 255, 0, 0, 255, 255, 255, 255}
	if !Opaque(Texture{DDS: &DDS{Format: FormatDXT1, Levels: [][]byte{dxt1}}}) {
		t.Fatal("four-colour DXT1 should be opaque")
	}
	dxt1[0], dxt1[1] = 0, 0
	if Opaque(Texture{DDS: &DDS{Format: FormatDXT1, Levels: [][]byte{dxt1}}}) {
		t.Fatal("DXT1 cutout accepted")
	}
	dxt3 := make([]byte, 16)
	for i := 0; i < 8; i++ {
		dxt3[i] = 255
	}
	if !Opaque(Texture{DDS: &DDS{Format: FormatDXT3, Levels: [][]byte{dxt3}}}) {
		t.Fatal("opaque DXT3 rejected")
	}
	dxt3[7] = 0
	if Opaque(Texture{DDS: &DDS{Format: FormatDXT3, Levels: [][]byte{dxt3}}}) {
		t.Fatal("DXT3 alpha ignored")
	}
	dxt5 := make([]byte, 16)
	dxt5[0], dxt5[1] = 255, 255
	for selector := 0; selector < 8; selector++ {
		dxt5[2] = byte(selector)
		if got := Opaque(Texture{DDS: &DDS{Format: FormatDXT5, Levels: [][]byte{dxt5}}}); got != (selector != 6) {
			t.Fatal("DXT5 six-alpha selector", selector, got)
		}
	}
	if Opaque(Texture{DDS: &DDS{Format: FormatRGBA, Levels: [][]byte{{1, 2, 3, 255}, {1, 2, 3, 254}}}}) {
		t.Fatal("transparent authored mip accepted")
	}
}
