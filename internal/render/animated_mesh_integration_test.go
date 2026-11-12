//go:build integration

package render

import (
	"bytes"
	"os"
	"runtime"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/go-gl/gl/v4.5-core/gl"
)

func TestAnimatedMeshMatchesStreamGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	window, err := platform.Open(platform.Config{Title: "animated mesh", Width: 64, Height: 64, Hidden: true})
	if err != nil {
		t.Fatal(err)
	}
	defer window.Close()
	if err := Init(false); err != nil {
		t.Fatal(err)
	}
	s, err := NewScene3D()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Delete()
	s.View, s.Proj, s.Ambient = Identity(), Identity(), 1
	indices := []uint32{0, 1, 2}
	vertices := []Vertex{{Pos: V3{-.8, -.8, .5}}, {Pos: V3{.8, -.8, .5}}, {Pos: V3{0, .8, .5}}}
	stream := NewStreamMesh()
	defer stream.Delete()
	animated := NewAnimatedMesh(len(vertices), indices)
	defer animated.Delete()
	for _, offset := range []float32{0, .2, -.3} {
		vertices[2].Pos[0] = offset
		stream.Stream(vertices, indices)
		animated.UpdateVertices(vertices)
		draw := func(mesh *Mesh) []byte {
			gl.Viewport(0, 0, 64, 64)
			gl.DepthMask(true)
			gl.ClearColor(0, 0, 0, 1)
			gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
			s.Begin()
			s.DrawModel(mesh, nil, Identity(), -1, false)
			pixels := make([]byte, 64*64*4)
			gl.ReadPixels(0, 0, 64, 64, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&pixels[0]))
			return pixels
		}
		old, updated := draw(stream), draw(animated)
		if !bytes.Equal(old, updated) {
			t.Fatalf("pose %v changed pixels", offset)
		}
	}
	if code := gl.GetError(); code != gl.NO_ERROR {
		t.Fatalf("GL error %#x", code)
	}
}
