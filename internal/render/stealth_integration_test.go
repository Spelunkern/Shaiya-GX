//go:build integration

package render

import (
	img "github.com/Spelunkern/Shaiya-GX/internal/data/image"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/go-gl/gl/v4.5-core/gl"
	"os"
	"runtime"
	"testing"
)

func TestStealthSurfaceDepthAndStateRestoreGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Title: "stealth", Width: 64, Height: 64, Hidden: true, Debug: true})
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
	k, err := NewSkinner()
	if err != nil {
		t.Fatal(err)
	}
	defer k.Delete()
	tex := Upload(img.Texture{RGBA: &img.RGBA{W: 1, H: 1, Pix: []byte{255, 0, 0, 255}}})
	defer tex.Delete()
	var vertices []SkinVertex
	for _, p := range []V3{{-.8, -.8, .5}, {.8, -.8, .5}, {.8, .8, .5}, {-.8, .8, .5}} {
		vertices = append(vertices, SkinVertex{Pos: p, Normal: V3{0, 1, 0}, Weight: 1})
	}
	mesh := NewSkinMesh(vertices, []uint32{0, 2, 1, 0, 3, 2})
	defer mesh.Delete()
	draw := func(ghost bool) [4]byte {
		gl.Viewport(0, 0, 64, 64)
		gl.DepthMask(true)
		gl.ClearDepth(1)
		gl.ClearColor(0, 0, 1, 1)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		s.Begin()
		k.Begin(s, 1)
		if ghost {
			k.SetOpacity(StealthOpacity)
		}
		k.SetPose(Identity(), nil)
		k.Draw(mesh, tex, .5)
		var pixel [4]byte
		gl.ReadPixels(32, 32, 1, 1, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&pixel[0]))
		return pixel
	}
	ghost := draw(true)
	if ghost[0] < 165 || ghost[0] > 167 || ghost[1] != 0 || ghost[2] < 88 || ghost[2] > 90 {
		t.Fatal("native alpha/cutout", ghost)
	}
	var depthMask bool
	gl.GetBooleanv(gl.DEPTH_WRITEMASK, &depthMask)
	if !depthMask || gl.IsEnabled(gl.BLEND) || gl.IsEnabled(gl.CULL_FACE) {
		t.Fatal("ghost leaked GL state")
	}
	if opaque := draw(false); opaque != [4]byte{255, 0, 0, 255} {
		t.Fatal("next skin was not opaque", opaque)
	}
	var rigid []Vertex
	for _, p := range []V3{{-.8, -.8, .5}, {.8, -.8, .5}, {.8, .8, .5}, {-.8, .8, .5}} {
		rigid = append(rigid, Vertex{Pos: p, Normal: V3{0, 1, 0}})
	}
	equipment := NewMesh(rigid, []uint32{0, 2, 1, 0, 3, 2})
	defer equipment.Delete()
	s.Ambient = 1
	gl.ClearColor(0, 0, 1, 1)
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	s.Begin()
	s.DrawModelFaded(equipment, tex, Identity(), .5, StealthOpacity)
	var faded [4]byte
	gl.ReadPixels(32, 32, 1, 1, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&faded[0]))
	if faded[0] < 165 || faded[0] > 167 || faded[1] != 0 || faded[2] < 88 || faded[2] > 90 {
		t.Fatal("equipment/cape opacity", faded)
	}
	back := NewSkinMesh(vertices, []uint32{0, 1, 2, 0, 2, 3})
	defer back.Delete()
	gl.ClearColor(0, 0, 1, 1)
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	s.Begin()
	k.Begin(s, 1)
	k.SetOpacity(StealthOpacity)
	k.SetPose(Identity(), nil)
	k.Draw(back, tex, .5)
	var backPixel [4]byte
	gl.ReadPixels(32, 32, 1, 1, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&backPixel[0]))
	if backPixel != [4]byte{0, 0, 255, 255} {
		t.Fatal("rear face is visible through the head", backPixel)
	}
	front := draw(true)
	green := Upload(img.Texture{RGBA: &img.RGBA{W: 1, H: 1, Pix: []byte{0, 255, 0, 255}}})
	defer green.Delete()
	behind := Identity()
	behind[14] = .1
	k.SetPose(behind, nil)
	k.Draw(mesh, green, .5)
	var occluded [4]byte
	gl.ReadPixels(32, 32, 1, 1, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&occluded[0]))
	if occluded != front {
		t.Fatal("far body surface bleeds through near body surface", front, occluded)
	}
	// Native opaque clothing often has zero texture alpha. Concealment replaces
	// that unused alpha; cutout parts must still reject actual transparent holes.
	zeroAlpha := Upload(img.Texture{RGBA: &img.RGBA{W: 1, H: 1, Pix: []byte{255, 0, 0, 0}}})
	defer zeroAlpha.Delete()
	for _, ref := range []float32{-1, .5} {
		gl.ClearColor(0, 0, 1, 1)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		s.Begin()
		k.Begin(s, 1)
		k.SetOpacity(StealthOpacity)
		k.SetPose(Identity(), nil)
		k.Draw(mesh, zeroAlpha, ref)
		var pixel [4]byte
		gl.ReadPixels(32, 32, 1, 1, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&pixel[0]))
		if ref < 0 {
			if pixel[0] < 165 || pixel[0] > 167 || pixel[1] != 0 {
				t.Fatal("opaque zero-alpha clothing disappeared", pixel)
			}
		} else if pixel != [4]byte{0, 0, 255, 255} {
			t.Fatal("cutout holes became opaque", pixel)
		}
	}

}
