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
	glyph := render.Upload(img.Texture{RGBA: &img.RGBA{W: 1, H: 1, Pix: []byte{255, 0, 0, 255}}})
	defer glyph.Delete()
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
	render.ResetDrawCalls()
	s.NameDepth(.75)
	s.Draw(glyph, 8, 8, 16, 16, 0, 0, 1, 1, [4]float32{1, 1, 1, 1})
	s.NameDepth(.125)
	s.Draw(glyph, 40, 8, 16, 16, 0, 0, 1, 1, [4]float32{1, 1, 1, 1})
	s.NameDepth(-1)
	s.DrawFlat(glyph, 8, 40, 16, 16, [4]float32{0, 1, 0, 1})
	s.End()
	if calls := render.DrawCalls(); calls != 1 {
		t.Fatalf("labels and solid quad used %d draw calls, want one", calls)
	}
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
	// A cooldown triangle can leave room for another triangle in the batch
	// while the following quad would overflow the vertex buffer.
	render.ResetDrawCalls()
	for range 15 {
		s.Draw(glyph, 0, 0, 4, 4, 0, 0, 1, 1, [4]float32{1, 1, 1, 1})
	}
	s.DrawCooldown(glyph, 8, 24, 16, .125, [4]float32{1, 1, 1, 1})
	s.Draw(glyph, 40, 24, 16, 16, 0, 0, 1, 1, [4]float32{1, 1, 1, 1})
	s.End()
	if calls := render.DrawCalls(); calls != 2 {
		t.Fatalf("mixed triangle/quad capacity used %d draw calls, want two", calls)
	}
	if err := gl.GetError(); err != gl.NO_ERROR {
		t.Fatalf("name depth GL error %#x", err)
	}
}

func TestWaterTargetColorOnlyNameDepthGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires local GL context")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	w, err := platform.Open(platform.Config{Title: "color only depth", Width: 64, Height: 64, Hidden: true})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	if err := render.Init(false); err != nil {
		t.Fatal(err)
	}
	target, err := render.NewWaterTarget(64, 64)
	if err != nil {
		t.Fatal(err)
	}
	defer target.Delete()
	s, err := render.NewSprites(16)
	if err != nil {
		t.Fatal(err)
	}
	glyph := render.Upload(img.Texture{RGBA: &img.RGBA{W: 1, H: 1, Pix: []byte{255, 0, 0, 255}}})
	defer glyph.Delete()

	gl.Viewport(0, 0, 64, 64)
	target.Capture(render.V3{0, 0, 1}, func() {
		gl.ClearDepth(.25)
		gl.DepthMask(true)
		gl.Clear(gl.DEPTH_BUFFER_BIT)
		target.BeginColorOnly()
		depthID, dw, dh := target.DepthTexture()
		s.Begin(64, 64)
		s.UseNameDepthTexture(depthID, dw, dh, false)
		if s.NameVisibleAt(12, 12, 64, 64, .75) || !s.NameVisibleAt(44, 12, 64, 64, .125) {
			t.Fatal("direct depth texture visibility mismatch")
		}
		render.ResetDrawCalls()
		s.NameDepth(.75)
		s.Draw(glyph, 8, 8, 16, 16, 0, 0, 1, 1, [4]float32{1, 1, 1, 1})
		s.NameDepth(.125)
		s.Draw(glyph, 40, 8, 16, 16, 0, 0, 1, 1, [4]float32{1, 1, 1, 1})
		s.NameDepth(-1)
		s.DrawFlat(glyph, 8, 40, 16, 16, [4]float32{0, 1, 0, 1})
		s.End()
		if calls := render.DrawCalls(); calls != 1 {
			t.Fatalf("color-only nameplate batch used %d draw calls, want 1", calls)
		}
	})
	target.Present()
	path := filepath.Join(t.TempDir(), "color-only-depth.png")
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
			t.Fatal("wrong depth occlusion on color-only target", sample, r, g, b)
		}
	}
	if err := gl.GetError(); err != gl.NO_ERROR {
		t.Fatalf("color-only name depth GL error %#x", err)
	}
}
