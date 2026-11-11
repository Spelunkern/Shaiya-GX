package render

import (
	"image"
	"image/png"
	"os"
	"unsafe"

	"github.com/go-gl/gl/v4.5-core/gl"
)

// SavePNG reads the default framebuffer (w x h) into a PNG file.
func SavePNG(path string, w, h int) error {
	px := make([]byte, w*h*4)
	gl.PixelStorei(gl.PACK_ALIGNMENT, 1)
	gl.ReadPixels(0, 0, int32(w), int32(h), gl.RGBA, gl.UNSIGNED_BYTE, unsafe.Pointer(&px[0]))
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		copy(img.Pix[y*w*4:(y+1)*w*4], px[(h-1-y)*w*4:(h-y)*w*4])
	}
	for i := 3; i < len(img.Pix); i += 4 {
		img.Pix[i] = 255
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}
