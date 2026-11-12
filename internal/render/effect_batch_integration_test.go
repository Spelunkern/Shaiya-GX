//go:build integration

package render

import (
	"bytes"
	"os"
	"runtime"
	"testing"

	img "github.com/Spelunkern/Shaiya-GX/internal/data/image"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/go-gl/gl/v4.5-core/gl"
)

func TestEffectBatchPreservesBlendingGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	window, err := platform.Open(platform.Config{Title: "effect batch", Width: 64, Height: 64, Hidden: true, Debug: true})
	if err != nil {
		t.Fatal(err)
	}
	defer window.Close()
	if err := Init(true); err != nil {
		t.Fatal(err)
	}
	s, err := NewScene3D()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Delete()
	s.View, s.Proj = Identity(), Identity()
	mesh := NewMesh([]Vertex{{Pos: V3{-.6, -.6, .5}}, {Pos: V3{.6, -.6, .5}}, {Pos: V3{.6, .6, .5}}, {Pos: V3{-.6, .6, .5}}}, []uint32{0, 1, 2, 0, 2, 3})
	defer mesh.Delete()
	texture := Upload(img.Texture{RGBA: &img.RGBA{W: 1, H: 1, Pix: []byte{255, 255, 255, 255}}})
	defer texture.Delete()
	worlds := []M4{Identity(), Identity(), Identity()}
	worlds[0][12], worlds[1][12], worlds[2][13] = -.2, .2, .2
	colors := [][4]float32{{1, 0, 0, .4}, {0, 1, 0, .6}, {0, 0, 1, .7}}
	draw := func(batched bool, source, destination int32) ([]byte, int) {
		gl.Viewport(0, 0, 64, 64)
		gl.DepthMask(true)
		gl.ClearColor(.1, .1, .1, 1)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		s.Begin()
		ResetDrawCalls()
		if batched {
			s.DrawEffectBatch(mesh, texture, worlds, colors, source, destination)
		} else {
			for i := range worlds {
				s.DrawEffect(mesh, texture, worlds[i], colors[i], source, destination)
			}
		}
		// A later ordinary effect must not inherit the per-instance colors.
		following := Identity()
		following[12], following[13] = .4, -.4
		s.DrawEffect(mesh, texture, following, [4]float32{.3, .7, .2, .4}, 4, 5)
		calls := DrawCalls()
		pixels := make([]byte, 64*64*4)
		gl.ReadPixels(0, 0, 64, 64, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&pixels[0]))
		return pixels, calls
	}
	for _, blend := range [][2]int32{{4, 5}, {4, 1}, {1, 1}, {2, 0}, {11, 8}, {12, 8}} {
		old, before := draw(false, blend[0], blend[1])
		now, after := draw(true, blend[0], blend[1])
		if !bytes.Equal(old, now) {
			t.Fatalf("batch changed blended pixels for %v", blend)
		}
		if before != 4 || after != 2 {
			t.Fatalf("blend %v: draw calls %d -> %d", blend, before, after)
		}
	}
	if code := gl.GetError(); code != gl.NO_ERROR {
		t.Fatalf("GL error %#x", code)
	}
}
