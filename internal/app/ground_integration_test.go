//go:build integration

package app

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"unsafe"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/game"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/proto"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/go-gl/gl/v4.5-core/gl"
)

func TestGroundModelsOriginalAssetsGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL and original assets")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Title: "ground items", Width: 512, Height: 512, Hidden: true, Debug: true})
	if err != nil {
		t.Fatal(err)
	}
	defer win.Close()
	if err := render.Init(true); err != nil {
		t.Fatal(err)
	}
	arc, err := archive.Open(os.Getenv("SHAIYA_DATA"))
	if err != nil {
		t.Fatal(err)
	}
	defer arc.Close()
	a, err := New(Config{}, arc)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	a.state, a.wv = game.New(), &worldView{}
	for i, typ := range []uint8{1, 22, 25, 26} {
		for n := 0; n < 4; n++ {
			id := uint32(i*4 + n + 1)
			a.state.GroundItems[id] = proto.GroundItem{ID: id, Type: typ, X: float32(i)*1.5 - 2.25, Z: float32(n) * 1.2}
		}
	}
	s := a.scene3d
	s.View = render.LookAtLH(render.V3{0, 5, -8}, render.V3{0, 0, 1.8}, render.V3{0, 1, 0})
	s.Proj = render.PerspectiveFovLH(.8, 1, .1, 100)
	gl.Viewport(0, 0, 512, 512)
	gl.ClearColor(.05, .05, .05, 1)
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	s.Begin()
	render.ResetDrawCalls()
	a.drawGroundItems()
	s.End()
	if calls := render.DrawCalls(); calls != 4 {
		t.Fatalf("16 drops should need four draws, got %d", calls)
	}
	for kind := 0; kind < 4; kind++ {
		if m := a.assets.GroundModel(kind); m == nil || m.Part.Mesh == nil || m.Part.Tex == nil || m.Radius <= 0 {
			t.Fatalf("original model %d did not load: %+v", kind, m)
		}
	}
	pixels := make([]byte, 512*512*4)
	gl.ReadPixels(0, 0, 512, 512, gl.RGBA, gl.UNSIGNED_BYTE, unsafe.Pointer(&pixels[0]))
	changed := 0
	for i := 0; i < len(pixels); i += 4 {
		if pixels[i] > 25 || pixels[i+1] > 25 || pixels[i+2] > 25 {
			changed++
		}
	}
	if changed < 100 {
		t.Fatalf("ground models did not produce visible pixels: %d", changed)
	}
	// Advancing the shared clock must brighten the same geometry without
	// increasing draw calls; the original multiplies each texture by a factor.
	var dim uint64
	for i := 0; i < len(pixels); i += 4 {
		dim += uint64(pixels[i]) + uint64(pixels[i+1]) + uint64(pixels[i+2])
	}
	a.clock = .35
	gl.DepthMask(true)
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	s.Begin()
	render.ResetDrawCalls()
	a.drawGroundItems()
	s.End()
	gl.ReadPixels(0, 0, 512, 512, gl.RGBA, gl.UNSIGNED_BYTE, unsafe.Pointer(&pixels[0]))
	var bright uint64
	for i := 0; i < len(pixels); i += 4 {
		bright += uint64(pixels[i]) + uint64(pixels[i+1]) + uint64(pixels[i+2])
	}
	if bright <= dim || render.DrawCalls() != 4 {
		t.Fatalf("ground pulse: dim %d bright %d draws %d", dim, bright, render.DrawCalls())
	}
	a.sprites.Begin(512, 512)
	a.drawPlayerNames(512, 512)
	a.sprites.End()
	var picked *platform.Input
	var pickedID uint32
	for y := 160; y < 390 && picked == nil; y++ {
		for x := 100; x < 420; x++ {
			click := &platform.Input{Clicked: true, MouseX: x, MouseY: y}
			if id := a.pickGroundModel(click, 512, 512); id != 0 && a.pickGroundLabel(click, 512, 512) == 0 {
				picked, pickedID = click, id
				break
			}
		}
	}
	if picked == nil {
		t.Fatal("could not click the visible mesh outside its label")
	}
	// A full-depth occluder makes the exact same click fail.
	gl.DepthMask(true)
	gl.ClearDepth(0)
	gl.Clear(gl.DEPTH_BUFFER_BIT)
	a.sprites.Begin(512, 512)
	a.drawPlayerNames(512, 512)
	a.sprites.End()
	if got := a.pickGroundModel(picked, 512, 512); got != 0 {
		t.Fatalf("occluded object %d remained clickable as %d", pickedID, got)
	}
	gl.ClearDepth(1)
	if dir := os.Getenv("SHAIYA_TEST_ARTIFACTS"); dir != "" {
		if err := render.SavePNG(filepath.Join(dir, "ground-models-preview.png"), 512, 512); err != nil {
			t.Fatal(err)
		}
	}
	if code := gl.GetError(); code != gl.NO_ERROR {
		t.Fatalf("ground models GL error %#x", code)
	}
}
