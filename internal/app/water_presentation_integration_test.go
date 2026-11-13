//go:build integration

package app

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/archive"
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
	"github.com/Spelunkern/Shaiya-GX/internal/scene"
	"github.com/go-gl/gl/v4.5-core/gl"
)

func TestWaterLocomotionAndContactGL(t *testing.T) {
	if os.Getenv("SHAIYA_TEST_RENDER") != "1" {
		t.Skip("requires GL and assets")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	win, err := platform.Open(platform.Config{Width: 512, Height: 512, Hidden: true, Debug: true})
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
	if err := a.Preview("world-stream"); err != nil {
		t.Fatal(err)
	}
	v := a.wv
	found := false
	for x := float32(20); x < float32(v.world.WLD.MapSize)-20 && !found; x += 10 {
		for z := float32(20); z < float32(v.world.WLD.MapSize)-20; z += 10 {
			if floor, deep := v.world.WaterFloor(x, z); deep && floor < -6 && v.world.Height(x+5, z) < -6 {
				v.place(render.V3{x, scene.SwimSurfaceY, z})
				found = true
				break
			}
		}
	}
	if !found {
		t.Fatal("no deep water")
	}
	v.facing = render.V3{1, 0, 0}
	v.eye = v.pos.Add(render.V3{0, 2, -10})
	v.cameraReady = false
	v.sendT = 100 // This presentation fixture sends no network traffic.
	if !v.self.PlayAction(12, true) {
		t.Fatal("missing prior sitting pose")
	}
	a.updateWorldStep(&platform.Input{}, 1.0/60)
	if !v.self.PlayingClip("006_swnormal") {
		t.Fatal("water entry retained the prior pose")
	}
	if len(v.waterPresentation.effects) != 1 {
		t.Fatal("surface entry ring missing")
	}
	v.moving, v.motion = true, 1
	for range 20 {
		a.updateWorldStep(&platform.Input{Down: "W"}, 1.0/60)
	}
	if !v.self.PlayingClip("007_swim") {
		t.Fatal("moving swimmer did not animate")
	}
	if len(v.waterPresentation.effects) < 2 {
		t.Fatal("moving swimmer left no wake")
	}
	v.moving, v.motion = false, 0
	a.updateWorldStep(&platform.Input{}, 1.0/60)
	if !v.self.PlayingClip("006_swnormal") {
		t.Fatal("stationary swimmer did not float")
	}
	for range 4 {
		a.updateWaterPresentation(.1, false, true)
	}
	a.frameDT = 0
	capture := func() []byte {
		gl.Viewport(0, 0, 512, 512)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		a.drawWorldView(512, 512)
		pixels := make([]byte, 512*512*4)
		gl.ReadPixels(0, 0, 512, 512, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&pixels[0]))
		return pixels
	}
	for range 4 {
		capture()
	}
	with := capture()
	if dir := os.Getenv("SHAIYA_TEST_ARTIFACTS"); dir != "" {
		if err := render.SavePNG(filepath.Join(dir, "swimming-contact.png"), 512, 512); err != nil {
			t.Fatal(err)
		}
	}
	effects := v.waterPresentation.effects
	v.waterPresentation.effects = nil
	without := capture()
	if bytes.Equal(with, without) {
		t.Fatal("authored water rings produced no visible pixels")
	}
	v.waterPresentation.effects = effects
	v.pos[1] = -4
	a.updateWaterPresentation(.1, false, true)
	if v.waterPresentation.touching {
		t.Fatal("deep diver emitted a surface ring")
	}
	for range 100 {
		a.updateWaterPresentation(.1, false, true)
	}
	if len(v.waterPresentation.effects) != 0 {
		t.Fatal("water particles survived their lifetime")
	}
	if err := gl.GetError(); err != gl.NO_ERROR {
		t.Fatalf("water GL error: %#x", err)
	}
	t.Log("real map update: prior pose → float → swim → float; authored wake visible and expires underwater")
}
