//go:build integration

package app

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/ui"
	"github.com/go-gl/gl/v4.5-core/gl"
)

func TestBlessingBarUsesOriginalFooterGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires original assets and GL")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Title: "blessing", Width: 1024, Height: 768, Hidden: true, Debug: true})
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
	draw := func(width int, known bool, count uint32) []byte {
		gl.Viewport(0, 0, int32(width), 768)
		gl.ClearColor(0, 0, 0, 1)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		a.sprites.Begin(width, 768)
		a.bottom.Draw(a.sprites, a.font, a.white, width, 768, ui.HUDState{BlessKnown: known, BlessCount: count, BlessMax: 12288}, false, false, false, false)
		a.sprites.End()
		pixels := make([]byte, width*768*4)
		gl.ReadPixels(0, 0, int32(width), 768, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&pixels[0]))
		return pixels
	}
	for _, width := range []int{800, 1024} {
		base := draw(width, false, 0)
		filled := draw(width, true, 6144)
		if bytes.Equal(base, filled) {
			t.Fatalf("blessing bar absent at width %d", width)
		}
		if again := draw(width, true, 6144); !bytes.Equal(filled, again) {
			t.Fatalf("bar changed while idle at width %d", width)
		}
	}
	if dir := os.Getenv("SHAIYA_TEST_ARTIFACTS"); dir != "" {
		if err := render.SavePNG(filepath.Join(dir, "blessing-bar.png"), 1024, 768); err != nil {
			t.Fatal(err)
		}
	}
	if code := gl.GetError(); code != gl.NO_ERROR {
		t.Fatalf("GL error %#x", code)
	}
}
