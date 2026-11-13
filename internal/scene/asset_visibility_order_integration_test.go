//go:build integration

package scene

import (
	img "github.com/Spelunkern/Shaiya-GX/internal/data/image"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/go-gl/gl/v4.5-core/gl"
	"os"
	"runtime"
	"testing"
)

// Culling a remote copy of A must not reverse the winner between coincident
// copies of A and B that remain visible. Equal depth cannot resolve ties.
func TestAssetVisibilityKeepsCoincidentMaterialsStableGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Title: "asset order", Width: 64, Height: 64, Hidden: true})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
	if err = render.Init(true); err != nil {
		t.Fatal(err)
	}
	r, err := render.NewScene3D()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Delete()
	r.Proj = render.Identity()
	r.Ambient = 1
	r.LightDir = render.V3{0, -1, 0}
	mesh := render.NewMesh([]render.Vertex{{Pos: render.V3{-.25, -.25, .5}, Normal: render.V3{0, 1, 0}}, {Pos: render.V3{.25, -.25, .5}, Normal: render.V3{0, 1, 0}}, {Pos: render.V3{.25, .25, .5}, Normal: render.V3{0, 1, 0}}, {Pos: render.V3{-.25, .25, .5}, Normal: render.V3{0, 1, 0}}}, []uint32{0, 1, 2, 0, 2, 3})
	defer mesh.Delete()
	red := render.Upload(img.Texture{RGBA: &img.RGBA{W: 1, H: 1, Pix: []byte{255, 0, 0, 255}}})
	defer red.Delete()
	blue := render.Upload(img.Texture{RGBA: &img.RGBA{W: 1, H: 1, Pix: []byte{0, 0, 255, 255}}})
	defer blue.Delete()
	a := &Model{center: render.V3{0, 0, .5}, radius: .4, Parts: []Part{{Mesh: mesh, Tex: red}}}
	b := &Model{center: render.V3{0, 0, .5}, radius: .4, Parts: []Part{{Mesh: mesh, Tex: blue}}}
	distant := render.Identity()
	distant[12] = -1.3
	stream := fieldStream{world: &World{}, assets: &Assets{models: map[string]*Model{"a": a, "b": b}}, drawObjects: []int{0, 1, 2}, objects: []cellObject{{file: "a", world: distant, scale: 1}, {file: "b", world: render.Identity(), scale: 1}, {file: "a", world: render.Identity(), scale: 1}}}
	stream.prepareDrawEntries()
	var pixels [2][4]byte
	for i, x := range []float32{.2, -.2} {
		gl.Viewport(0, 0, 64, 64)
		gl.DepthMask(true)
		gl.ClearDepth(1)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		r.View = render.Identity()
		r.View[12] = x
		r.Begin()
		stream.draw(r)
		gl.ReadPixels(32, 32, 1, 1, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&pixels[i][0]))
	}
	if pixels[0] != pixels[1] {
		t.Fatalf("unrelated instance visibility changed coincident material: %v -> %v", pixels[0], pixels[1])
	}
	// The same model can occur in both native passes. Its MAni copy must use
	// >=8 while the ordinary copy uses >=80, without sharing one batch state.
	semi := render.Upload(img.Texture{RGBA: &img.RGBA{W: 1, H: 1, Pix: []byte{255, 255, 255, 40}}})
	defer semi.Delete()
	a.Parts[0].Tex = semi
	left, right := render.Identity(), render.Identity()
	left[12] = -.5
	right[12] = .5
	stream.objects = []cellObject{{file: "a", world: left, scale: 1, mani: true}, {file: "a", world: right, scale: 1}}
	stream.drawObjects = []int{0, 1}
	stream.prepareDrawEntries()
	gl.ClearColor(0, 0, 0, 1)
	gl.DepthMask(true)
	gl.ClearDepth(1)
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	r.View = render.Identity()
	r.Begin()
	stream.draw(r)
	var l, h [4]byte
	gl.ReadPixels(16, 32, 1, 1, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&l[0]))
	gl.ReadPixels(48, 32, 1, 1, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&h[0]))
	if l[0] != 255 || h[0] != 0 {
		t.Fatalf("native passes merged: MAni=%v ordinary=%v", l, h)
	}

}
