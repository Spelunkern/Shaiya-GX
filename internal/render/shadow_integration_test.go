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

func TestShadowCaptureAlphaProjectionDepthAndReuse(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires local GL context")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Title: "shadow probe", Width: 64, Height: 64, Hidden: true, Debug: true})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
	if err = Init(true); err != nil {
		t.Fatal(err)
	}
	s, err := NewScene3D()
	if err != nil {
		t.Fatal(err)
	}
	s.View, s.Proj = Identity(), Identity()
	target, err := NewShadowMap()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if target != nil {
			target.Delete()
		}
	}()
	image := Upload(img.Texture{RGBA: &img.RGBA{W: 2, H: 1, Pix: []byte{255, 255, 255, 0, 255, 255, 255, 255}}})
	defer image.Delete()
	vertices := []Vertex{{Pos: V3{-1, -1, .5}, UV: [2]float32{0, 0}}, {Pos: V3{1, -1, .5}, UV: [2]float32{1, 0}}, {Pos: V3{1, 1, .5}, UV: [2]float32{1, 1}}, {Pos: V3{-1, 1, .5}, UV: [2]float32{0, 1}}}
	indices := []uint32{0, 1, 2, 0, 2, 3}
	mesh := NewMesh(vertices, indices)
	defer mesh.Delete()
	gl.Viewport(0, 0, 64, 64)
	for pass := 0; pass < 3; pass++ {
		s.Shadow = true
		alphaRef := float32(.5)
		if pass == 1 {
			alphaRef = -1
		} // opaque RGB textures can have unused zero alpha
		target.Capture(func() { s.Begin(); s.DrawModel(mesh, image, Identity(), alphaRef, false) })
		var viewport [4]int32
		gl.GetIntegerv(gl.VIEWPORT, &viewport[0])
		if viewport != [4]int32{0, 0, 64, 64} {
			t.Fatal("capture leaked viewport", viewport)
		}
		s.Shadow = false
		s.Begin()
		gl.ClearColor(1, 1, 1, 1)
		gl.ClearDepth(1)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		target.Project(s, Identity(), vertices, indices, .5)
		for _, sample := range []struct {
			x    int
			want byte
		}{{8, 255}, {56, 127}} {
			if pass == 1 {
				sample.want = 127
			}
			var pixel [4]byte
			gl.ReadPixels(int32(sample.x), 32, 1, 1, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&pixel[0]))
			if delta := int(pixel[0]) - int(sample.want); delta < -1 || delta > 1 {
				t.Fatal("shadow alpha/projection incorrect", pass, sample, pixel)
			}
			var depth float32
			gl.ReadPixels(int32(sample.x), 32, 1, 1, gl.DEPTH_COMPONENT, gl.FLOAT, gl.Ptr(&depth))
			if depth != 1 {
				t.Fatal("shadow wrote scene depth", depth)
			}
		}
		// A following ordinary model must retain its colour and opaque alpha.
		s.Begin()
		s.DrawSolid(mesh, [4]float32{1, 0, 0, 1})
		var pixel [4]byte
		gl.ReadPixels(56, 32, 1, 1, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&pixel[0]))
		if pixel != [4]byte{255, 0, 0, 255} {
			t.Fatal("shadow leaked into following pass", pixel)
		}
	}
	// A cutout receiver must not project shadows into its transparent holes.
	target.Capture(func() { s.Shadow = true; s.Begin(); s.DrawModel(mesh, image, Identity(), -1, false) })
	s.Shadow = false
	s.Begin()
	gl.ClearColor(1, 1, 1, 1)
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	target.ProjectMasked(s, Identity(), vertices, indices, .5, image)
	for _, sample := range []struct {
		x    int32
		want byte
	}{{8, 255}, {56, 127}} {
		var pixel [4]byte
		gl.ReadPixels(sample.x, 32, 1, 1, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&pixel[0]))
		if delta := int(pixel[0]) - int(sample.want); delta < -1 || delta > 1 {
			t.Fatalf("receiver alpha at %d: %v", sample.x, pixel)
		}
	}
	color, fbo, buffer := target.color, target.fbo, target.depth
	target.Delete()
	target = nil
	if gl.IsTexture(color) || gl.IsFramebuffer(fbo) || gl.IsRenderbuffer(buffer) {
		t.Fatal("shadow resources survived deletion")
	}
	if code := gl.GetError(); code != gl.NO_ERROR {
		t.Fatalf("shadow GL error %#x", code)
	}
}
