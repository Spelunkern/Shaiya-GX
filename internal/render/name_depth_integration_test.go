//go:build integration

package render_test

import (
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	img "github.com/Spelunkern/Shaiya-GX/internal/data/image"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/go-gl/gl/v4.5-core/gl"
)

func TestNameDepthOccludesLabelsAndPreservesHUD(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires local GL context")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	w, err := platform.Open(platform.Config{Title: "name depth probe", Width: 64, Height: 64, Hidden: true})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	if err := render.Init(true); err != nil {
		t.Fatal(err)
	}
	s, err := render.NewSprites(16)
	if err != nil {
		t.Fatal(err)
	}
	white := render.Upload(img.Texture{RGBA: &img.RGBA{W: 1, H: 1, Pix: []byte{255, 255, 255, 255}}})
	defer white.Delete()
	gl.Viewport(0, 0, 64, 64)
	gl.ClearColor(0, 0, 1, 1)
	gl.ClearDepth(.25)
	gl.DepthMask(true)
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	s.Begin(64, 64)
	s.CaptureNameDepth(64, 64)
	if s.NameVisibleAt(12, 12, 64, 64, .75) || !s.NameVisibleAt(44, 12, 64, 64, .125) || s.NameVisibleAt(0, 0, 63, 64, .125) {
		t.Fatal("label click depth does not match occlusion or viewport")
	}
	s.InvalidateNameDepth()
	if s.NameVisibleAt(44, 12, 64, 64, .125) {
		t.Fatal("invalidated frame retained clickable depth")
	}
	gl.ClearDepth(.8)
	gl.Clear(gl.DEPTH_BUFFER_BIT)
	s.CaptureNameDepth(64, 64)
	if !s.NameVisibleAt(12, 12, 64, 64, .75) {
		t.Fatal("new capture reused a stale pixel")
	}
	gl.ClearDepth(.25)
	gl.Clear(gl.DEPTH_BUFFER_BIT)
	s.CaptureNameDepth(64, 64)
	s.NameDepth(.75)
	s.Draw(white, 8, 8, 16, 16, 0, 0, 1, 1, [4]float32{1, 0, 0, 1})
	s.NameDepth(.125)
	render.ResetDrawCalls()
	s.Draw(white, 40, 8, 16, 16, 0, 0, 1, 1, [4]float32{1, 0, 0, 1})
	s.NameDepth(.125)
	s.Draw(white, 40, 8, 16, 16, 0, 0, 1, 1, [4]float32{1, 0, 0, 1})
	s.NameDepth(-1)
	if calls := render.DrawCalls(); calls != 1 {
		t.Fatalf("same-depth labels used %d draw calls, want one", calls)
	}
	s.Draw(white, 8, 40, 16, 16, 0, 0, 1, 1, [4]float32{0, 1, 0, 1})
	s.End()
	path := filepath.Join(t.TempDir(), "name-depth.png")
	if err := render.SavePNG(path, 64, 64); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	picture, err := png.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	for _, sample := range []struct {
		x, y int
		want [3]uint32
	}{{12, 12, [3]uint32{0, 0, 65535}}, {44, 12, [3]uint32{65535, 0, 0}}, {12, 44, [3]uint32{0, 65535, 0}}} {
		r, g, b, _ := picture.At(sample.x, sample.y).RGBA()
		if [3]uint32{r, g, b} != sample.want {
			t.Fatal("wrong depth occlusion or state leaked into HUD", sample, r, g, b)
		}
	}
	if err := gl.GetError(); err != gl.NO_ERROR {
		t.Fatalf("name depth GL error %#x", err)
	}
}
