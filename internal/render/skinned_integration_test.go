//go:build integration

package render

import (
	"math"
	"os"
	"runtime"
	"testing"

	img "github.com/Spelunkern/Shaiya-GX/internal/data/image"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/go-gl/gl/v4.5-core/gl"
)

func TestSkinPaletteSupportsFullByteIndicesAndZeroWeightGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires local GL context")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Title: "bone palette", Width: 64, Height: 64, Hidden: true, Debug: true})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
	if err := Init(true); err != nil {
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
	for _, p := range []V3{{-.3, -.3, .5}, {.3, -.3, .5}, {.3, .3, .5}, {-.3, .3, .5}} {
		vertices = append(vertices, SkinVertex{Pos: p, Normal: V3{0, 1, 0}, Weight: 1, Bone0: 200, Bone1: 255})
	}
	mesh := NewSkinMesh(vertices, []uint32{0, 1, 2, 0, 2, 3})
	bones := make([]M4, 256)
	for i := range bones {
		bones[i] = Identity()
	}
	bones[200][12] = .5
	for i := range bones[255] {
		bones[255][i] = float32(math.NaN())
	}
	draw := func(pose []M4) {
		gl.Viewport(0, 0, 64, 64)
		gl.DepthMask(true)
		gl.ClearDepth(1)
		gl.ClearColor(0, 0, 0, 1)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		s.Begin()
		k.Begin(s, 1)
		k.SetPose(Identity(), pose)
		k.Draw(mesh, tex, -1)
	}
	draw(bones)
	var pixel [4]byte
	gl.ReadPixels(48, 32, 1, 1, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&pixel[0]))
	if pixel != [4]byte{255, 0, 0, 255} {
		t.Fatal("bone200 or zero-weight NaN bone corrupts skin", pixel)
	}
	// A later short palette must not use the stale matrices of a previous actor.
	short := []M4{Identity(), Identity(), Identity(), Identity()}
	short[3][12] = -.5
	draw(short)
	gl.ReadPixels(16, 32, 1, 1, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&pixel[0]))
	if pixel != [4]byte{255, 0, 0, 255} {
		t.Fatal("short palette used stale actor matrices", pixel)
	}

	// Missing materials must not sample whichever actor/water texture was last bound.
	for _, rgba := range [][]byte{{255, 0, 0, 255}, {0, 0, 255, 255}} {
		previous := Upload(img.Texture{RGBA: &img.RGBA{W: 1, H: 1, Pix: rgba}})
		gl.BindTextureUnit(0, previous.ID)
		gl.DepthMask(true)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		s.Begin()
		k.Begin(s, 1)
		k.SetPose(Identity(), short)
		k.Draw(mesh, nil, -1)
		gl.ReadPixels(16, 32, 1, 1, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&pixel[0]))
		if pixel != [4]byte{255, 255, 255, 255} {
			t.Fatal("missing skin texture borrowed preceding material", rgba, pixel)
		}
		gl.BindTextureUnit(0, previous.ID)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		s.Begin()
		static := NewMesh([]Vertex{{Pos: V3{-1, -1, .5}, Normal: V3{0, 1, 0}}, {Pos: V3{1, -1, .5}, Normal: V3{0, 1, 0}}, {Pos: V3{0, 1, .5}, Normal: V3{0, 1, 0}}}, []uint32{0, 1, 2})
		s.Ambient = 1
		s.Begin()
		s.DrawModel(static, nil, Identity(), -1, false)
		gl.ReadPixels(32, 32, 1, 1, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&pixel[0]))
		if pixel != [4]byte{255, 255, 255, 255} {
			t.Fatal("untextured world model borrowed preceding material", rgba, pixel)
		}
		static.Delete()
		previous.Delete()
	}
	if code := gl.GetError(); code != gl.NO_ERROR {
		t.Fatalf("skin palette GL error %#x", code)
	}
}
