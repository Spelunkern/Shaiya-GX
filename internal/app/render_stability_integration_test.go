//go:build integration

package app

import (
	"bytes"
	"os"
	"runtime"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/go-gl/gl/v4.5-core/gl"
)

func TestWorldCharacterRemainsStableAcrossUIPassesGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires local GL context and original assets")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Title: "character stability", Width: 320, Height: 320, Hidden: true, Debug: true})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
	if err := render.Init(true); err != nil {
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
	draw := func() []byte {
		gl.Viewport(0, 0, 320, 320)
		gl.ClearDepth(1)
		gl.DepthMask(true)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		a.drawWorldView(320, 320)
		pixels := make([]byte, 320*320*4)
		gl.ReadPixels(0, 0, 320, 320, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&pixels[0]))
		return pixels
	}
	for i := 0; i < 240 && !a.SceneReady(); i++ {
		draw()
	}
	if !a.SceneReady() {
		t.Fatal("field did not finish streaming")
	}
	if a.scene3d.Proj[10] >= 400/float32(399.9) {
		t.Fatal("far clip truncates WLD fog end")
	}
	reference := draw()
	for frame := 0; frame < 12; frame++ {
		a.hudPortrait(320, 320)
		a.sprites.Begin(320, 320)
		a.drawInventoryPreview(320, 320)
		a.sprites.End()
		if got := draw(); !bytes.Equal(reference, got) {
			changed := 0
			for i := range got {
				if got[i] != reference[i] {
					changed++
				}
			}
			t.Fatalf("frozen world/character changed after UI pass: frame %d, %d channels", frame, changed)
		}
	}
	if code := gl.GetError(); code != gl.NO_ERROR {
		t.Fatalf("character stability GL error %#x", code)
	}
}
