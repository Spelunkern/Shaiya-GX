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

func TestGMCyanNameTagGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("GL and resources")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	win, err := platform.Open(platform.Config{Width: 1280, Height: 800, Hidden: true, Debug: true})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
	if err = render.Init(true); err != nil {
		t.Fatal(err)
	}
	arc, err := archive.OpenDirectory(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	a, err := New(Config{}, arc)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	if err = a.Preview("world-stream"); err != nil {
		t.Fatal(err)
	}
	for i := 0; !a.SceneReady(); i++ {
		if i > 500 {
			t.Fatal("world did not load")
		}
		a.Draw(1280, 800)
	}
	out := os.Getenv("SHAIYA_CAPTURE_DIR")
	if out == "" {
		out = t.TempDir()
	}
	if err = os.MkdirAll(out, 0700); err != nil {
		t.Fatal(err)
	}

	slot := a.state.Slots[0]
	slot.Name = "Administrador"
	a.state.Slots[0] = slot
	a.state.Self.IsGM = true
	a.Draw(1280, 800)
	if err = render.SavePNG(filepath.Join(out, "gm-world.png"), 1280, 800); err != nil {
		t.Fatal(err)
	}
	// Isolate the name pass so UI colours cannot satisfy the GM tag check.
	gl.ClearColor(0, 0, 0, 1)
	gl.ClearDepth(1)
	gl.DepthMask(true)
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	a.sprites.Begin(1280, 800)
	a.drawPlayerNames(1280, 800)
	a.sprites.End()
	pixels := make([]byte, 1280*800*4)
	gl.ReadPixels(0, 0, 1280, 800, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(pixels))
	cyan, white := 0, 0
	for i := 0; i < len(pixels); i += 4 {
		r, g, b := int(pixels[i]), int(pixels[i+1]), int(pixels[i+2])
		if b > 150 && g > r+40 && b > g+15 {
			cyan++
		}
		if r > 180 && g > 180 && b > 180 {
			white++
		}
	}
	if cyan < 10 || white < 10 {
		t.Fatalf("GM prefix or white name missing: cyan=%d white=%d", cyan, white)
	}
	if err = render.SavePNG(filepath.Join(out, "gm-nameplate.png"), 1280, 800); err != nil {
		t.Fatal(err)
	}
	if code := gl.GetError(); code != gl.NO_ERROR {
		t.Fatalf("GM name GL error %#x", code)
	}
}
