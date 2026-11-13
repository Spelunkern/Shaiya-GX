//go:build integration

package ui

import (
	img "github.com/Spelunkern/Shaiya-GX/internal/data/image"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/go-gl/gl/v4.5-core/gl"
	"os"
	"runtime"
	"testing"
)

func TestNativeBlueCastPixelsGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Width: 64, Height: 96, Hidden: true})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
	if err = render.Init(false); err != nil {
		t.Fatal(err)
	}
	s, err := render.NewSprites(8)
	if err != nil {
		t.Fatal(err)
	}
	solid := render.Upload(img.Texture{RGBA: &img.RGBA{W: 1, H: 1, Pix: []byte{255, 255, 255, 255}}})
	defer solid.Delete()
	for _, state := range []struct {
		left float32
		fill int
	}{{3, 0}, {1.5, 31}, {.001, 62}, {0, 0}} {
		left := state.left
		gl.Viewport(0, 0, 64, 96)
		gl.ClearColor(1, 0, 0, 1)
		gl.Clear(gl.COLOR_BUFFER_BIT)
		s.Begin(64, 96)
		(&HUD{}).DrawCast(s, solid, 20, 10, left, 3)
		s.Flush()
		pixels := make([]byte, 64*96*4)
		gl.ReadPixels(0, 0, 64, 96, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&pixels[0]))
		for y := 0; y < 96; y++ {
			for x := 0; x < 64; x++ {
				want := [4]byte{255, 0, 0, 255}
				if left > 0 && x >= 20 && x < 25 && y >= 10 && y < 74 {
					want = [4]byte{0, 0, 0, 255}
				}
				fill := state.fill
				if left > 0 && x >= 21 && x < 24 && y >= 11 && y < 11+fill {
					want = [4]byte{0, 0, 255, 255}
				}
				i := ((95-y)*64 + x) * 4
				got := [4]byte(pixels[i : i+4])
				if got != want {
					t.Fatalf("left=%v pixel %d,%d: %v want %v", left, x, y, got, want)
				}
			}
		}
	}
}
