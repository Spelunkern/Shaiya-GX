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

func TestWaterClipStaticAndSkinnedGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires local GL context")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Title: "water clip", Width: 64, Height: 64, Hidden: true, Debug: true})
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
	defer s.Delete()
	k, err := NewSkinner()
	if err != nil {
		t.Fatal(err)
	}
	defer k.Delete()
	s.View, s.Proj, s.Ambient = Identity(), Identity(), 1
	s.ClipPlane = [4]float32{0, 1, 0, 0}
	points := []V3{{-1, -1, .5}, {1, -1, .5}, {1, 1, .5}, {-1, 1, .5}}
	vertices := make([]Vertex, 0, 4)
	skin := make([]SkinVertex, 0, 4)
	for _, p := range points {
		vertices = append(vertices, Vertex{Pos: p, Normal: V3{0, 1, 0}})
		skin = append(skin, SkinVertex{Pos: p, Normal: V3{0, 1, 0}})
	}
	indices := []uint32{0, 1, 2, 0, 2, 3}
	mesh, skinMesh := NewMesh(vertices, indices), NewSkinMesh(skin, indices)
	defer mesh.Delete()
	defer skinMesh.Delete()
	for _, skinned := range []bool{false, true} {
		gl.Viewport(0, 0, 64, 64)
		gl.ClearColor(0, 0, 0, 1)
		gl.DepthMask(true)
		gl.ClearDepth(1)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		s.Begin()
		if skinned {
			k.Begin(s, 1)
			k.SetPose(Identity(), nil)
			k.Draw(skinMesh, nil, -1)
		} else {
			s.DrawModel(mesh, nil, Identity(), -1, false)
		}
		for _, sample := range []struct {
			y     int32
			white bool
		}{{16, false}, {48, true}} {
			var pixel [4]byte
			gl.ReadPixels(32, sample.y, 1, 1, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&pixel[0]))
			if (pixel[0] > 200) != sample.white {
				t.Fatalf("skinned=%v y=%d pixel=%v", skinned, sample.y, pixel)
			}
		}
	}
	s.End()
	if gl.IsEnabled(gl.CLIP_DISTANCE0) {
		t.Fatal("clip state leaked into UI")
	}
	if code := gl.GetError(); code != gl.NO_ERROR {
		t.Fatalf("GL error %#x", code)
	}
}

func TestWaterReflectionTargetCompositeAndRestoreGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires local GL context")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Title: "water reflection", Width: 64, Height: 64, Hidden: true, Debug: true})
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
	defer s.Delete()
	target, err := NewWaterTarget(32, 32)
	if err != nil {
		t.Fatal(err)
	}
	defer target.Delete()
	texture := Upload(img.Texture{RGBA: &img.RGBA{W: 1, H: 1, Pix: []byte{255, 255, 255, 255}}})
	defer texture.Delete()
	mesh := NewMesh([]Vertex{{Pos: V3{-1, -1, .5}}, {Pos: V3{1, -1, .5}}, {Pos: V3{1, 1, .5}}, {Pos: V3{-1, 1, .5}}}, []uint32{0, 1, 2, 0, 2, 3})
	defer mesh.Delete()
	gl.Viewport(0, 0, 64, 64)
	gl.Scissor(2, 3, 60, 58)
	gl.Enable(gl.SCISSOR_TEST)
	target.Capture(V3{1, 0, 0}, func() {})
	var viewport, scissor [4]int32
	var framebuffer int32
	gl.GetIntegerv(gl.VIEWPORT, &viewport[0])
	gl.GetIntegerv(gl.SCISSOR_BOX, &scissor[0])
	gl.GetIntegerv(gl.DRAW_FRAMEBUFFER_BINDING, &framebuffer)
	if viewport != [4]int32{0, 0, 64, 64} || scissor != [4]int32{2, 3, 60, 58} || framebuffer != 0 || !gl.IsEnabled(gl.SCISSOR_TEST) {
		t.Fatal("capture leaked framebuffer or viewport state")
	}
	gl.Disable(gl.SCISSOR_TEST)
	s.View, s.Proj, s.WaterReflection = Identity(), Identity(), target
	gl.ClearColor(0, 0, 0, 1)
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	s.Begin()
	s.DrawWater(mesh, texture, 1, [4]float32{1, 1, 1, 1})
	var pixel [4]byte
	gl.ReadPixels(32, 32, 1, 1, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&pixel[0]))
	if pixel[0] < 250 || pixel[1] < 225 || pixel[1] > 233 || pixel[2] != pixel[1] {
		t.Fatal("reflection was not composited", pixel)
	}
	normal := Upload(img.Texture{RGBA: &img.RGBA{W: 1, H: 1, Pix: []byte{128, 128, 255, 255}}})
	defer normal.Delete()
	s.WaterNormals = [2]*Texture{normal, normal}
	gl.ClearColor(0, 0, 1, 1)
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	s.CaptureWaterBackground()
	s.Begin()
	s.DrawWater(mesh, texture, 1, [4]float32{1, 1, 1, 1})
	gl.ReadPixels(32, 32, 1, 1, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&pixel[0]))
	if pixel[0] < 180 || pixel[1] > 150 || pixel[2] < 180 {
		t.Fatal("screen refraction and reflected color did not reach water", pixel)
	}
	s.End()
	if code := gl.GetError(); code != gl.NO_ERROR {
		t.Fatalf("GL error %#x", code)
	}
}
