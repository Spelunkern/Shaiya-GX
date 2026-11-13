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

func TestStaticIndexWorldPixelsGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("GL")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Width: 640, Height: 360, Hidden: true})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
	if err := render.Init(false); err != nil {
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
	if err := a.Preview("world-stream"); err != nil {
		t.Fatal(err)
	}
	a.frameDT = 0
	for i := 0; i < 8 || !a.SceneReady(); i++ {
		if i >= 240 {
			t.Fatal("world residency did not complete")
		}
		gl.Viewport(0, 0, 640, 360)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		a.drawWorldView(640, 360)
	}
	if !a.SceneReady() {
		t.Fatal("world not ready")
	}
	if dir := os.Getenv("SHAIYA_TEST_ARTIFACTS"); dir != "" {
		if err := render.SavePNG(filepath.Join(dir, "static-index-world.png"), 640, 360); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("resident GPU bytes %d, draw calls %d", render.GPUBytes(), render.DrawCalls())
}
