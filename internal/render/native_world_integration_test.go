//go:build integration

package render

import (
	"bytes"
	img "github.com/Spelunkern/Shaiya-GX/internal/data/image"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/go-gl/gl/v4.5-core/gl"
	"os"
	"runtime"
	"testing"
)

func TestOriginalWorldTexturePolicyGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Title: "native world textures", Width: 64, Height: 64, Hidden: true})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
	if err = Init(true); err != nil {
		t.Fatal(err)
	}
	base := make([]byte, 32)
	for i := 0; i < 4; i++ {
		copy(base[i*8:], []byte{31, 0, 31, 0, 0, 0, 0, 0})
	}
	level1 := []byte{0, 248, 0, 248, 0, 0, 0, 0}
	source := img.Texture{DDS: &img.DDS{W: 8, H: 8, Format: img.FormatDXT1, Levels: [][]byte{base, level1}}}
	tex := UploadWorld(source)
	defer tex.Delete()
	var levels, filter int32
	gl.GetTextureParameteriv(tex.ID, gl.TEXTURE_IMMUTABLE_LEVELS, &levels)
	if levels != 4 {
		t.Fatalf("world levels=%d want4", levels)
	}
	gl.GetTextureParameteriv(tex.ID, gl.TEXTURE_MIN_FILTER, &filter)
	if filter != gl.LINEAR_MIPMAP_NEAREST {
		t.Fatal("native world mip selection changed", filter)
	}
	for i, want := range source.DDS.Levels {
		got := make([]byte, len(want))
		gl.GetCompressedTextureImage(tex.ID, int32(i), int32(len(got)), gl.Ptr(&got[0]))
		if !bytes.Equal(got, want) {
			t.Fatalf("overwrote authored level %d", i)
		}
	}
	rgba := make([]byte, 4)
	gl.GetTextureImage(tex.ID, 3, gl.RGBA, gl.UNSIGNED_BYTE, 4, gl.Ptr(&rgba[0]))
	if rgba[0] < 240 || rgba[2] > 16 {
		t.Fatalf("generated level did not descend from last authored red mip: %v", rgba)
	}
	plain := Upload(source)
	defer plain.Delete()
	gl.GetTextureParameteriv(plain.ID, gl.TEXTURE_IMMUTABLE_LEVELS, &levels)
	if levels != 2 {
		t.Fatal("changed non-world texture policy")
	}
	s, err := NewScene3D()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Delete()
	s.View, s.Proj = Identity(), Identity()
	s.Ambient = 1
	s.LightDir = V3{0, -1, 0}
	mesh := NewMesh([]Vertex{{Pos: V3{-1, -1, .5}, Normal: V3{0, 1, 0}}, {Pos: V3{1, -1, .5}, Normal: V3{0, 1, 0}}, {Pos: V3{0, 1, .5}, Normal: V3{0, 1, 0}}}, []uint32{0, 1, 2})
	defer mesh.Delete()
	for _, mani := range []bool{false, true} {
		threshold := byte(80)
		if mani {
			threshold = 8
		}
		for _, a := range []byte{threshold - 1, threshold, threshold + 1} {
			image := Upload(img.Texture{RGBA: &img.RGBA{W: 1, H: 1, Pix: []byte{255, 255, 255, a}}})
			gl.Viewport(0, 0, 64, 64)
			gl.DepthMask(true)
			gl.ClearDepth(1)
			gl.ClearColor(0, 0, 0, 1)
			gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
			s.Begin()
			s.DrawModel(mesh, image, Identity(), NativeWorldAlphaRef(mani), false)
			var p [4]byte
			gl.ReadPixels(32, 32, 1, 1, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&p[0]))
			image.Delete()
			if (p[0] > 0) != (a >= threshold) {
				t.Fatalf("mani=%v alpha=%d threshold=%d pixel=%v", mani, a, threshold, p)
			}
		}
	}
	if code := gl.GetError(); code != gl.NO_ERROR {
		t.Fatalf("native world GL error %#x", code)
	}
}
