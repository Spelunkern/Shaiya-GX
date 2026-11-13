//go:build integration

package platform

import (
	"github.com/go-gl/gl/v4.5-core/gl"
	"github.com/go-gl/glfw/v3.4/glfw"
	"os"
	"runtime"
	"testing"
)

// The null-platform monitor exercises real GLFW transitions without changing
// the user's monitor mode or showing a fullscreen window on their desktop.
func TestDisplayModeRoundTripHeadless(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_DISPLAY_MODES") != "1" {
		t.Skip("requires isolated null-platform test")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	glfw.InitHint(glfw.PlatformHint, int(glfw.PlatformNull))
	defer glfw.InitHint(glfw.PlatformHint, int(glfw.AnyPlatform))
	if err := glfw.Init(); err != nil {
		t.Fatal(err)
	}
	defer glfw.Terminate()
	glfw.DefaultWindowHints()
	defer glfw.DefaultWindowHints()
	glfw.WindowHint(glfw.ClientAPI, glfw.NoAPI)
	glfw.WindowHint(glfw.Visible, glfw.False)
	raw, err := glfw.CreateWindow(640, 480, "isolated mode test", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Destroy()
	w := &Window{Window: raw, normalSize: windowState{Width: 640, Height: 480}}
	modes := w.DisplayModes()
	if len(modes) < 2 {
		t.Fatal("null monitor has no test mode")
	}
	mode := modes[len(modes)-1]
	if err = w.ApplyDisplayMode(mode); err != nil {
		t.Fatal(err)
	}
	width, height := w.GetSize()
	if w.GetMonitor() == nil || width != mode.Width || height != mode.Height {
		t.Fatal("fullscreen transition", width, height)
	}
	if err = w.ApplyDisplayMode(DisplayMode{}); err != nil {
		t.Fatal(err)
	}
	width, height = w.GetSize()
	if w.GetMonitor() != nil || width != 640 || height != 480 {
		t.Fatal("window size was not restored", width, height)
	}
	if err = w.ApplyDisplayMode(DisplayMode{1234, 567}); err == nil {
		t.Fatal("accepted a non-advertised mode")
	}
	t.Logf("null monitor fullscreen %dx%d -> window 640x480; unsupported mode refused", mode.Width, mode.Height)
}

func TestWindowedResizePreservesGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires real GL context")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	w, err := Open(Config{Width: 640, Height: 480, Hidden: true})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	if err = gl.Init(); err != nil {
		t.Fatal(err)
	}
	var tex uint32
	gl.CreateTextures(gl.TEXTURE_2D, 1, &tex)
	defer gl.DeleteTextures(1, &tex)
	gl.TextureStorage2D(tex, 1, gl.RGBA8, 1, 1)
	data := [4]byte{17, 93, 201, 255}
	gl.TextureSubImage2D(tex, 0, 0, 0, 1, 1, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&data[0]))
	w.SetSize(800, 600)
	glfw.PollEvents()
	if err = w.ApplyDisplayMode(DisplayMode{}); err != nil {
		t.Fatal(err)
	}
	var after [4]byte
	gl.GetTextureImage(tex, 0, gl.RGBA, gl.UNSIGNED_BYTE, 4, gl.Ptr(&after[0]))
	if after != data || gl.GetError() != gl.NO_ERROR {
		t.Fatal("window resize lost GL resources", after)
	}
	width, height := w.GetSize()
	if width != 800 || height != 600 {
		t.Fatal("window resize failed", width, height)
	}
}
