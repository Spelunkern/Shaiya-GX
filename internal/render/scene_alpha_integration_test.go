//go:build integration

package render

import (
	"os"
	"runtime"
	"testing"

	img "github.com/Spelunkern/Shaiya-GX/internal/data/image"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/go-gl/gl/v4.5-core/gl"
)

func TestDispelDestinationAlphaStaysOpaqueGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	w, err := platform.Open(platform.Config{Width: 32, Height: 32, Hidden: true})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	if err = Init(true); err != nil {
		t.Fatal(err)
	}
	s, err := NewScene3D()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Delete()
	s.View, s.Proj = Identity(), Identity()
	mesh := NewMesh([]Vertex{{Pos: V3{-1, -1, .5}}, {Pos: V3{1, -1, .5}}, {Pos: V3{1, 1, .5}}, {Pos: V3{-1, 1, .5}}}, []uint32{0, 1, 2, 0, 2, 3})
	defer mesh.Delete()
	texture := Upload(img.Texture{RGBA: &img.RGBA{W: 1, H: 1, Pix: []byte{255, 255, 255, 128}}})
	defer texture.Delete()
	target, err := NewWaterTarget(32, 32)
	if err != nil {
		t.Fatal(err)
	}
	defer target.Delete()
	gl.ColorMask(true, false, true, true)
	target.Capture(V3{.2, .4, .6}, func() {
		s.Begin()
		// An earlier translucent layer must still blend with its texture alpha.
		s.DrawEffect(mesh, texture, Identity(), [4]float32{1, 0, 0, 1}, 4, 5)
		var before, after [4]byte
		gl.ReadPixels(16, 16, 1, 1, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&before[0]))
		if before[0] < 152 || before[0] > 155 || before[2] < 75 || before[2] > 77 || before[3] != 255 {
			t.Fatal("source transparency or opaque scene alpha lost", before)
		}
		// Dispel's destination-alpha dependency: with alpha=1 this layer
		// leaves the background intact, rather than multiplying it toward black.
		s.DrawEffect(mesh, texture, Identity(), [4]float32{0, 0, 0, 1}, 0, 6)
		gl.ReadPixels(16, 16, 1, 1, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&after[0]))
		if before != after {
			t.Fatal("destination-alpha layer darkened background", before, after)
		}
		s.End()
	})
	var mask [4]bool
	gl.GetBooleanv(gl.COLOR_WRITEMASK, &mask[0])
	if mask != [4]bool{true, false, true, true} {
		t.Fatal("color write mask leaked", mask)
	}
	// The scope must initialize alpha even with alpha writes initially disabled,
	// and must preserve the caller's RGB mask, scissor and clear colour.
	gl.ColorMask(true, true, true, true)
	gl.ClearColor(.3, .5, .7, 0)
	gl.Clear(gl.COLOR_BUFFER_BIT)
	gl.ColorMask(false, true, true, false)
	gl.Scissor(0, 0, 1, 1)
	gl.Enable(gl.SCISSOR_TEST)
	restore := KeepSceneAlphaOpaque()
	var p [4]byte
	gl.ReadPixels(16, 16, 1, 1, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&p[0]))
	if p[3] != 255 || !gl.IsEnabled(gl.SCISSOR_TEST) {
		t.Fatal("alpha init/scissor", p)
	}
	restore()
	gl.GetBooleanv(gl.COLOR_WRITEMASK, &mask[0])
	if mask != [4]bool{false, true, true, false} {
		t.Fatal("scope did not restore mask", mask)
	}
	gl.Disable(gl.SCISSOR_TEST)
	gl.ColorMask(true, true, true, true)
	if code := gl.GetError(); code != gl.NO_ERROR {
		t.Fatalf("GL error %#x", code)
	}
}
