//go:build integration

package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/go-gl/gl/v4.5-core/gl"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestOriginalInventoryGoldGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL and original assets")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Width: 1024, Height: 768, Hidden: true, Debug: true})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
	if err = render.Init(true); err != nil {
		t.Fatal(err)
	}
	arc, err := archive.Open(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	a, err := New(Config{}, arc)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	gl.Viewport(0, 0, 1024, 768)
	gl.ClearColor(.08, .1, .12, 1)
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	a.sprites.Begin(1024, 768)
	a.inventoryUI.Draw(a.sprites, a.font, 1024, 768, 1, 10_000_000, "")
	a.sprites.End()
	x, y := a.inventoryUI.Origin(1024, 768)
	pixels := make([]byte, 110*20*4)
	gl.ReadPixels(int32(x+110), int32(768-y-480), 110, 20, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&pixels[0]))
	orange := 0
	for i := 0; i < len(pixels); i += 4 {
		if pixels[i] > 180 && pixels[i+1] > 65 && pixels[i+1] < 165 && pixels[i+2] < 60 {
			orange++
		}
	}
	if orange < 10 {
		t.Fatal("ten million gold did not render in native orange", orange)
	}
	if code := gl.GetError(); code != gl.NO_ERROR {
		t.Fatalf("gold GL error %#x", code)
	}
	dir := os.Getenv("SHAIYA_TEST_ARTIFACTS")
	if dir == "" {
		dir = t.TempDir()
	}
	if err = render.SavePNG(filepath.Join(dir, "gold-ten-million.png"), 1024, 768); err != nil {
		t.Fatal(err)
	}
}
